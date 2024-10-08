// This file is part of arduino aws-s3-integration.
//
// Copyright 2024 ARDUINO SA (http://www.arduino.cc/)
//
// This software is released under the Mozilla Public License Version 2.0,
// which covers the main part of aws-s3-integration.
// The terms of this license can be found at:
// https://www.mozilla.org/media/MPL/2.0/index.815ca599c9df.txt
//
// You can be released from the requirements of the above licenses by purchasing
// a commercial license. Buying such a license is mandatory if you want to
// modify or otherwise use the software for commercial activities involving the
// Arduino software without disclosing the source code of your own applications.
// To purchase a commercial license, send an email to license@arduino.cc.

package tsextractor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"crypto/rand"
	"math/big"

	"github.com/arduino/aws-s3-integration/internal/csv"
	"github.com/arduino/aws-s3-integration/internal/iot"
	iotclient "github.com/arduino/iot-client-go/v2"
	"github.com/sirupsen/logrus"
)

const importConcurrency = 10
const retryCount = 5

type TsExtractor struct {
	iotcl  iot.API
	logger *logrus.Entry
}

func New(iotcl iot.API, logger *logrus.Entry) *TsExtractor {
	return &TsExtractor{iotcl: iotcl, logger: logger}
}

func computeTimeAlignment(resolutionSeconds, timeWindowInMinutes int, enableAlignTimeWindow bool) (time.Time, time.Time) {
	// Compute time alignment
	if resolutionSeconds <= 60 {
		resolutionSeconds = 300 // Align to 5 minutes
	}

	timeAlignmentSeconds := resolutionSeconds
	if enableAlignTimeWindow {
		timeAlignmentSeconds = timeWindowInMinutes * 60
	}
	to := time.Now().Truncate(time.Duration(timeAlignmentSeconds) * time.Second).UTC()
	if !enableAlignTimeWindow && resolutionSeconds <= 900 {
		// Shift time window to avoid missing data
		to = to.Add(-time.Duration(300) * time.Second)
	}
	from := to.Add(-time.Duration(timeWindowInMinutes) * time.Minute)
	return from, to
}

func isRawResolution(resolution int) bool {
	return resolution <= 0
}

func (a *TsExtractor) ExportTSToFile(
	ctx context.Context,
	timeWindowInMinutes int,
	thingsMap map[string]iotclient.ArduinoThing,
	resolution int,
	aggregationStat string,
	enableAlignTimeWindow bool) (*csv.CsvWriter, time.Time, error) {

	// Truncate time to given resolution
	from, to := computeTimeAlignment(resolution, timeWindowInMinutes, enableAlignTimeWindow)

	// Open csv output writer
	writer, err := csv.NewWriter(from, a.logger, isRawResolution(resolution))
	if err != nil {
		return nil, from, err
	}

	var wg sync.WaitGroup
	tokens := make(chan struct{}, importConcurrency)
	errorChannel := make(chan error, len(thingsMap))

	if isRawResolution(resolution) {
		a.logger.Infoln("=====> Exporting data. Time window: ", timeWindowInMinutes, "m (resolution: ", resolution, "s). From ", from, " to ", to, " - aggregation: raw")
	} else {
		a.logger.Infoln("=====> Exporting data. Time window: ", timeWindowInMinutes, "m (resolution: ", resolution, "s). From ", from, " to ", to, " - aggregation: ", aggregationStat)
	}
	for _, thing := range thingsMap {

		if len(thing.Properties) == 0 {
			a.logger.Warn("Skipping thing with no properties: ", thing.Id)
			continue
		}

		tokens <- struct{}{}
		wg.Add(1)

		go func(thing iotclient.ArduinoThing, writer *csv.CsvWriter) {
			defer func() { <-tokens }()
			defer wg.Done()

			detectedProperties := []string{}
			isRaw := isRawResolution(resolution)
			if isRaw {
				// Populate raw time series data
				populatedProperties, err := a.populateRawTSDataIntoS3(ctx, from, to, thing, writer)
				if err != nil {
					a.logger.Error("Error populating raw time series data: ", err)
					errorChannel <- err
					return
				}
				if len(populatedProperties) > 0 {
					detectedProperties = append(detectedProperties, populatedProperties...)
				}
			} else {
				// Populate numeric time series data
				populatedProperties, err := a.populateNumericTSDataIntoS3(ctx, from, to, thing, resolution, aggregationStat, writer)
				if err != nil {
					a.logger.Error("Error populating time series data: ", err)
					errorChannel <- err
					return
				}
				if len(populatedProperties) > 0 {
					detectedProperties = append(detectedProperties, populatedProperties...)
				}

				// Populate string time series data, if any
				populatedProperties, err = a.populateStringTSDataIntoS3(ctx, from, to, thing, resolution, writer)
				if err != nil {
					a.logger.Error("Error populating string time series data: ", err)
					errorChannel <- err
					return
				}
				if len(populatedProperties) > 0 {
					detectedProperties = append(detectedProperties, populatedProperties...)
				}
			}

			// Populate last value samples for ON_CHANGE properties, if needed
			err = a.populateLastValueSamplesForOnChangeProperties(isRaw, thing, detectedProperties, writer)
			if err != nil {
				a.logger.Error("Error populating last value data: ", err)
				errorChannel <- err
				return
			}

		}(thing, writer)
	}

	// Wait for all routines termination
	a.logger.Infoln("Waiting for all data extraction jobs to terminate...")
	wg.Wait()
	close(errorChannel)

	// Check if there were errors
	detectedErrors := false
	for err := range errorChannel {
		if err != nil {
			a.logger.Error(err)
		}
	}
	if detectedErrors {
		return writer, from, errors.New("errors detected during data export")
	}

	return writer, from, nil
}

