#!/bin/bash

GOOS=linux CGO_ENABLED=0 go build -o bootstrap -tags lambda.norpc lambda.go
zip arduino-s3-integration-lambda.zip bootstrap
rm bootstrap
echo "arduino-s3-integration-lambda.zip archive created"
