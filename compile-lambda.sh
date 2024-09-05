#!/bin/bash

mkdir -p deployment/binaries
GOOS=linux CGO_ENABLED=0 go build -o bootstrap -tags lambda.norpc lambda.go
zip arduino-s3-integration-lambda.zip bootstrap
mv arduino-s3-integration-lambda.zip deployment/binaries/
rm bootstrap
echo "deployment/binaries/arduino-s3-integration-lambda.zip archive created"
