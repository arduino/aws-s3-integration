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

	"github.com/arduino/aws-s3-integration/internal/csv"
	"github.com/arduino/aws-s3-integration/internal/iot"
	"github.com/arduino/aws-s3-integration/internal/s3"
	iotclient "github.com/arduino/iot-client-go/v2"
	"github.com/sirupsen/logrus"
)

const importConcurrency = 10

type TsExtractor struct {
	iotcl  *iot.Client
	logger *logrus.Entry
}

func New(iotcl *iot.Client, logger *logrus.Entry) *TsExtractor {
	return &TsExtractor{iotcl: iotcl, logger: logger}
}

func (a *TsExtractor) ExportTSToS3(
	ctx context.Context,
	timeWindowInMinutes int,
	thingsMap map[string]iotclient.ArduinoThing,
	resolution int,
	destinationS3Bucket string) error {

	to := time.Now().Truncate(time.Hour).UTC()
	from := to.Add(-time.Duration(timeWindowInMinutes) * time.Minute)

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

	a.logger.Infoln("=====> Export perf data - time window: ", timeWindowInMinutes, " minutes")
	for thingID, thing := range thingsMap {

		if thing.Properties == nil || len(thing.Properties) == 0 {
			a.logger.Warn("Skipping thing with no properties: ", thingID)
			continue
		}

		wg.Add(1)
		tokens <- struct{}{}

		go func(thingID string, thing iotclient.ArduinoThing, writer *csv.CsvWriter) {
			defer func() { <-tokens }()
			defer wg.Done()

			err := a.populateTSDataIntoS3(ctx, from, to, thingID, thing, resolution, writer)
			if err != nil {
				a.logger.Error("Error populating time series data: ", err)
				return
			}
		}(thingID, thing, writer)
	}

	// Wait for all routines termination
	wg.Wait()

	// Close csv output writer and upload to s3
	writer.Close()
	defer writer.Delete()

	destinationKey := fmt.Sprintf("%s/%s.csv", from.Format("2006-01-02"), from.Format("2006-01-02-15"))
	if err := s3cl.WriteFile(ctx, destinationKey, writer.GetFilePath()); err != nil {
		return err
	}

	return nil
}

func (a *TsExtractor) populateTSDataIntoS3(
	ctx context.Context,
	from time.Time,
	to time.Time,
	thingID string,
	thing iotclient.ArduinoThing,
	resolution int,
	writer *csv.CsvWriter) error {

	var batched *iotclient.ArduinoSeriesBatch
	var err error
	var retry bool
	for i := 0; i < 3; i++ {
		batched, retry, err = a.iotcl.GetTimeSeriesByThing(ctx, thingID, from, to, int64(resolution))
		if !retry {
			break
		} else {
			// This is due to a rate limit on the IoT API, we need to wait a bit before retrying
			a.logger.Infof("Rate limit reached for thing %s. Waiting 1 second before retrying.\n", thingID)
			time.Sleep(1 * time.Second)
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

		propertyName := extractPropertyName(thing, propertyID)

		for i := 0; i < len(response.Times); i++ {

			ts := response.Times[i]
			value := response.Values[i]
			row := make([]string, 6)
			row[0] = ts.UTC().Format(time.RFC3339)
			row[1] = thingID
			row[2] = thing.Name
			row[3] = propertyID
			row[4] = propertyName
			row[5] = strconv.FormatFloat(value, 'f', 3, 64)

			samples = append(samples, row)
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

func extractPropertyName(thing iotclient.ArduinoThing, propertyID string) string {
	propertyName := ""
	for _, prop := range thing.Properties {
		if prop.Id == propertyID {
			propertyName = prop.Name
			break
		}
	}
	return propertyName
}
