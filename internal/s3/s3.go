package s3

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	awsS3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type S3Client struct {
	client     *awsS3.Client
	bucketName string
}

func NewS3Client(bucketName string) (*S3Client, error) {
	cfg, err := config.LoadDefaultConfig(
		context.Background(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}
	cl := awsS3.NewFromConfig(cfg)
	// Check if we have permission to access the buckets
	checkIfBucketExists(cl, bucketName)
	return &S3Client{
		client:     cl,
		bucketName: bucketName,
	}, nil
}

func checkIfBucketExists(client *awsS3.Client, bucketName string) {
	params := awsS3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}
	if _, err := client.HeadBucket(context.Background(), &params); err != nil {
		var nsb *types.NoSuchBucket
		if errors.As(err, &nsb) {
			panic(fmt.Errorf("failed to verify that bucket %q exists: does not exist or it's not visible", bucketName))
		}
		panic(fmt.Errorf("failed to verify that bucket %q exists: %w", bucketName, err))
	}
}

func (s *S3Client) WriteFile(ctx context.Context, key, filePath string) error {
	inFile, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %s %w", filePath, err)
	}
	defer inFile.Close()
	params := awsS3.PutObjectInput{
		Bucket: aws.String(s.bucketName),
		Key:    aws.String(key),
		Body:   inFile,
	}
	_, err = s.client.PutObject(ctx, &params)
	if err != nil {
		return fmt.Errorf("failed to write file to S3: %w", err)
	}
	return nil
}
