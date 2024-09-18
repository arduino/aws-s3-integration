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

package main

import (
	"context"
	"errors"
	"os"

	"github.com/arduino/aws-s3-integration/app/exporter"
	"github.com/arduino/aws-s3-integration/internal/parameters"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/sirupsen/logrus"
)

type AWSS3ImportTrigger struct {
	Dev bool `json:"dev"`
}

const (
	GlobalArduinoPrefix = "/arduino/s3-importer"

	// Parameters for backward compatibility
	IoTApiKey           = GlobalArduinoPrefix + "/iot/api-key"
	IoTApiSecret        = GlobalArduinoPrefix + "/iot/api-secret"
	IoTApiOrgId         = GlobalArduinoPrefix + "/iot/org-id"
	IoTApiTags          = GlobalArduinoPrefix + "/iot/filter/tags"
	SamplesResoSec      = GlobalArduinoPrefix + "/iot/samples-resolution-seconds"
	SamplesReso         = GlobalArduinoPrefix + "/iot/samples-resolution"
	Scheduling          = GlobalArduinoPrefix + "/iot/scheduling"
	DestinationS3Bucket = GlobalArduinoPrefix + "/destination-bucket"

	// Per stack parameters
	PerStackArduinoPrefix    = "/arduino/s3-exporter/" + parameters.StackName
	IoTApiKeyStack           = PerStackArduinoPrefix + "/iot/api-key"
	IoTApiSecretStack        = PerStackArduinoPrefix + "/iot/api-secret"
	IoTApiOrgIdStack         = PerStackArduinoPrefix + "/iot/org-id"
	IoTApiTagsStack          = PerStackArduinoPrefix + "/iot/filter/tags"
	SamplesResoStack         = PerStackArduinoPrefix + "/iot/samples-resolution"
	SchedulingStack          = PerStackArduinoPrefix + "/iot/scheduling"
	DestinationS3BucketStack = PerStackArduinoPrefix + "/destination-bucket"
	AggregationStatStack     = PerStackArduinoPrefix + "/iot/aggregation-statistic"
	AlignWithTimeWindowStack = PerStackArduinoPrefix + "/iot/align_with_time_window"
	EnableCompressionStack   = PerStackArduinoPrefix + "/enable_compression"

	SamplesResolutionSeconds           = 300
	DefaultTimeExtractionWindowMinutes = 60
)

