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

package parameters

import (
	"context"
	"strconv"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type ParametersClient struct {
	ssmcl *ssm.Client
}

func New() (*ParametersClient, error) {

	awsOpts := []func(*config.LoadOptions) error{}

	cfg, err := config.LoadDefaultConfig(
		context.Background(),
		awsOpts...,
	)
	if err != nil {
		return nil, err
	}

	cl := ssm.NewFromConfig(cfg)

	return &ParametersClient{
		ssmcl: cl,
	}, nil
}

func (c *ParametersClient) ReadConfig(param string) (*string, error) {
	value, err := c.ssmcl.GetParameter(context.Background(), &ssm.GetParameterInput{
		Name:           aws.String(param),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	paramValue := value.Parameter.Value
	if paramValue == nil || *paramValue == "<empty>" {
		defaultValue := ""
		return &defaultValue, nil
	}
	return paramValue, nil
}

func (c *ParametersClient) ReadIntConfig(param string) (*int, error) {
	value, err := c.ssmcl.GetParameter(context.Background(), &ssm.GetParameterInput{
		Name:           aws.String(param),
		WithDecryption: aws.Bool(true),
	})
	if err != nil {
		return nil, err
	}
	if value.Parameter.Value == nil {
		defaultValue := -1
		return &defaultValue, nil
	}
	strconvValue, err := strconv.Atoi(*value.Parameter.Value)
	if err != nil {
		return nil, err
	}
	return &strconvValue, nil
}
