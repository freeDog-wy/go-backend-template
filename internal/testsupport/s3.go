package testsupport

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/freeDog-wy/go-backend-template/pkg/envfile"
)

const (
	s3EndpointEnv        = "TEST_S3_ENDPOINT"
	s3RegionEnv          = "TEST_S3_REGION"
	s3AccessKeyIDEnv     = "TEST_S3_ACCESS_KEY_ID"
	s3SecretAccessKeyEnv = "TEST_S3_SECRET_ACCESS_KEY"
	s3BucketEnv          = "TEST_S3_BUCKET"
)

// S3 contains an isolated bucket created for one integration test.
type S3 struct {
	Endpoint        string
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	Bucket          string
}

// OpenS3 creates a unique bucket using explicit integration-test credentials.
func OpenS3(t testing.TB) *S3 {
	t.Helper()
	if err := envfile.LoadNearest(".env"); err != nil {
		t.Fatalf("load nearest .env: %v", err)
	}

	resource := &S3{
		Endpoint:        requiredEnv(t, s3EndpointEnv),
		Region:          requiredEnv(t, s3RegionEnv),
		AccessKeyID:     requiredEnv(t, s3AccessKeyIDEnv),
		SecretAccessKey: requiredEnv(t, s3SecretAccessKeyEnv),
		Bucket:          fmt.Sprintf("%s-%d", requiredEnv(t, s3BucketEnv), time.Now().UnixNano()),
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(),
		awsconfig.WithRegion(resource.Region),
		awsconfig.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(resource.AccessKeyID, resource.SecretAccessKey, "")),
		awsconfig.WithBaseEndpoint(strings.TrimRight(resource.Endpoint, "/")),
	)
	if err != nil {
		t.Fatalf("configure S3 integration client: %v", err)
	}
	client := s3.NewFromConfig(awsCfg, func(o *s3.Options) { o.UsePathStyle = true })
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(resource.Bucket)}); err != nil {
		t.Fatalf("create S3 test bucket %q: %v", resource.Bucket, err)
	}
	t.Cleanup(func() { cleanupS3Bucket(t, client, resource.Bucket) })
	return resource
}

func requiredEnv(t testing.TB, key string) string {
	t.Helper()
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		t.Fatalf("%s must be set for S3 integration tests", key)
	}
	return value
}

func cleanupS3Bucket(t testing.TB, client *s3.Client, bucket string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			t.Logf("list S3 test bucket %q for cleanup: %v", bucket, err)
			return
		}
		objects := make([]types.ObjectIdentifier, 0, len(page.Contents))
		for _, object := range page.Contents {
			objects = append(objects, types.ObjectIdentifier{Key: object.Key})
		}
		if len(objects) == 0 {
			continue
		}
		if _, err := client.DeleteObjects(ctx, &s3.DeleteObjectsInput{Bucket: aws.String(bucket), Delete: &types.Delete{Objects: objects, Quiet: aws.Bool(true)}}); err != nil {
			t.Logf("delete objects from S3 test bucket %q: %v", bucket, err)
			return
		}
	}
	if _, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(bucket)}); err != nil {
		t.Logf("delete S3 test bucket %q: %v", bucket, err)
	}
}