func randomRateLimitingSleep() {
	// Random sleep to avoid rate limiting (1s + random(0-500ms))
	n, err := rand.Int(rand.Reader, big.NewInt(500))
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	randomSleep := n.Int64() + 1000
	time.Sleep(time.Duration(randomSleep) * time.Millisecond)
}

func (a *TsExtractor) populateNumericTSDataIntoS3(
	ctx context.Context,
	from time.Time,
	to time.Time,
	thing iotclient.ArduinoThing,
	resolution int,
	aggregationStat string,
	writer *csv.CsvWriter) ([]string, error) {

	if resolution <= 60 {
		resolution = 60
	}

	populatedProperties := []string{}
	var batched *iotclient.ArduinoSeriesBatch
	var err error
	var retry bool
	for i := 0; i < retryCount; i++ {
		batched, retry, err = a.iotcl.GetTimeSeriesByThing(ctx, thing.Id, from, to, int64(resolution), aggregationStat)
		if !retry {
			break
		} else {
			// This is due to a rate limit on the IoT API, we need to wait a bit before retrying
			a.logger.Warnf("Rate limit reached for thing %s. Waiting 1 second before retrying.\n", thing.Id)
			randomRateLimitingSleep()
		}
	}
	if err != nil {
		return nil, err
	}

	sampleCount := int64(0)
	samples := [][]string{}
	for _, response := range batched.Responses {
		if response.CountValues == 0 {
			continue
		}

		propertyID := strings.Replace(response.Query, "property.", "", 1)
		a.logger.Debugf("Thing %s - Property %s - %d values\n", thing.Id, propertyID, response.CountValues)
		sampleCount += response.CountValues

		propertyName, propertyType := extractPropertyNameAndType(thing, propertyID)

		for i := 0; i < len(response.Times); i++ {

			ts := response.Times[i]
			value := response.Values[i]
			if !slices.Contains(populatedProperties, propertyID) {
				populatedProperties = append(populatedProperties, propertyID)
			}
			samples = append(samples, composeRow(ts, thing.Id, thing.Name, propertyID, propertyName, propertyType, strconv.FormatFloat(value, 'f', -1, 64), aggregationStat))
		}
	}

	// Write samples to csv ouput file
	if len(samples) > 0 {
		if err := writer.Write(samples); err != nil {
			return nil, err
		}
		a.logger.Debugf("Thing %s [%s] saved %d values\n", thing.Id, thing.Name, sampleCount)
	}

	return populatedProperties, nil
}

