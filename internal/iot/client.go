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
	"fmt"
	"time"

	iotclient "github.com/arduino/iot-client-go/v2"
	"golang.org/x/oauth2"
)

var ErrOtaAlreadyInProgress = fmt.Errorf("ota already in progress")

// Client can perform actions on Arduino IoT Cloud.
type Client struct {
	api   *iotclient.APIClient
	token oauth2.TokenSource
}

// NewClient returns a new client implementing the Client interface.
// It needs client Credentials for cloud authentication.
func NewClient(key, secret, organization string) (*Client, error) {
	cl := &Client{}
	err := cl.setup(key, secret, organization)
	if err != nil {
		err = fmt.Errorf("instantiate new iot client: %w", err)
		return nil, err
	}
	return cl, nil
}

// DeviceList retrieves and returns a list of all Arduino IoT Cloud devices
// belonging to the user performing the request.
func (cl *Client) DeviceList(ctx context.Context, tags map[string]string) ([]iotclient.ArduinoDevicev2, error) {
	ctx, err := ctxWithToken(ctx, cl.token)
	if err != nil {
		return nil, err
	}

	request := cl.api.DevicesV2Api.DevicesV2List(ctx)
	if tags != nil {
		t := make([]string, 0, len(tags))
		for key, val := range tags {
			// Use the 'key:value' format required from the backend
			t = append(t, key+":"+val)
		}
		request = request.Tags(t)
	}
	devices, _, err := cl.api.DevicesV2Api.DevicesV2ListExecute(request)
	if err != nil {
		err = fmt.Errorf("listing devices: %w", errorDetail(err))
		return nil, err
	}
	return devices, nil
}

// DeviceShow allows to retrieve a specific device, given its id,
// from Arduino IoT Cloud.
func (cl *Client) DeviceShow(ctx context.Context, id string) (*iotclient.ArduinoDevicev2, error) {
	ctx, err := ctxWithToken(ctx, cl.token)
	if err != nil {
		return nil, err
	}

	request := cl.api.DevicesV2Api.DevicesV2Show(ctx, id)
	dev, _, err := cl.api.DevicesV2Api.DevicesV2ShowExecute(request)
	if err != nil {
		err = fmt.Errorf("retrieving device, %w", errorDetail(err))
		return nil, err
	}
	return dev, nil
}

// DeviceTagsCreate allows to create or overwrite tags on a device of Arduino IoT Cloud.
func (cl *Client) DeviceTagsCreate(ctx context.Context, id string, tags map[string]string) error {
	ctx, err := ctxWithToken(ctx, cl.token)
	if err != nil {
		return err
	}

	for key, val := range tags {
		t := iotclient.Tag{Key: key, Value: val}
		request := cl.api.DevicesV2TagsApi.DevicesV2TagsUpsert(ctx, id)
		request = request.Tag(t)
		_, err := cl.api.DevicesV2TagsApi.DevicesV2TagsUpsertExecute(request)
		if err != nil {
			err = fmt.Errorf("cannot create tag %s: %w", key, errorDetail(err))
			return err
		}
	}
	return nil
}

// DeviceTagsDelete deletes the tags of a device of Arduino IoT Cloud,
// given the device id and the keys of the tags.
func (cl *Client) DeviceTagsDelete(ctx context.Context, id string, keys []string) error {
	ctx, err := ctxWithToken(ctx, cl.token)
	if err != nil {
		return err
	}

	for _, key := range keys {
		request := cl.api.DevicesV2TagsApi.DevicesV2TagsDelete(ctx, id, key)
		_, err := cl.api.DevicesV2TagsApi.DevicesV2TagsDeleteExecute(request)
		if err != nil {
			err = fmt.Errorf("cannot delete tag %s: %w", key, errorDetail(err))
			return err
		}
	}
	return nil
}

