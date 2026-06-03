package repositories

import (
	"bytes"
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type S3Repository struct {
	client     *s3.Client
	otelClient TelemetryClient
}

func NewS3Repository(cfg aws.Config, otelClient TelemetryClient) *S3Repository {
	return &S3Repository{
		client: s3.NewFromConfig(cfg, func(o *s3.Options) {
			o.UsePathStyle = true
		}),
		otelClient: otelClient,
	}
}

func (r *S3Repository) UploadBytes(ctx context.Context, bucket, key string, data []byte, contentType string) (string, error) {
	ctx, span := r.otelClient.StartSpan(ctx, "S3Repository.UploadBytes",
		WithAttribute("bucket", bucket),
		WithAttribute("key", key),
		WithAttribute("contentType", contentType),
	)
	defer span.End()

	_, err := r.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        bytes.NewReader(data),
		ContentType: aws.String(contentType),
	})
	if err != nil {
		span.RecordError(err)
		span.SetStatus("error", err.Error())
		return "", err
	}
	s3URI := fmt.Sprintf("s3://%s/%s", bucket, key)
	span.SetStatus("ok", "success")
	return s3URI, nil
}
