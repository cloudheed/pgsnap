// Package storage provides storage backend implementations for backup data.
package storage

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Backend implements Backend using Amazon S3 or S3-compatible storage.
type S3Backend struct {
	client *s3.Client
	bucket string
	prefix string
}

// S3Options configures the S3 backend.
type S3Options struct {
	Bucket    string
	Region    string
	Endpoint  string // Custom endpoint for S3-compatible storage (MinIO, etc.)
	AccessKey string
	SecretKey string
	Prefix    string // Optional prefix for all keys
}

// NewS3Backend creates a new S3Backend with the given options.
func NewS3Backend(ctx context.Context, opts S3Options) (*S3Backend, error) {
	var cfgOpts []func(*config.LoadOptions) error

	cfgOpts = append(cfgOpts, config.WithRegion(opts.Region))

	// Use explicit credentials if provided
	if opts.AccessKey != "" && opts.SecretKey != "" {
		cfgOpts = append(cfgOpts, config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(opts.AccessKey, opts.SecretKey, ""),
		))
	}

	cfg, err := config.LoadDefaultConfig(ctx, cfgOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var clientOpts []func(*s3.Options)

	// Custom endpoint for S3-compatible storage
	if opts.Endpoint != "" {
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(opts.Endpoint)
			o.UsePathStyle = true // Required for most S3-compatible services
		})
	}

	client := s3.NewFromConfig(cfg, clientOpts...)

	return &S3Backend{
		client: client,
		bucket: opts.Bucket,
		prefix: opts.Prefix,
	}, nil
}

// Put stores data from the reader under the given key.
func (b *S3Backend) Put(ctx context.Context, key string, r io.Reader, size int64) error {
	if err := validateKey(key); err != nil {
		return err
	}

	fullKey := b.fullKey(key)

	_, err := b.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(b.bucket),
		Key:           aws.String(fullKey),
		Body:          r,
		ContentLength: aws.Int64(size),
	})
	if err != nil {
		return fmt.Errorf("failed to put object: %w", err)
	}

	return nil
}

// Get retrieves the object stored under the given key.
func (b *S3Backend) Get(ctx context.Context, key string) (io.ReadCloser, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	fullKey := b.fullKey(key)

	result, err := b.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get object: %w", err)
	}

	return result.Body, nil
}

// List returns information about objects matching the given prefix.
func (b *S3Backend) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	fullPrefix := b.fullKey(prefix)

	var objects []ObjectInfo

	paginator := s3.NewListObjectsV2Paginator(b.client, &s3.ListObjectsV2Input{
		Bucket: aws.String(b.bucket),
		Prefix: aws.String(fullPrefix),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects: %w", err)
		}

		for _, obj := range page.Contents {
			// Remove prefix to get relative key
			key := *obj.Key
			if b.prefix != "" {
				key = key[len(b.prefix):]
				if len(key) > 0 && key[0] == '/' {
					key = key[1:]
				}
			}

			objects = append(objects, ObjectInfo{
				Key:          key,
				Size:         *obj.Size,
				LastModified: *obj.LastModified,
			})
		}
	}

	return objects, nil
}

// Delete removes the object stored under the given key.
func (b *S3Backend) Delete(ctx context.Context, key string) error {
	if err := validateKey(key); err != nil {
		return err
	}

	fullKey := b.fullKey(key)

	// Check if object exists first
	_, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		if isS3NotFound(err) {
			return ErrNotFound
		}
		return fmt.Errorf("failed to check object: %w", err)
	}

	_, err = b.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		return fmt.Errorf("failed to delete object: %w", err)
	}

	return nil
}

// Stat returns information about the object stored under the given key.
func (b *S3Backend) Stat(ctx context.Context, key string) (*ObjectInfo, error) {
	if err := validateKey(key); err != nil {
		return nil, err
	}

	fullKey := b.fullKey(key)

	result, err := b.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(fullKey),
	})
	if err != nil {
		if isS3NotFound(err) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}

	return &ObjectInfo{
		Key:          key,
		Size:         *result.ContentLength,
		LastModified: *result.LastModified,
		ContentType:  aws.ToString(result.ContentType),
	}, nil
}

func (b *S3Backend) fullKey(key string) string {
	if b.prefix == "" {
		return key
	}
	return b.prefix + "/" + key
}

func isS3NotFound(err error) bool {
	// Check for common S3 not found error patterns
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "NotFound") || contains(errStr, "NoSuchKey") || contains(errStr, "404")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
