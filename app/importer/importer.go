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

package importer

import (
	"context"

	"github.com/arduino/aws-s3-integration/business/tsextractor"
	"github.com/arduino/aws-s3-integration/internal/iot"
	"github.com/arduino/aws-s3-integration/internal/utils"
	iotclient "github.com/arduino/iot-client-go/v2"
	"github.com/sirupsen/logrus"
)

func StartImport(ctx context.Context, logger *logrus.Entry, key, secret, orgid string, tagsF *string, resolution, timeWindowMinutes int, destinationS3Bucket string, aggregationStat string) error {

	// Init client
	iotcl, err := iot.NewClient(key, secret, orgid)
	if err != nil {
		return err
	}

	if tagsF == nil {
		logger.Infoln("Things - searching with no filter")
	} else {
		logger.Infoln("Things - searching by tags: ", *tagsF)
	}
	things, err := iotcl.ThingList(ctx, nil, nil, true, utils.ParseTags(tagsF))
	if err != nil {
		return err
	}
	thingsMap := make(map[string]iotclient.ArduinoThing, len(things))
	for _, thing := range things {
		logger.Infoln("  Thing: ", thing.Id, thing.Name)
		thingsMap[thing.Id] = thing
	}

	// Extract data points from thing and push to S3
	tsextractorClient := tsextractor.New(iotcl, logger)
	if err := tsextractorClient.ExportTSToS3(ctx, timeWindowMinutes, thingsMap, resolution, destinationS3Bucket, aggregationStat); err != nil {
		logger.Error("Error aligning time series samples: ", err)
	}

	return nil
}