func composeRow(ts time.Time, thingID string, thingName string, propertyID string, propertyName string, propertyType string, value string, aggregation string) []string {
	row := make([]string, 8)
	row[0] = ts.UTC().Format(time.RFC3339)
	row[1] = thingID
	row[2] = thingName
	row[3] = propertyID
	row[4] = propertyName
	row[5] = propertyType
	row[6] = value
	row[7] = aggregation
	return row
}

func composeRawRow(ts time.Time, thingID string, thingName string, propertyID string, propertyName string, propertyType string, value string) []string {
	row := make([]string, 7)
	row[0] = ts.UTC().Format(time.RFC3339)
	row[1] = thingID
	row[2] = thingName
	row[3] = propertyID
	row[4] = propertyName
	row[5] = propertyType
	row[6] = value
	return row
}

func extractPropertyNameAndType(thing iotclient.ArduinoThing, propertyID string) (string, string) {
	propertyName := ""
	propertyType := ""
	for _, prop := range thing.Properties {
		if prop.Id == propertyID {
			propertyName = prop.Name
			propertyType = prop.Type
			break
		}
	}
	if propertyType == "STATUS" {
		propertyType = "BOOLEAN"
	}
	return propertyName, propertyType
}

func isStringProperty(ptype string) bool {
	return iot.IsPropertyString(ptype) || iot.IsPropertyLocation(ptype)
}

func (a *TsExtractor) populateStringTSDataIntoS3(
	ctx context.Context,
	from time.Time,
	to time.Time,
	thing iotclient.ArduinoThing,
	resolution int,
	writer *csv.CsvWriter) ([]string, error) {

	// Filter properties by char type
	stringProperties := []string{}
	for _, prop := range thing.Properties {
		if isStringProperty(prop.Type) {
			stringProperties = append(stringProperties, prop.Id)
		}
	}

	if len(stringProperties) == 0 {
		return nil, nil
	}

	populatedProperties := []string{}
	var batched *iotclient.ArduinoSeriesBatchSampled
	var err error
	var retry bool
	for i := 0; i < retryCount; i++ {
		batched, retry, err = a.iotcl.GetTimeSeriesStringSampling(ctx, stringProperties, from, to, int32(resolution))
		if !retry {
			break
		} else {
			// This is due to a rate limit on the IoT API, we need to wait a bit before retrying
			a.logger.Warnf("Rate limit reached for thing %s. Waiting 1 second before retrying.\n", thing.Id)
			randomRateLimitingSleep()
		}
	}
	if err != nil {
		return nil, err
	}

	sampleCount := int64(0)
	samples := [][]string{}
	for _, response := range batched.Responses {
		if response.CountValues == 0 {
			continue
		}

		propertyID := strings.Replace(response.Query, "property.", "", 1)
		a.logger.Debugf("Thing %s - String Property %s - %d values\n", thing.Id, propertyID, response.CountValues)
		sampleCount += response.CountValues

		propertyName, propertyType := extractPropertyNameAndType(thing, propertyID)

		for i := 0; i < len(response.Times); i++ {

			ts := response.Times[i]
			value := response.Values[i]
			if value == nil {
				continue
			}
			if !slices.Contains(populatedProperties, propertyID) {
				populatedProperties = append(populatedProperties, propertyID)
			}
			samples = append(samples, composeRow(ts, thing.Id, thing.Name, propertyID, propertyName, propertyType, a.interfaceToString(value), "SAMPLED"))
		}
	}

	// Write samples to csv ouput file
	if len(samples) > 0 {
		if err := writer.Write(samples); err != nil {
			return nil, err
		}
		a.logger.Debugf("Thing %s [%s] string properties saved %d values\n", thing.Id, thing.Name, sampleCount)
	}

	return populatedProperties, nil
}

