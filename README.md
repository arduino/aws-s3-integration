# Arduino AWS S3 CSV exporter

This project provides a way to extract time series samples from Arduino cloud, publishing to a S3 destination bucket.
Data are extracted at the given resolution via a scheduled Lambda function. Then samples are stored in CSV files and saved to S3.

## Architecture

S3 exporter is based on a Go lambda function triggered by periodic event from EventBridge.
Job is configured to extract samples for a 60min time window with the default resolution of 5min.
One file is created per execution, containing all samples for selected things. Time series samples are exported at UTC timezone.
By default, all Arduino things present in the account are exported: it is possible to filter them via tags configuration.

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

CFT deployment requires:
* an AWS account with rights for creating a new CFT stack. Account must have rights to create:
  * S3 buckets
  * IAM Roles
  * Lambda functions
  * EventBridge rules
  * SSM parameters (Parameter store)

Before stack creation, two S3 buckets has to be created:
* a temporary bucket where lambda binaries and CFT can be uploaded
* CSVs destination bucket, where all generated file will be uploaded 

To deploy stack, follow these steps:
* download [lambda code binaries](deployment/binaries/arduino-s3-integration-lambda.zip) and [Cloud Formation Template](deployment/cloud-formation-template/deployment.yaml)
* Upload CFT and zip files on an S3 bucket accessible by the AWS account. This step is required by CFT procedure.
* Upload Cloud Formation Templaplate 
* Start creation of a new cloud formation stack. Follow these steps:
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

### Compile code

Code requires go v 1.22.
To compile code:

```console
foo@bar:~$ ./compile-lambda.sh
arduino-s3-integration-lambda.zip archive created
```