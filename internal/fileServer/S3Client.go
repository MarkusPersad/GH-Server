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
	ctx := context.Background()
	s3Client := s3.NewFromConfig(aws.Config{
		Region:       region,
		BaseEndpoint: aws.String(endpoint),
		Credentials:  aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKey, secretKey, "")),
	}, func(o *s3.Options) {
		o.UsePathStyle = true
	})
	if _, err := s3Client.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	}); err != nil {
		if _, err = s3Client.CreateBucket(ctx, &s3.CreateBucketInput{
			Bucket: aws.String(bucketName),
		}); err != nil {
			zaplog.Zap.Error(fmt.Sprintf("创建bucket失败:%s", err.Error()))
		}
	}
	return s3Client
}
