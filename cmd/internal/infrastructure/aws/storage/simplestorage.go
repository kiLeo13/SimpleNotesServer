package storage

import (
	"bytes"
	"context"
	"errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"mime"
	"net/http"
	"os"
	"path/filepath"
)

const PathAttachments = "attachments/"

var ErrorEmptyKey = errors.New("key is empty")

type S3Client interface {
	UploadFile(data []byte, key string) error
	DeleteFile(key string) error
}

type storageClient struct {
	bucket string
	client *s3.Client
}

func NewStorageClient() (S3Client, error) {
	region := os.Getenv("AWS_S3_REGION")
	bucket := os.Getenv("S3_BUCKET_NAME")
	cfg, err := config.LoadDefaultConfig(context.Background(), config.WithRegion(region))
	if err != nil {
		return nil, err
	}

	client := s3.NewFromConfig(cfg)
	return &storageClient{
		bucket: bucket,
		client: client,
	}, nil
}

func (s *storageClient) UploadFile(data []byte, key string) error {
	if key == "" {
		return ErrorEmptyKey
	}

	mimeType := mime.TypeByExtension(filepath.Ext(key))
	if mimeType == "" {
		mimeType = http.DetectContentType(data)
	}

	input := &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: &mimeType,
	}

	_, err := s.client.PutObject(context.Background(), input)
	if err != nil {
		return err
	}
	return nil
}

func (s *storageClient) DeleteFile(key string) error {
	if key == "" {
		return ErrorEmptyKey
	}

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}

	_, err := s.client.DeleteObject(context.Background(), input)
	if err != nil {
		return err
	}
	return nil
}
