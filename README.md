# Arduino AWS S3 exporter

This project provides a way to extract time series samples from Arduino cloud, publishing to a S3 destination bucket.
Things can be filterd by tags.

## Deployment schema

S3 exporter is based on a Go lambda function triggered by periodic events from EventBridge.
Job is configured to extract samples for a 60min time window.
One file is created per run, containing all samples for the given hour. Time series samples are exported in UTC.

CSV produced has the following structure:
```console
timestamp,thing_id,thing_name,property_id,property_name,value
2024-09-04T11:00:00Z,07846f3c-37ae-4722-a3f5-65d7b4449ad3,H7,137c02d0-b50f-47fb-a2eb-b6d23884ec51,m3,3.000
2024-09-04T11:01:00Z,07846f3c-37ae-4722-a3f5-65d7b4449ad3,H7,137c02d0-b50f-47fb-a2eb-b6d23884ec51,m3,7.000
2024-09-04T11:02:00Z,07846f3c-37ae-4722-a3f5-65d7b4449ad3,H7,137c02d0-b50f-47fb-a2eb-b6d23884ec51,m3,11.000
2024-09-04T11:03:00Z,07846f3c-37ae-4722-a3f5-65d7b4449ad3,H7,137c02d0-b50f-47fb-a2eb-b6d23884ec51,m3,15.000
```

Files are organized in S3 bucket by date and files of the same day are grouped.
```
<bucket>:2024-09-04/2024-09-04-10.csv
<bucket>:2024-09-04/2024-09-04-11.csv
<bucket>:2024-09-04/2024-09-04-12.csv
```

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

### Configuration parameters

Here is a list of all configuration properties available in 'Parameter store'. 

| Parameter | Description |
| --------- | ----------- |
| /arduino/s3-importer/iot/api-key  | IoT API key |
| /arduino/s3-importer/iot/api-secret | IoT API secret |
| /arduino/s3-importer/iot/org-id    | (optional) organization id |
| /arduino/s3-importer/iot/filter/tags    | (optional) tags filtering. Syntax: tag=value,tag2=value2  |
| /arduino/s3-importer/iot/samples-resolution-seconds  | (optional) samples resolution (default: 300s) |
| /arduino/s3-importer/destination-bucket  | S3 destination bucket |

### Policies

See policies defined in [cloud formation template](deployment/cloud-formation-template/deployment.yaml)
