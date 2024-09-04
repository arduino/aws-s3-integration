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

package iot

import (
	"encoding/json"
	"fmt"

	iotclient "github.com/arduino/iot-client-go/v2"
)

// errorDetail takes a generic iot-client-go error
// and tries to return a more detailed error.
func errorDetail(err error) error {
	apiErr, ok := err.(iotclient.GenericOpenAPIError)
	if !ok {
		return err
	}

	modErr, ok := apiErr.Model().(iotclient.ModelError)
	if ok {
		if modErr.Detail != nil {
			return fmt.Errorf("%w: %s", err, *modErr.Detail)
		} else {
			return err
		}
	}

	body := make(map[string]interface{})
	if bodyErr := json.Unmarshal(apiErr.Body(), &body); bodyErr != nil {
		return err
	}
	detail, ok := body["detail"]
	if !ok {
		return err
	}
	return fmt.Errorf("%w: %v", err, detail)
}
