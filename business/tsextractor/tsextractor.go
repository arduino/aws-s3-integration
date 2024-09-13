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
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"crypto/rand"
	"math/big"

	"github.com/arduino/aws-s3-integration/internal/csv"
	"github.com/arduino/aws-s3-integration/internal/iot"
	"github.com/arduino/aws-s3-integration/internal/s3"
	iotclient "github.com/arduino/iot-client-go/v2"
	"github.com/sirupsen/logrus"
)

const importConcurrency = 10
const retryCount = 5

type TsExtractor struct {
	iotcl  *iot.Client
	logger *logrus.Entry
}

func New(iotcl *iot.Client, logger *logrus.Entry) *TsExtractor {
	return &TsExtractor{iotcl: iotcl, logger: logger}
}

func computeTimeAlignment(resolutionSeconds, timeWindowInMinutes int) (time.Time, time.Time) {
	// Compute time alignment
	if resolutionSeconds <= 60 {
		resolutionSeconds = 300 // Align to 5 minutes
	}
	to := time.Now().Truncate(time.Duration(resolutionSeconds) * time.Second).UTC()
	from := to.Add(-time.Duration(timeWindowInMinutes) * time.Minute)
	return from, to
}

func (a *TsExtractor) ExportTSToS3(
	ctx context.Context,
	timeWindowInMinutes int,
	thingsMap map[string]iotclient.ArduinoThing,
	resolution int,
	destinationS3Bucket string) error {

	// Truncate time to given resolution
	from, to := computeTimeAlignment(resolution, timeWindowInMinutes)

	// Open s3 output writer
	s3cl, err := s3.NewS3Client(destinationS3Bucket)
	if err != nil {
		return err
	}

	// Open csv output writer
	writer, err := csv.NewWriter(from, a.logger)
	if err != nil {
		return err
	}

	var wg sync.WaitGroup
	tokens := make(chan struct{}, importConcurrency)

	a.logger.Infoln("=====> Exporting data. Time window: ", timeWindowInMinutes, "m (resolution: ", resolution, "s). From ", from, " to ", to)
	for thingID, thing := range thingsMap {

		if thing.Properties == nil || len(thing.Properties) == 0 {
			a.logger.Warn("Skipping thing with no properties: ", thingID)
			continue
		}

		tokens <- struct{}{}
		wg.Add(1)

		go func(thingID string, thing iotclient.ArduinoThing, writer *csv.CsvWriter) {
			defer func() { <-tokens }()
			defer wg.Done()

			if resolution <= 0 {
				// Populate raw time series data
				err := a.populateRawTSDataIntoS3(ctx, from, to, thingID, thing, writer)
				if err != nil {
					a.logger.Error("Error populating raw time series data: ", err)
					return
				}
			} else {
				// Populate numeric time series data
				err := a.populateNumericTSDataIntoS3(ctx, from, to, thingID, thing, resolution, writer)
				if err != nil {
					a.logger.Error("Error populating time series data: ", err)
					return
				}

				// Populate string time series data, if any
				err = a.populateStringTSDataIntoS3(ctx, from, to, thingID, thing, resolution, writer)
				if err != nil {
					a.logger.Error("Error populating string time series data: ", err)
					return
				}
			}
		}(thingID, thing, writer)
	}

	// Wait for all routines termination
	a.logger.Infoln("Waiting for all data extraction jobs to terminate...")
	wg.Wait()

	// Close csv output writer and upload to s3
	writer.Close()
	defer writer.Delete()

	var formattedFrom string
	if resolution == 3600 {
		formattedFrom = from.Format("2006-01-02-15-00")
	} else if resolution > 3600 {
		formattedFrom = from.Format("2006-01-02-00-00")
	} else {
		formattedFrom = from.Format("2006-01-02-15-04")
	}
	destinationKey := fmt.Sprintf("%s/%s.csv", from.Format("2006-01-02"), formattedFrom)
	a.logger.Infof("Uploading file %s to bucket %s\n", destinationKey, s3cl.DestinationBucket())
	if err := s3cl.WriteFile(ctx, destinationKey, writer.GetFilePath()); err != nil {
		return err
	}

	return nil
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
	thingID string,
	thing iotclient.ArduinoThing,
	resolution int,
	writer *csv.CsvWriter) error {

	if resolution <= 60 {
		resolution = 60
	}

	var batched *iotclient.ArduinoSeriesBatch
	var err error
	var retry bool
	for i := 0; i < retryCount; i++ {
		batched, retry, err = a.iotcl.GetTimeSeriesByThing(ctx, thingID, from, to, int64(resolution))
		if !retry {
			break
		} else {
			// This is due to a rate limit on the IoT API, we need to wait a bit before retrying
			a.logger.Warnf("Rate limit reached for thing %s. Waiting 1 second before retrying.\n", thingID)
			randomRateLimitingSleep()
		}
	}
	if err != nil {
		return err
	}

	sampleCount := int64(0)
	samples := [][]string{}
	for _, response := range batched.Responses {
		if response.CountValues == 0 {
			continue
		}

		propertyID := strings.Replace(response.Query, "property.", "", 1)
		a.logger.Debugf("Thing %s - Property %s - %d values\n", thingID, propertyID, response.CountValues)
		sampleCount += response.CountValues

		propertyName, propertyType := extractPropertyNameAndType(thing, propertyID)

		for i := 0; i < len(response.Times); i++ {

			ts := response.Times[i]
			value := response.Values[i]
			samples = append(samples, composeRow(ts, thingID, thing.Name, propertyID, propertyName, propertyType, strconv.FormatFloat(value, 'f', -1, 64)))
		}
	}

	// Write samples to csv ouput file
	if len(samples) > 0 {
		if err := writer.Write(samples); err != nil {
			return err
		}
		a.logger.Debugf("Thing %s [%s] saved %d values\n", thingID, thing.Name, sampleCount)
	}

	return nil
}

