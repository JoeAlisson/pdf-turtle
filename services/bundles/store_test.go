//go:build integration

package bundles

import (
	"bytes"
	"context"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"
)

func TestMinioStore_Save(t *testing.T) {
	ctx := context.Background()
	endpoint := getEnvOrDefault("S3_ENDPOINT", "localhost:9000")
	accessKey := getEnvOrDefault("S3_ACCESS_KEY", "minio")
	secretKey := getEnvOrDefault("S3_SECRET_KEY", "minio123")
	bucketName := "test-save-html-bundle"

	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})
	if err != nil {
		t.Fatalf("error creating minio client: %v", err)
	}

	exists, err := mc.BucketExists(ctx, bucketName)
	if err != nil {
		t.Fatalf("error checking bucket: %v", err)
	}

	if !exists {
		if err = mc.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			t.Fatalf("error creating bucket: %v", err)
		}
	}

	store, err := NewMinioStore(MinioOptions{
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucketName,
		UseSSL:    false,
	})

	if err != nil {
		t.Fatalf("error creating minio store: %v", err)
	}

	_ = store.client.MakeBucket(context.Background(), store.bucket, minio.MakeBucketOptions{})

	t.Run("Should save a file in the store", func(t *testing.T) {
		id, err := store.Save(context.Background(), Info{
			ContentType:    "text/plain",
			Name:           "test",
			FileName:       "test.txt",
			Size:           4,
			TemplateEngine: "golang",
			Data:           bytes.NewReader([]byte("test")),
		})

		if err != nil {
			t.Fatalf("error saving file: %v", err)
		}

		obj, err := store.client.GetObject(context.Background(), store.bucket, bundlePath+id.String(), minio.GetObjectOptions{})
		if err != nil {
			t.Fatalf("error getting object: %v", err)
		}

		defer obj.Close()

		info, err := obj.Stat()
		if err != nil {
			t.Fatalf("error getting object info: %v", err)
		}

		if info.ContentType != "text/plain" {
			t.Errorf("expected content type to be text/plain, got %s", info.ContentType)
		}

		if info.Size != 4 {
			t.Errorf("expected size to be 4, got %d", info.Size)
		}

		if info.UserMetadata["Name"] != "test" {
			t.Errorf("expected name to be test, got %s", info.UserMetadata["Name"])
		}

		if info.UserMetadata["File-Name"] != "test.txt" {
			t.Errorf("expected filename to be test.txt, got %s", info.UserMetadata["File-Name"])
		}

		if info.UserMetadata["Template-Engine"] != "golang" {
			t.Errorf("expected template engine to be golang, got %s", info.UserMetadata["Template-Engine"])
		}

		if info.UserMetadata["Id"] != id.String() {
			t.Errorf("expected id to be %s, got %s", id.String(), info.UserMetadata["Id"])
		}

		if err = store.client.RemoveObject(context.Background(), store.bucket, bundlePath+id.String(), minio.RemoveObjectOptions{}); err != nil {
			t.Fatalf("error removing object: %v", err)
		}
	})
}

func getEnvOrDefault(key string, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
