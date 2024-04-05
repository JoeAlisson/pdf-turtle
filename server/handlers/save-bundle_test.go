//go:build integration

package handlers_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lucas-gaitzsch/pdf-turtle/config"
	"github.com/lucas-gaitzsch/pdf-turtle/server"
	"github.com/lucas-gaitzsch/pdf-turtle/services/bundles"

	"github.com/google/uuid"
)

func TestSaveHtmlBundleHandler(t *testing.T) {

	ctx := context.Background()
	// TODO use var envs
	bStore, _ := bundles.NewMinioStore(bundles.MinioOptions{
		Endpoint:  "localhost:9000", //os.Getenv("S3_ENDPOINT"),
		AccessKey: "minio",          //os.Getenv("S3_ACCESS_KEY"),
		SecretKey: "minio123",       //os.Getenv("S3_SECRET_KEY"),
		Bucket:    "test",
		UseSSL:    false,
	})
	bundleProviderService := bundles.NewBundleProviderService(bStore)
	ctx = context.WithValue(ctx, config.ContextKeyBundleProviderService, bundleProviderService)

	s := &server.Server{}
	s.Serve(ctx)

	t.Cleanup(func() {
		s.Close(ctx)
	})

	files := []struct {
		Name    string
		Content string
	}{
		{
			Name:    "index.html",
			Content: "<html><body><h1>Test {{ .Name }}</h1></body></html>",
		},
		{
			Name:    "footer.html",
			Content: "<footer>Test Footer</footer>",
		},
	}

	zipBuf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuf)

	for _, file := range files {
		f, _ := zipWriter.Create(file.Name)
		f.Write([]byte(file.Content))
	}

	_ = zipWriter.Close()

	t.Run("Should save a html bundle", func(t *testing.T) {
		buf := new(bytes.Buffer)
		w := multipart.NewWriter(buf)

		ff, err := w.CreateFormFile("bundle", "bundle.zip")
		if err != nil {
			t.Fatalf("error creating form file: %v", err)
		}
		if _, err = ff.Write(zipBuf.Bytes()); err != nil {
			t.Fatalf("error writing file to form file: %v", err)
		}
		if err = w.WriteField("name", "test-save-bundle"); err != nil {
			t.Fatalf("error writing field to form file: %v", err)
		}
		if err = w.WriteField("templateEngine", "golang"); err != nil {
			t.Fatalf("error writing field to form file: %v", err)
		}

		_ = w.Close()

		req := httptest.NewRequest("POST", "/api/html-bundle/save", buf)
		req.Header.Set("Content-Type", w.FormDataContentType())

		resp, err := s.Instance.Test(req, -1)
		if err != nil {
			t.Fatalf("error sending request: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected status code to be 201, got %d", resp.StatusCode)
		}

		result := struct {
			ID string `json:"id"`
		}{}
		_ = json.NewDecoder(resp.Body).Decode(&result)
		if result.ID == "" {
			t.Error("expected id to be set")
		}

		id, err := uuid.Parse(result.ID)
		if err != nil {
			t.Fatalf("error parsing id: %v", err)
		}
		info, err := bStore.GetFromStore(ctx, id)
		if err != nil {
			t.Fatalf("error getting bundle from store: %v", err)
		}

		defer info.Data.Close()

		if info.Name != "test-save-bundle" {
			t.Errorf("expected name to be test-save-bundle, got %s", info.Name)
		}

		if info.TemplateEngine != "golang" {
			t.Errorf("expected template engine to be golang, got %s", info.TemplateEngine)
		}

		if info.FileName != "bundle.zip" {
			t.Errorf("expected filename to be bundle.zip, got %s", info.FileName)
		}

		if info.ContentType != "application/octet-stream" {
			t.Errorf("expected content type to be application/octet-stream, got %s", info.ContentType)
		}

		if info.Size != int64(zipBuf.Len()) {
			t.Errorf("expected size to be %d, got %d", zipBuf.Len(), info.Size)
		}

		if info.Id != result.ID {
			t.Errorf("expected id to be %s, got %s", id, info.Id)
		}

		data := new(bytes.Buffer)
		_, _ = data.ReadFrom(info.Data)

		if bytes.Compare(zipBuf.Bytes(), data.Bytes()) != 0 {
			t.Error("expected data to be the same")
		}
	})
}
