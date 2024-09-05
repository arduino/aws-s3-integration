package main

import (
	"context"
	"errors"
	"os"

	"github.com/arduino/aws-s3-integration/app/importer"
	"github.com/arduino/aws-s3-integration/internal/parameters"
	"github.com/sirupsen/logrus"
)

type AWSS3ImportTrigger struct {
	Dev bool `json:"dev"`
}

const (
	ArduinoPrefix               = "/arduino/s3-importer"
	IoTApiKey                   = ArduinoPrefix + "/iot/api-key"
	IoTApiSecret                = ArduinoPrefix + "/iot/api-secret"
	IoTApiOrgId                 = ArduinoPrefix + "/iot/org-id"
	IoTApiTags                  = ArduinoPrefix + "/iot/filter/tags"
	SamplesResoSec              = ArduinoPrefix + "/iot/samples-resolution-seconds"
	DestinationS3Bucket         = ArduinoPrefix + "/destination-bucket"
	SamplesResolutionSeconds    = 300
	TimeExtractionWindowMinutes = 60
)

func HandleRequest(ctx context.Context, dev bool) (*string, error) {

	logger := logrus.NewEntry(logrus.New())

	var tags *string

	logger.Infoln("------ Reading parameters from SSM")
	paramReader, err := parameters.New()
	if err != nil {
		return nil, err
	}
	apikey, err := paramReader.ReadConfig(IoTApiKey)
	if err != nil {
		logger.Error("Error reading parameter "+IoTApiKey, err)
	}
	apiSecret, err := paramReader.ReadConfig(IoTApiSecret)
	if err != nil {
		logger.Error("Error reading parameter "+IoTApiSecret, err)
	}
	destinationS3Bucket, err := paramReader.ReadConfig(DestinationS3Bucket)
	if err != nil || destinationS3Bucket == nil || *destinationS3Bucket == "" {
		logger.Error("Error reading parameter "+DestinationS3Bucket, err)
	}
	origId, _ := paramReader.ReadConfig(IoTApiOrgId)
	organizationId := ""
	if origId != nil {
		organizationId = *origId
	}
	if apikey == nil || apiSecret == nil {
		return nil, errors.New("key and secret are required")
	}
	tagsParam, _ := paramReader.ReadConfig(IoTApiTags)
	if tagsParam != nil {
		tags = tagsParam
	}
	resolution, err := paramReader.ReadIntConfig(SamplesResoSec)
	if err != nil {
		logger.Warn("Error reading parameter "+SamplesResoSec+". Set resolution to default value", err)
		res := SamplesResolutionSeconds
		resolution = &res
	}
	if *resolution < 60 || *resolution > 3600 {
		logger.Errorf("Resolution %d is invalid", *resolution)
		return nil, errors.New("resolution must be between 60 and 3600")
	}

	logger.Infoln("------ Running import...")
	if dev || os.Getenv("DEV") == "true" {
		logger.Infoln("Running in dev mode")
		os.Setenv("IOT_API_URL", "https://api2.oniudra.cc")
	}
	logger.Infoln("key:", *apikey)
	logger.Infoln("secret:", "*********")
	logger.Infoln("resolution:", *resolution)
	if organizationId != "" {
		logger.Infoln("organizationId:", organizationId)
	} else {
		logger.Infoln("organizationId: not set")
	}
	if tags != nil {
		logger.Infoln("tags:", *tags)
	}

	err = importer.StartImport(ctx, logger, *apikey, *apiSecret, organizationId, tags, *resolution, TimeExtractionWindowMinutes, *destinationS3Bucket)
	if err != nil {
		return nil, err
	}

	message := "Data exported successfully"
	return &message, nil
}

func main() {
	msg, err := HandleRequest(context.Background(), true)
	if err != nil {
		logrus.Error(err)
	}
	if msg != nil {
		logrus.Info(*msg)
	}
}
