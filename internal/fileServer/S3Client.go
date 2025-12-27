package fileServer

import (
	"GH-Server/pkg/zaplog"
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	_ "github.com/joho/godotenv/autoload"
)

var (
	endpoint   = os.Getenv("RUSTFS_ENDPOINT")
	accessKey  = os.Getenv("RUSTFS_ACCESS_KEY")
	secretKey  = os.Getenv("RUSTFS_SECRET_KEY")
	bucketName = os.Getenv("RUSTFS_BUCKET")
	region     = os.Getenv("RUSTFS_REGION")
)

type RustFSService interface {
}

func New() *s3.Client {
	s3Client := s3.NewFromConfig(aws.Config{
		Region:       region,
		BaseEndpoint: aws.String(endpoint),
		Credentials:  aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	})
	if _, err := s3Client.CreateBucket(context.Background(), &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	}); err != nil {
		zaplog.Zap.Panic(fmt.Sprintf("failed to create bucket: %v", err))
	}
	return s3Client
}