func composeRow(ts time.Time, thingID string, thingName string, propertyID string, propertyName string, propertyType string, value string) []string {
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
	return ptype == "CHARSTRING"
}

func (a *TsExtractor) populateStringTSDataIntoS3(
	ctx context.Context,
	from time.Time,
	to time.Time,
	thingID string,
	thing iotclient.ArduinoThing,
	resolution int,
	writer *csv.CsvWriter) error {

	// Filter properties by char type
	stringProperties := []string{}
	for _, prop := range thing.Properties {
		if isStringProperty(prop.Type) {
			stringProperties = append(stringProperties, prop.Id)
		}
	}

	if len(stringProperties) == 0 {
		return nil
	}

	var batched *iotclient.ArduinoSeriesBatchSampled
	var err error
	var retry bool
	for i := 0; i < retryCount; i++ {
		batched, retry, err = a.iotcl.GetTimeSeriesStringSampling(ctx, stringProperties, from, to, int32(resolution))
		if !retry {
			break
		} else {
			// This is due to a rate limit on the IoT API, we need to wait a bit before retrying
			a.logger.Warnf("Rate limit reached for thing %s. Waiting 1 second before retrying.\n", thingID)
			randomRateLimitingSleep()
		}
	}
	if err != nil {
		return err
	}

	sampleCount := int64(0)
	samples := [][]string{}
	for _, response := range batched.Responses {
		if response.CountValues == 0 {
			continue
		}

		propertyID := strings.Replace(response.Query, "property.", "", 1)
		a.logger.Debugf("Thing %s - String Property %s - %d values\n", thingID, propertyID, response.CountValues)
		sampleCount += response.CountValues

		propertyName, propertyType := extractPropertyNameAndType(thing, propertyID)

		for i := 0; i < len(response.Times); i++ {

			ts := response.Times[i]
			value := response.Values[i]
			if value == nil {
				continue
			}
			samples = append(samples, composeRow(ts, thingID, thing.Name, propertyID, propertyName, propertyType, interfaceToString(value)))
		}
	}

	// Write samples to csv ouput file
	if len(samples) > 0 {
		if err := writer.Write(samples); err != nil {
			return err
		}
		a.logger.Debugf("Thing %s [%s] string properties saved %d values\n", thingID, thing.Name, sampleCount)
	}

	return nil
}

func (a *TsExtractor) populateRawTSDataIntoS3(
	ctx context.Context,
	from time.Time,
	to time.Time,
	thingID string,
	thing iotclient.ArduinoThing,
	writer *csv.CsvWriter) error {

	var batched *iotclient.ArduinoSeriesRawBatch
	var err error
	var retry bool
	for i := 0; i < retryCount; i++ {
		batched, retry, err = a.iotcl.GetRawTimeSeriesByThing(ctx, thingID, from, to)
		if !retry {
			break
		} else {
			// This is due to a rate limit on the IoT API, we need to wait a bit before retrying
			a.logger.Warnf("Rate limit reached for thing %s. Waiting 1 second before retrying.\n", thingID)
			randomRateLimitingSleep()
		}
	}
	if err != nil {
		return err
	}

	sampleCount := int64(0)
	samples := [][]string{}
	for _, response := range batched.Responses {
		if response.CountValues == 0 {
			continue
		}

		propertyID := strings.Replace(response.Query, "property.", "", 1)
		a.logger.Debugf("Thing %s - Query %s Property %s - %d values\n", thingID, response.Query, propertyID, response.CountValues)
		sampleCount += response.CountValues

		propertyName, propertyType := extractPropertyNameAndType(thing, propertyID)

		for i := 0; i < len(response.Times); i++ {

			ts := response.Times[i]
			value := response.Values[i]
			if value == nil {
				continue
			}
			samples = append(samples, composeRow(ts, thingID, thing.Name, propertyID, propertyName, propertyType, interfaceToString(value)))
		}
	}

	// Write samples to csv ouput file
	if len(samples) > 0 {
		if err := writer.Write(samples); err != nil {
			return err
		}
		a.logger.Debugf("Thing %s [%s] raw data saved %d values\n", thingID, thing.Name, sampleCount)
	}

	return nil
}

func interfaceToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return strconv.Itoa(v)
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(v)
	default:
		return fmt.Sprintf("%v", v)
	}
}
