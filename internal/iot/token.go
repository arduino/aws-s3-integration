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
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"

	iotclient "github.com/arduino/iot-client-go/v2"
	"golang.org/x/oauth2"
	cc "golang.org/x/oauth2/clientcredentials"
)

func GetArduinoAPIBaseURL() string {
	baseURL := "https://api2.arduino.cc"
	if url := os.Getenv("IOT_API_URL"); url != "" {
		baseURL = url
	}
	return baseURL
}

// Build a new token source to forge api JWT tokens based on provided credentials
func NewUserTokenSource(client, secret, baseURL, organizationId string) oauth2.TokenSource {
	// We need to pass the additional "audience" var to request an access token.
	additionalValues := url.Values{}
	additionalValues.Add("audience", "https://api2.arduino.cc/iot")
	if organizationId != "" {
		additionalValues.Add("organization_id", organizationId)
	}
	// Set up OAuth2 configuration.
	config := cc.Config{
		ClientID:       client,
		ClientSecret:   secret,
		TokenURL:       baseURL + "/iot/v1/clients/token",
		EndpointParams: additionalValues,
	}

	// Retrieve a token source that allows to retrieve tokens
	// with an automatic refresh mechanism.
	return config.TokenSource(context.Background())
}

func ctxWithToken(ctx context.Context, src oauth2.TokenSource) (context.Context, error) {
	// Retrieve a valid token from the src.
	_, err := src.Token()
	if err != nil {
		if strings.Contains(err.Error(), "401") {
			return nil, errors.New("wrong credentials")
		}
		return nil, fmt.Errorf("cannot retrieve a valid token: %w", err)
	}
	return context.WithValue(ctx, iotclient.ContextOAuth2, src), nil
}
