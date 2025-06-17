//go:build integration

package handlers_test

import (
	"bytes"
	"context"
	"github.com/lucas-gaitzsch/pdf-turtle/loopback"
	"github.com/lucas-gaitzsch/pdf-turtle/services/assetsprovider"
	"github.com/lucas-gaitzsch/pdf-turtle/services/renderer"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/lucas-gaitzsch/pdf-turtle/config"
	"github.com/lucas-gaitzsch/pdf-turtle/server"
	"github.com/lucas-gaitzsch/pdf-turtle/services/bundles"
)

func TestRenderBundleByIdHandler(t *testing.T) {
	ctx := context.Background()
	endpoint := getEnvOrDefault("S3_ENDPOINT", "localhost:9000")
	accessKey := getEnvOrDefault("S3_ACCESS_KEY", "minio")
	secretKey := getEnvOrDefault("S3_SECRET_KEY", "minio123")
	bucketName := "test-render-bundle-by-id"

	mc, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: false,
	})

	exists, err := mc.BucketExists(ctx, bucketName)
	if err != nil {
		t.Fatalf("error checking bucket: %v", err)
	}

	if !exists {
		if err = mc.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{}); err != nil {
			t.Fatalf("error creating bucket: %v", err)
		}
	}

	bStore, err := bundles.NewMinioStore(bundles.MinioOptions{
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucketName,
		UseSSL:    false,
	})

	if err != nil {
		t.Fatalf("error creating minio store: %v", err)
	}

	bundleService := bundles.NewBundleProviderService(bStore)
	ctx = context.WithValue(ctx, config.ContextKeyBundleProviderService, bundleService)
	ctx = context.WithValue(ctx, config.ContextKeyRendererService, renderer.NewRendererBackgroundService(ctx))
	ctx = context.WithValue(ctx, config.ContextKeyAssetsProviderService, assetsprovider.NewAssetsProviderService())

	s := &server.Server{}
	s.Serve(ctx)

	srvLoopback := loopback.Server{}
	srvLoopback.Serve(ctx)

	files := fileList{
		{Name: "index.html", Content: "<html><body><h1>Test {{ .Name }}</h1></body></html>"},
		{Name: "footer.html", Content: "<footer>Test Footer {{ .Footer }}</footer>"},
	}

	zipBuf := createZipFileBuffer(files)

	info := bundles.Info{
		Name:           "pre-save-test-render-bundle-by-id",
		TemplateEngine: "golang",
		Data:           bytes.NewReader(zipBuf.Bytes()),
		Size:           int64(zipBuf.Len()),
		ContentType:    "application/zip",
	}

	err = bStore.Save(ctx, info)
	if err != nil {
		t.Fatalf("error saving bundle: %v", err)
	}

	t.Cleanup(func() {
		srvLoopback.Close(ctx)
		s.Close(ctx)

		if err := mc.RemoveObject(ctx, bucketName, "bundles", minio.RemoveObjectOptions{ForceDelete: true}); err != nil {
			t.Errorf("error removing object: %v", err)
		}

		if err := mc.RemoveBucket(ctx, bucketName); err != nil {
			t.Errorf("error removing bucket: %v", err)
		}
	})

	t.Run("Should render bundle by id", func(t *testing.T) {
		model := `{"Name": "Render Bundler", "Footer": "Render Footer"}`
		req := httptest.NewRequest("POST", "/api/pdf/from/html-bundle/"+url.PathEscape(info.Name), strings.NewReader(model))

		resp, err := s.Instance.Test(req, -1)
		if err != nil {
			t.Fatalf("error testing request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status code 200, got %d", resp.StatusCode)
		}

		if resp.Header.Get("Content-Type") != "application/pdf" {
			t.Errorf("expected content type application/pdf, got %s", resp.Header.Get("Content-Type"))
		}

		if resp.Header.Get("Content-Disposition") != "attachment; filename=\"pre-save-test-render-bundle-by-id.pdf\"" {
			t.Errorf("expected content disposition attachment; filename=\"pre-save-test-render-bundle-by-id.pdf\", got %s", resp.Header.Get("Content-Disposition"))
		}

		if resp.Body == nil {
			t.Fatalf("expected response body, got nil")
		}
		defer resp.Body.Close()

		data := make([]byte, 512)
		_, err = resp.Body.Read(data)
		dataType := http.DetectContentType(data)
		if dataType != "application/pdf" {
			t.Errorf("expected content type application/pdf, got %s", dataType)
		}
	})
}