// ThingShow allows to retrieve a specific thing, given its id,
// from Arduino IoT Cloud.
func (cl *Client) ThingShow(ctx context.Context, id string) (*iotclient.ArduinoThing, error) {
	ctx, err := ctxWithToken(ctx, cl.token)
	if err != nil {
		return nil, err
	}

	request := cl.api.ThingsV2Api.ThingsV2Show(ctx, id)
	thing, _, err := cl.api.ThingsV2Api.ThingsV2ShowExecute(request)
	if err != nil {
		err = fmt.Errorf("retrieving thing, %w", errorDetail(err))
		return nil, err
	}
	return thing, nil
}

// ThingList returns a list of things on Arduino IoT Cloud.
func (cl *Client) ThingList(ctx context.Context, ids []string, device *string, props bool, tags map[string]string) ([]iotclient.ArduinoThing, error) {
	ctx, err := ctxWithToken(ctx, cl.token)
	if err != nil {
		return nil, err
	}

	request := cl.api.ThingsV2Api.ThingsV2List(ctx)
	request = request.ShowProperties(props)

	if ids != nil {
		request = request.Ids(ids)
	}

	if device != nil {
		request = request.DeviceId(*device)
	}

	if tags != nil {
		t := make([]string, 0, len(tags))
		for key, val := range tags {
			// Use the 'key:value' format required from the backend
			t = append(t, key+":"+val)
		}
		request = request.Tags(t)
	}

	things, _, err := cl.api.ThingsV2Api.ThingsV2ListExecute(request)
	if err != nil {
		err = fmt.Errorf("retrieving things, %w", errorDetail(err))
		return nil, err
	}
	return things, nil
}

// ThingTagsCreate allows to create or overwrite tags on a thing of Arduino IoT Cloud.
func (cl *Client) ThingTagsCreate(ctx context.Context, id string, tags map[string]string) error {
	ctx, err := ctxWithToken(ctx, cl.token)
	if err != nil {
		return err
	}

	for key, val := range tags {
		t := iotclient.Tag{Key: key, Value: val}
		request := cl.api.ThingsV2TagsApi.ThingsV2TagsUpsert(ctx, id)
		_, err := cl.api.ThingsV2TagsApi.ThingsV2TagsUpsertExecute(request.Tag(t))
		if err != nil {
			err = fmt.Errorf("cannot create tag %s: %w", key, errorDetail(err))
			return err
		}
	}
	return nil
}

// ThingTagsDelete deletes the tags of a thing of Arduino IoT Cloud,
// given the thing id and the keys of the tags.
func (cl *Client) ThingTagsDelete(ctx context.Context, id string, keys []string) error {
	ctx, err := ctxWithToken(ctx, cl.token)
	if err != nil {
		return err
	}

	for _, key := range keys {
		request := cl.api.ThingsV2TagsApi.ThingsV2TagsDelete(ctx, id, key)
		_, err := cl.api.ThingsV2TagsApi.ThingsV2TagsDeleteExecute(request)
		if err != nil {
			err = fmt.Errorf("cannot delete tag %s: %w", key, errorDetail(err))
			return err
		}
	}
	return nil
}

func (cl *Client) GetTimeSeries(ctx context.Context, properties []string, from, to time.Time, interval int64) (*iotclient.ArduinoSeriesBatch, bool, error) {
	if len(properties) == 0 {
		return nil, false, fmt.Errorf("no properties provided")
	}

	ctx, err := ctxWithToken(ctx, cl.token)
	if err != nil {
		return nil, false, err
	}

	requests := make([]iotclient.BatchQueryRequestMediaV1, 0, len(properties))
	for _, prop := range properties {
		if prop == "" {
			continue
		}
		requests = append(requests, iotclient.BatchQueryRequestMediaV1{
			From:     from,
			Interval: &interval,
			Q:        fmt.Sprintf("property.%s", prop),
			To:       to,
		})
	}

	if len(requests) == 0 {
		return nil, false, fmt.Errorf("no valid properties provided")
	}

	batchQueryRequestsMediaV1 := iotclient.BatchQueryRequestsMediaV1{
		Requests: requests,
	}

	request := cl.api.SeriesV2Api.SeriesV2BatchQuery(ctx)
	request = request.BatchQueryRequestsMediaV1(batchQueryRequestsMediaV1)
	ts, httpResponse, err := cl.api.SeriesV2Api.SeriesV2BatchQueryExecute(request)
	if err != nil {
		err = fmt.Errorf("retrieving time series: %w", errorDetail(err))
		if httpResponse != nil && httpResponse.StatusCode == 429 { // Retry if rate limited
			return nil, true, err
		}
		return nil, false, err
	}
	return ts, false, nil
}