func (a *TsExtractor) populateRawTSDataIntoS3(
	ctx context.Context,
	from time.Time,
	to time.Time,
	thing iotclient.ArduinoThing,
	writer *csv.CsvWriter) ([]string, error) {

	populatedProperties := []string{}
	var batched *iotclient.ArduinoSeriesRawBatch
	var err error
	var retry bool
	for i := 0; i < retryCount; i++ {
		batched, retry, err = a.iotcl.GetRawTimeSeriesByThing(ctx, thing.Id, from, to)
		if !retry {
			break
		} else {
			// This is due to a rate limit on the IoT API, we need to wait a bit before retrying
			a.logger.Warnf("Rate limit reached for thing %s. Waiting 1 second before retrying.\n", thing.Id)
			randomRateLimitingSleep()
		}
	}
	if err != nil {
		return nil, err
	}

	sampleCount := int64(0)
	samples := [][]string{}
	for _, response := range batched.Responses {
		if response.CountValues == 0 {
			continue
		}

		propertyID := strings.Replace(response.Query, "property.", "", 1)
		a.logger.Debugf("Thing %s - Query %s Property %s - %d values\n", thing.Id, response.Query, propertyID, response.CountValues)
		sampleCount += response.CountValues

		propertyName, propertyType := extractPropertyNameAndType(thing, propertyID)

		for i := 0; i < len(response.Times); i++ {

			ts := response.Times[i]
			value := response.Values[i]
			if value == nil {
				continue
			}
			if !slices.Contains(populatedProperties, propertyID) {
				populatedProperties = append(populatedProperties, propertyID)
			}
			samples = append(samples, composeRawRow(ts, thing.Id, thing.Name, propertyID, propertyName, propertyType, a.interfaceToString(value)))
		}
	}

	// Write samples to csv ouput file
	if len(samples) > 0 {
		if err := writer.Write(samples); err != nil {
			return nil, err
		}
		a.logger.Debugf("Thing %s [%s] raw data saved %d values\n", thing.Id, thing.Name, sampleCount)
	}

	return populatedProperties, nil
}

func (a *TsExtractor) interfaceToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	case map[string]any:
		encoded, err := json.Marshal(v)
		if err != nil {
			a.logger.Error("Error encoding map to json: ", err)
			return fmt.Sprintf("%v", v)
		}
		return string(encoded)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func isLastValueAllowedProperty(prop iotclient.ArduinoProperty) bool {
	return prop.UpdateStrategy == "ON_CHANGE" && (isStringProperty(prop.Type) || iot.IsPropertyBool(prop.Type) || iot.IsPropertyNumberType(prop.Type))
}

func (a *TsExtractor) populateLastValueSamplesForOnChangeProperties(
	isRaw bool,
	thing iotclient.ArduinoThing,
	propertiesWithExtractedValue []string,
	writer *csv.CsvWriter) error {

	// Check if there are ON_CHANGE properties
	if len(thing.Properties) == 0 {
		return nil
	}
	samples := [][]string{}
	sampleCount := 0
	for _, prop := range thing.Properties {
		if isLastValueAllowedProperty(prop) && !slices.Contains(propertiesWithExtractedValue, prop.Id) {
			if prop.ValueUpdatedAt == nil {
				continue
			}
			propName, propType := extractPropertyNameAndType(thing, prop.Id)
			var toAdd []string
			if isRaw {
				toAdd = composeRawRow(*prop.ValueUpdatedAt, thing.Id, thing.Name, prop.Id, propName, propType, a.interfaceToString(prop.LastValue))
			} else {
				toAdd = composeRow(*prop.ValueUpdatedAt, thing.Id, thing.Name, prop.Id, propName, propType, a.interfaceToString(prop.LastValue), "LAST_VALUE")
			}
			samples = append(samples, toAdd)
			sampleCount++
		}
	}

	// Write samples to csv ouput file
	if len(samples) > 0 {
		if err := writer.Write(samples); err != nil {
			return err
		}
		a.logger.Debugf("Thing %s [%s] last value data saved %d values\n", thing.Id, thing.Name, sampleCount)
	}

	return nil
}
