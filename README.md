# AWS IoT S3 importer

This project provides a way to extract time series samples from Arduino cloud, publishing to an S3 destination bucket.
Things can be filterd by tags.

## Deployment schema

Imported is based on a Go lambda function triggered by periodic events from EventBridge.
Job is configured to extract samples for a 60min time window: trigger is configured accordingly on EventBridge.

### Policies

See policies defined in [cloud formation template](deployment/cloud-formation-template/deployment.yaml)

### Configuration parameters

| Parameter | Description |
| --------- | ----------- |
| /arduino/s3-importer/iot/api-key  | IoT API key |
| /arduino/s3-importer/iot/api-secret | IoT API secret |
| /arduino/s3-importer/iot/org-id    | (optional) organization id |
| /arduino/s3-importer/iot/filter/tags    | (optional) tags filtering. Syntax: tag=value,tag2=value2  |
| /arduino/s3-importer/iot/samples-resolution-seconds  | (optional) samples resolution (default: 300s) |
| /arduino/s3-importer/destination-bucket  | S3 destination bucket |

## Deployment via Cloud Formation Template

It is possible to deploy required resources via [cloud formation template](deployment/cloud-formation-template/deployment.yaml)
Required steps to deploy project:
* compile lambda
```console
foo@bar:~$ ./compile-lambda.sh
arduino-s3-integration-lambda.zip archive created
```
* Save zip file on an S3 bucket accessible by the AWS account
* Start creation of a new cloud formation stack provising the [cloud formation template](deployment/cloud-formation-template/deployment.yaml)
* Fill all required parameters (mandatory: Arduino API key and secret, S3 bucket and key where code has been uploaded, destination S3 bucket. Optionally, tag filter for filtering things, organization identifier and samples resolution)
