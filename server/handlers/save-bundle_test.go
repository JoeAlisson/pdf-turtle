//go:build integration

package handlers_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lucas-gaitzsch/pdf-turtle/config"
	"github.com/lucas-gaitzsch/pdf-turtle/server"
	"github.com/lucas-gaitzsch/pdf-turtle/services/bundles"

	"github.com/google/uuid"
)

type fileList []struct {
	Name    string
	Content string
}

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
	bundleService := bundles.NewBundleProviderService(bStore)
	ctx = context.WithValue(ctx, config.ContextKeyBundleProviderService, bundleService)

	s := &server.Server{}
	s.Serve(ctx)

	t.Cleanup(func() {
		s.Close(ctx)
	})

	files := fileList{
		{Name: "index.html", Content: "<html><body><h1>Test {{ .Name }}</h1></body></html>"},
		{Name: "footer.html", Content: "<footer>Test Footer</footer>"},
	}

	zipBuf := createZipFileBuffer(files)

	t.Run("Should save a html bundle", func(t *testing.T) {
		metadata := map[string]string{
			"name":           "test-save-bundle",
			"templateEngine": "golang",
		}
		buf, contentType, err := createRequestBody(zipBuf, metadata)
		if err != nil {
			t.Fatalf("error creating request body: %v", err)
		}

		req := httptest.NewRequest("POST", "/api/html-bundle/save", buf)
		req.Header.Set("Content-Type", contentType)

		resp, err := s.Instance.Test(req, -1)
		if err != nil {
			t.Fatalf("error sending request: %v", err)
		}

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected status code to be 201, got %d", resp.StatusCode)
		}

		response := struct {
			ID string `json:"id"`
		}{}

		_ = json.NewDecoder(resp.Body).Decode(&response)
		if response.ID == "" {
			t.Error("expected id to be set")
		}

		id, err := uuid.Parse(response.ID)
		if err != nil {
			t.Fatalf("error parsing id: %v", err)
		}

		info, err := bStore.GetFromStore(ctx, id)
		if err != nil {
			t.Fatalf("error getting bundle from store: %v", err)
		}
		defer func(Data io.ReadCloser) {
			_ = Data.Close()
		}(info.Data)

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

		if info.Id != response.ID {
			t.Errorf("expected id to be %s, got %s", id, info.Id)
		}

		data := new(bytes.Buffer)
		_, _ = data.ReadFrom(info.Data)

		if bytes.Compare(zipBuf.Bytes(), data.Bytes()) != 0 {
			t.Error("expected data to be the same")
		}
	})

	t.Run("Should update a html bundle", func(t *testing.T) {
		metadata := map[string]string{
			"name":           "test-update-bundle",
			"templateEngine": "golang",
		}
		buf, contentType, err := createRequestBody(zipBuf, metadata)
		if err != nil {
			t.Fatalf("error creating request body: %v", err)
		}
		req := httptest.NewRequest("POST", "/api/html-bundle/save", buf)
		req.Header.Set("Content-Type", contentType)

		resp, err := s.Instance.Test(req, -1)
		if err != nil {
			t.Fatalf("error sending request: %v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected status code to be 201, got %d", resp.StatusCode)
		}

		response := struct {
			ID string `json:"id"`
		}{}
		_ = json.NewDecoder(resp.Body).Decode(&response)
		if response.ID == "" {
			t.Fatalf("expected id to be set")
		}

		f := fileList{
			{Name: "index.html", Content: "<html><body><h1>Test {{ .Name }}</h1></body></html>"},
			{Name: "footer.html", Content: "<footer>Test Footer</footer>"},
			{Name: "header.html", Content: "<header>Test Header</header>"},
		}

		newZipBuf := createZipFileBuffer(f)
		metadata["id"] = response.ID
		buf, contentType, err = createRequestBody(newZipBuf, metadata)
		if err != nil {
			t.Fatalf("error creating request body: %v", err)
		}

		req = httptest.NewRequest("POST", "/api/html-bundle/save", buf)
		req.Header.Set("Content-Type", contentType)

		resp, err = s.Instance.Test(req, -1)
		if err != nil {
			t.Fatalf("error sending request: %v", err)
		}

		if resp.StatusCode != http.StatusCreated {
			t.Errorf("expected status code to be 200, got %d", resp.StatusCode)
		}

		_ = json.NewDecoder(resp.Body).Decode(&response)

		id, err := uuid.Parse(response.ID)
		if err != nil {
			t.Fatalf("error parsing id: %v", err)
		}
		info, err := bStore.GetFromStore(ctx, id)
		if err != nil {
			t.Fatalf("error getting bundle from store: %v", err)
		}

		if info.Name != "test-update-bundle" {
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

		if info.Size != int64(newZipBuf.Len()) {
			t.Errorf("expected size to be %d, got %d", newZipBuf.Len(), info.Size)
		}

		if info.Id != response.ID {
			t.Errorf("expected id to be %s, got %s", id, info.Id)
		}

		data := new(bytes.Buffer)
		_, _ = data.ReadFrom(info.Data)

		if bytes.Compare(newZipBuf.Bytes(), data.Bytes()) != 0 {
			t.Error("expected data to be the same")
		}
		_ = info.Data.Close()
	})
}

func createRequestBody(zipBuf *bytes.Buffer, metadata map[string]string) (*bytes.Buffer, string, error) {
	buf := new(bytes.Buffer)
	w := multipart.NewWriter(buf)

	ff, err := w.CreateFormFile("bundle", "bundle.zip")
	if err != nil {
		return nil, "", err
	}
	if _, err = ff.Write(zipBuf.Bytes()); err != nil {
		return nil, "", err
	}

	for key, value := range metadata {
		if err = w.WriteField(key, value); err != nil {
			return nil, "", err
		}
	}
	_ = w.Close()
	return buf, w.FormDataContentType(), err
}

func createZipFileBuffer(files fileList) *bytes.Buffer {
	zipBuf := new(bytes.Buffer)
	zipWriter := zip.NewWriter(zipBuf)

	for _, file := range files {
		f, _ := zipWriter.Create(file.Name)
		f.Write([]byte(file.Content))
	}

	_ = zipWriter.Close()
	return zipBuf
}
