package fileServer

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type FileUpload interface {
	UploadFile(ctx context.Context, key string, data io.Reader, contentType string) (*manager.UploadOutput, error)
}

func (fs *fileService) UploadFile(ctx context.Context, key string, data io.Reader, contentType string) (*manager.UploadOutput, error) {
	return manager.NewUploader(fs.Client).Upload(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucketName),
		Key:         aws.String(key),
		Body:        data,
		ContentType: aws.String(contentType),
	})
}