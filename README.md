# Arduino AWS S3 CSV exporter

This project provides a way to extract time series samples from Arduino cloud, publishing to a S3 destination bucket.
Data are extracted at the given resolution via a scheduled Lambda function. Samples are stored in CSV files and saved to S3.
By default, data extraction is performed every hour (configurable), extracting samples aggregated at 5min resolution (configurable).
Aggregation is performed as average over aggregation period. Non numeric values like strings are sampled at the given resolution.

## Architecture

S3 exporter is based on a GO Lambda function triggered by periodic event from EventBridge.
Function is triggered at a fixed rate (by default, 1 hour), starting from the deployment time.
Rate also define the time extraction window. So, with a 1 hour scheduling, one hour of data are extracted.
One file is created per execution and contains all samples for selected things. Time series samples are exported at UTC timezone.
By default, all Arduino things present in the account are exported: it is possible to filter them via [tags](#tag-filtering).

CSV produced has the following structure:
```console
timestamp,thing_id,thing_name,property_id,property_name,property_type,value,aggregation_statistic
2024-09-04T11:00:00Z,07846f3c-37ae-4722-a3f5-65d7b4449ad3,H7,137c02d0-b50f-47fb-a2eb-b6d23884ec51,m3,FLOAT,3,AVG
2024-09-04T11:01:00Z,07846f3c-37ae-4722-a3f5-65d7b4449ad3,H7,137c02d0-b50f-47fb-a2eb-b6d23884ec51,m3,FLOAT,7,AVG
2024-09-04T11:02:00Z,07846f3c-37ae-4722-a3f5-65d7b4449ad3,H7,137c02d0-b50f-47fb-a2eb-b6d23884ec51,m3,FLOAT,11,AVG
2024-09-04T11:03:00Z,07846f3c-37ae-4722-a3f5-65d7b4449ad3,H7,137c02d0-b50f-47fb-a2eb-b6d23884ec51,m3,FLOAT,15,AVG
```

Files are organized by date and files of the same day are grouped.
```
<bucket>:2024-09-04/2024-09-04-10-00.csv
<bucket>:2024-09-04/2024-09-04-11-00.csv
<bucket>:2024-09-04/2024-09-04-12-00.csv
```

Data extraction is aligned with function execution time.
It is possible to align data extracted with extraction time window (for example, export last complete hour) by configuring `/arduino/s3-exporter/{stack-name}/iot/align_with_time_window` property.

## Deployment via Cloud Formation Template

It is possible to deploy required resources via [cloud formation template](deployment/cloud-formation-template/deployment.yaml)

AWS user must have permissions to:
  * create a new CFT stack (policy: AWSCloudFormationFullAccess)
  * S3 buckets (policy: AmazonS3FullAccess)
  * IAM Roles (policy: IAMFullAccess)
  * Lambda functions (policy: AWSLambda_FullAccess)
  * EventBridge rules (policy: AmazonEventBridgeFullAccess)
  * SSM parameters (Parameter store) (policy: AmazonSSMFullAccess)

Before stack creation, two S3 buckets have to be created:
* a temporary bucket where lambda binaries and CFT can be uploaded
* CSVs destination bucket, where all generated file will be uploaded 
bucket must be in the same region where stack will be created.

Follow these steps to deploy a new stack:
* download [lambda code binaries .zip archive](https://github.com/arduino/aws-s3-integration/releases) and [Cloud Formation Template .yaml file](https://github.com/arduino/aws-s3-integration/releases)
* upload CFT and binary zip file on an S3 bucket accessible by the AWS account. For the CFT yaml file, copy the Object URL (it will be required in next step).
  
![object URL](docs/objecturl.png)

* start creation of a new cloud formation stack

![CFT 1](docs/cft-stack-1.png)

* fill all required parameters.
  <br/>**Mandatory**: Arduino API key and secret, S3 bucket where code has been uploaded, destination S3 bucket
  <br/>**Optional**: tag filter for filtering things, organization identifier and samples resolution

![CFT 2](docs/cft-stack-2.png)

### Configuration parameters

Here is a list of all configuration properties supported by exporter. It is possible to edit them in AWS Parameter store.
These parameters are filled by CFT at stack creation time and can be adjusted later in case of need (for example, API keys rotation)

| Parameter | Description |
| --------- | ----------- |
| /arduino/s3-exporter/{stack-name}/iot/api-key  | IoT API key |
| /arduino/s3-exporter/{stack-name}/iot/api-secret | IoT API secret |
| /arduino/s3-exporter/{stack-name}/iot/org-id    | (optional) organization id |
| /arduino/s3-exporter/{stack-name}/iot/filter/tags    | (optional) tags filtering. Syntax: tag=value,tag2=value2  |
| /arduino/s3-exporter/{stack-name}/iot/samples-resolution  | (optional) samples aggregation resolution (1/5/15 minutes, 1 hour, raw) |
| /arduino/s3-exporter/{stack-name}/iot/scheduling | Execution scheduling |
| /arduino/s3-exporter/{stack-name}/iot/align_with_time_window | Align data extraction with time windows (for example, last complte hour) |
| /arduino/s3-exporter/{stack-name}/iot/aggregation-statistic | Aggregation statistic |
| /arduino/s3-exporter/{stack-name}/destination-bucket  | S3 destination bucket |
| /arduino/s3-exporter/{stack-name}/enable_compression  | Compress CSV files with gzip before uploading to S3 bucket |

### Tag filtering

It is possible to filter only the Arduino Things of interest.
You can use tag filtering if you need to reduce export to a specific set of Things.

* Add a tag in Arduino Cloud UI on all Things you want to export. To do that, select a thing, go in 'Metadata' section and 'Add' a new tag.

![tag 2](docs/tag-2.png)

![tag 1](docs/tag-1.png)

* Configure tag filter during CFT creation of by editing '/arduino/s3-exporter/<stack-name>/iot/filter/tags' parameter (syntax: tag1=value1,tag2=value2).

![tag filter](docs/tag-filter.png)

### Building code

Core is built by dedicated git workflow. Release can be trigged via applying a new tag.
It's also possible to compile code locally. Code compile requires go v 1.22.
To compile code:

```console
foo@bar:~$ ./compile-lambda.sh
arduino-s3-integration-lambda.zip archive created

OR

foo@bar:~$ task go:build
```