func (cl *Client) GetTimeSeriesByThing(ctx context.Context, thingID string, from, to time.Time, interval int64) (*iotclient.ArduinoSeriesBatch, bool, error) {
	if thingID == "" {
		return nil, false, fmt.Errorf("no thing provided")
	}

	ctx, err := ctxWithToken(ctx, cl.token)
	if err != nil {
		return nil, false, err
	}

	requests := []iotclient.BatchQueryRequestMediaV1{
		{
			From:     from,
			Interval: &interval,
			Q:        fmt.Sprintf("thing.%s", thingID),
			To:       to,
		},
	}

	if len(requests) == 0 {
		return nil, false, fmt.Errorf("no valid properties provided")
	}

	batchQueryRequestsMediaV1 := iotclient.BatchQueryRequestsMediaV1{
		Requests: requests,
	}

	request := cl.api.SeriesV2Api.SeriesV2BatchQuery(ctx)
	request = request.BatchQueryRequestsMediaV1(batchQueryRequestsMediaV1)
	ts, httpResponse, err := cl.api.SeriesV2Api.SeriesV2BatchQueryExecute(request)
	if err != nil {
		err = fmt.Errorf("retrieving time series: %w", errorDetail(err))
		if httpResponse != nil && httpResponse.StatusCode == 429 { // Retry if rate limited
			return nil, true, err
		}
		return nil, false, err
	}
	return ts, false, nil
}

func (cl *Client) GetTimeSeriesStringSampling(ctx context.Context, properties []string, from, to time.Time, interval int32) (*iotclient.ArduinoSeriesBatchSampled, bool, error) {
	if len(properties) == 0 {
		return nil, false, fmt.Errorf("no properties provided")
	}

	ctx, err := ctxWithToken(ctx, cl.token)
	if err != nil {
		return nil, false, err
	}

	requests := make([]iotclient.BatchQuerySampledRequestMediaV1, 0, len(properties))
	for _, prop := range properties {
		if prop == "" {
			continue
		}
		requests = append(requests, iotclient.BatchQuerySampledRequestMediaV1{
			From:     &from,
			Interval: &interval,
			Q:        fmt.Sprintf("property.%s", prop),
			To:       &to,
		})
	}

	if len(requests) == 0 {
		return nil, false, fmt.Errorf("no valid properties provided")
	}

	batchQueryRequestsMediaV1 := iotclient.BatchQuerySampledRequestsMediaV1{
		Requests: requests,
	}

	request := cl.api.SeriesV2Api.SeriesV2BatchQuerySampling(ctx)
	request = request.BatchQuerySampledRequestsMediaV1(batchQueryRequestsMediaV1)
	ts, httpResponse, err := cl.api.SeriesV2Api.SeriesV2BatchQuerySamplingExecute(request)
	if err != nil {
		err = fmt.Errorf("retrieving time series sampling: %w", errorDetail(err))
		if httpResponse != nil && httpResponse.StatusCode == 429 { // Retry if rate limited
			return nil, true, err
		}
		return nil, false, err
	}
	return ts, false, nil
}

func (cl *Client) setup(client, secret, organization string) error {
	baseURL := GetArduinoAPIBaseURL()

	// Configure a token source given the user's credentials.
	cl.token = NewUserTokenSource(client, secret, baseURL)

	config := iotclient.NewConfiguration()
	if organization != "" {
		config.AddDefaultHeader("X-Organization", organization)
	}
	config.Servers = iotclient.ServerConfigurations{
		{
			URL:         fmt.Sprintf("%s/iot", baseURL),
			Description: "IoT API endpoint",
		},
	}
	cl.api = iotclient.NewAPIClient(config)

	return nil
}