func HandleRequest(ctx context.Context, event *AWSS3ImportTrigger) (*string, error) {

	logger := logrus.NewEntry(logrus.New())

	stackName := os.Getenv("STACK_NAME")

	var apikey *string
	var apiSecret *string
	var destinationS3Bucket *string
	var tags *string
	var orgId *string
	var err error
	var aggregationStat *string
	enabledCompression := false
	enableAlignTimeWindow := false

	logger.Infoln("------ Reading parameters from SSM")
	paramReader, err := parameters.New()
	if err != nil {
		return nil, err
	}

	if stackName != "" {
		logger.Infoln("------ Configured stack: " + stackName)
		apikey, err = paramReader.ReadConfigByStack(IoTApiKeyStack, stackName)
		if err != nil {
			logger.Error("Error reading parameter "+paramReader.ResolveParameter(IoTApiKeyStack, stackName), err)
		}
		apiSecret, err = paramReader.ReadConfigByStack(IoTApiSecretStack, stackName)
		if err != nil {
			logger.Error("Error reading parameter "+paramReader.ResolveParameter(IoTApiSecretStack, stackName), err)
		}
		destinationS3Bucket, err = paramReader.ReadConfigByStack(DestinationS3BucketStack, stackName)
		if err != nil || destinationS3Bucket == nil || *destinationS3Bucket == "" {
			logger.Error("Error reading parameter "+paramReader.ResolveParameter(DestinationS3BucketStack, stackName), err)
		}
		orgId, _ = paramReader.ReadConfigByStack(IoTApiOrgIdStack, stackName)
		tagsParam, _ := paramReader.ReadConfigByStack(IoTApiTagsStack, stackName)
		if tagsParam != nil {
			tags = tagsParam
		}
		aggregationStat, _ = paramReader.ReadConfigByStack(AggregationStatStack, stackName)

		alignTs, _ := paramReader.ReadConfigByStack(AlignWithTimeWindowStack, stackName)
		if alignTs != nil && *alignTs == "true" {
			enableAlignTimeWindow = true
		}

		compression, _ := paramReader.ReadConfigByStack(EnableCompressionStack, stackName)
		if compression != nil && *compression == "true" {
			enabledCompression = true
		}

	} else {
		apikey, err = paramReader.ReadConfig(IoTApiKey)
		if err != nil {
			logger.Error("Error reading parameter "+IoTApiKey, err)
		}
		apiSecret, err = paramReader.ReadConfig(IoTApiSecret)
		if err != nil {
			logger.Error("Error reading parameter "+IoTApiSecret, err)
		}
		destinationS3Bucket, err = paramReader.ReadConfig(DestinationS3Bucket)
		if err != nil || destinationS3Bucket == nil || *destinationS3Bucket == "" {
			logger.Error("Error reading parameter "+DestinationS3Bucket, err)
		}
		orgId, _ = paramReader.ReadConfig(IoTApiOrgId)
		tagsParam, _ := paramReader.ReadConfig(IoTApiTags)
		if tagsParam != nil {
			tags = tagsParam
		}
	}

	organizationId := ""
	if orgId != nil {
		organizationId = *orgId
	}
	if apikey == nil || apiSecret == nil {
		return nil, errors.New("key and secret are required")
	}
	if aggregationStat == nil {
		avgAggregation := "AVG"
		aggregationStat = &avgAggregation
	}

	// Resolve resolution
	resolution, err := configureExtractionResolution(logger, paramReader, stackName)
	if err != nil {
		return nil, err
	}

	// Resolve scheduling
	extractionWindowMinutes, err := configureDataExtractionTimeWindow(logger, paramReader, stackName)
	if err != nil {
		return nil, err
	}

	if *extractionWindowMinutes > 60 && *resolution <= 60 {
		logger.Warn("Resolution must be greater than 60 seconds for time windows greater than 60 minutes. Setting resolution to 5 minutes.")
		defReso := SamplesResolutionSeconds
		resolution = &defReso
	}

	logger.Infoln("------ Running import")
	if event.Dev || os.Getenv("DEV") == "true" {
		logger.Infoln("Running in dev mode")
		os.Setenv("IOT_API_URL", "https://api2.oniudra.cc")
	}
	logger.Infoln("key:", *apikey)
	logger.Infoln("secret:", "*********")
	if organizationId != "" {
		logger.Infoln("organizationId:", organizationId)
	} else {
		logger.Infoln("organizationId: not set")
	}
	if tags != nil {
		logger.Infoln("tags:", *tags)
	}
	if *resolution <= 0 {
		logger.Infoln("resolution: raw")
	} else {
		logger.Infoln("resolution:", *resolution, "seconds")
	}
	logger.Infoln("aggregation statistic:", *aggregationStat)
	logger.Infoln("data extraction time window:", *extractionWindowMinutes, "minutes")
	logger.Infoln("file compression enabled:", enabledCompression)
	logger.Infoln("align time window:", enableAlignTimeWindow)

	err = exporter.StartExporter(ctx, logger, *apikey, *apiSecret, organizationId, tags, *resolution, *extractionWindowMinutes, *destinationS3Bucket, *aggregationStat, enabledCompression, enableAlignTimeWindow)
	if err != nil {
		message := "Error detected during data export"
		return &message, err
	}

	message := "Data exported successfully"
	return &message, nil
}

func configureExtractionResolution(logger *logrus.Entry, paramReader *parameters.ParametersClient, stack string) (*int, error) {
	var resolution *int
	var res *string
	var err error
	if stack != "" {
		res, err = paramReader.ReadConfigByStack(SamplesResoStack, stack)
		if err != nil {
			logger.Error("Error reading parameter "+paramReader.ResolveParameter(SamplesResoStack, stack), err)
		}
	} else {
		resolution, err = paramReader.ReadIntConfig(SamplesReso)
		if err != nil {
			// Possibly this parameter is not set. Try SamplesReso
			res, err = paramReader.ReadConfig(SamplesReso)
			if err != nil {
				logger.Error("Error reading parameter "+SamplesReso, err)
				return nil, err
			}
		} else {
			return resolution, nil
		}
	}

	val := SamplesResolutionSeconds
	switch *res {
	case "raw":
		val = -1
	case "1 minute":
		val = 60
	case "5 minutes":
		val = 300
	case "15 minutes":
		val = 900
	case "1 hour":
		val = 3600
	}
	resolution = &val
	if *resolution > 3600 {
		logger.Errorf("Resolution %d is invalid", *resolution)
		return nil, errors.New("resolution must be between -1 and 3600")
	}
	return resolution, nil
}

func configureDataExtractionTimeWindow(logger *logrus.Entry, paramReader *parameters.ParametersClient, stack string) (*int, error) {
	var schedule *string
	var err error
	if stack != "" {
		schedule, err = paramReader.ReadConfigByStack(SchedulingStack, stack)
		if err != nil {
			logger.Error("Error reading parameter "+paramReader.ResolveParameter(SchedulingStack, stack), err)
		}
	} else {
		schedule, err = paramReader.ReadConfig(Scheduling)
	}
	if err != nil {
		logger.Error("Error reading parameter "+Scheduling, err)
		return nil, err
	}
	extractionWindowMinutes := DefaultTimeExtractionWindowMinutes
	switch *schedule {
	case "5 minutes":
		extractionWindowMinutes = 5
	case "15 minutes":
		extractionWindowMinutes = 15
	case "1 hour":
		extractionWindowMinutes = 60
	case "1 day":
		extractionWindowMinutes = 24 * 60
	}
	return &extractionWindowMinutes, nil
}

func main() {
	lambda.Start(HandleRequest)
}
