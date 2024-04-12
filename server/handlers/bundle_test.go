//go:build integration

package handlers_test

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
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

	bStore, _ := bundles.NewMinioStore(bundles.MinioOptions{
		Endpoint:  endpoint,
		AccessKey: accessKey,
		SecretKey: secretKey,
		Bucket:    bucketName,
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

		req := httptest.NewRequest("POST", "/api/html-bundle", buf)
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

		info, err := bStore.Get(ctx, id)
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
		req := httptest.NewRequest("POST", "/api/html-bundle", buf)
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

		req = httptest.NewRequest("POST", "/api/html-bundle", buf)
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
		info, err := bStore.Get(ctx, id)
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

func TestGetHtmlBundleHandler(t *testing.T) {
	ctx := context.Background()
	endpoint := getEnvOrDefault("S3_ENDPOINT", "localhost:9000")
	accessKey := getEnvOrDefault("S3_ACCESS_KEY", "minio")
	secretKey := getEnvOrDefault("S3_SECRET_KEY", "minio123")
	bucketName := "test-get-html-bundle"

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

	s := &server.Server{}
	s.Serve(ctx)

	files := fileList{
		{Name: "index.html", Content: "<html><body><h1>Test {{ .Name }}</h1></body></html>"},
		{Name: "footer.html", Content: "<footer>Test Footer</footer>"},
	}

	zipBuf := createZipFileBuffer(files)

	info := bundles.BundleInfo{
		Id:             uuid.NewString(),
		Name:           "pre-save-test-get-bundle",
		TemplateEngine: "golang",
		Data:           io.NopCloser(zipBuf),
		Size:           int64(zipBuf.Len()),
		ContentType:    "application/zip",
		FileName:       "bundle.zip",
	}

	_, err = bStore.Save(ctx, info)
	if err != nil {
		t.Fatalf("error saving bundle: %v", err)
	}

	t.Run("Should get a html bundle", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/html-bundle/"+info.Id, nil)

		resp, err := s.Instance.Test(req, -1)
		if err != nil {
			t.Fatalf("error sending request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status code to be 200, got %d", resp.StatusCode)
		}

		mediaType, params, err := mime.ParseMediaType(resp.Header.Get("Content-Type"))
		if err != nil {
			t.Fatalf("error parsing media type: %v", err)
		}
		if mediaType != "multipart/form-data" {
			t.Errorf("expected media type to be multipart/form-data, got %s", mediaType)
		}

		mw := multipart.NewReader(resp.Body, params["boundary"])
		form, err := mw.ReadForm(32 << 20)
		if err != nil {
			t.Fatalf("error reading form: %v", err)
		}
		fileHeader := form.File["bundle"][0]
		if fileHeader.Filename != "bundle.zip" {
			t.Errorf("expected filename to be bundle.zip, got %s", fileHeader.Filename)
		}

		if fileHeader.Header.Get("Content-Type") != "application/zip" {
			t.Errorf("expected content type to be application/zip, got %s", fileHeader.Header.Get("Content-Type"))
		}

		if fileHeader.Size != info.Size {
			t.Errorf("expected size to be %d, got %d", info.Size, fileHeader.Size)
		}

		f, err := fileHeader.Open()
		if err != nil {
			t.Fatalf("error opening file: %v", err)
		}
		data := new(bytes.Buffer)
		_, err = data.ReadFrom(info.Data)
		if err != nil {
			t.Fatalf("error reading from file: %v", err)
		}
		_ = f.Close()
		if bytes.Compare(zipBuf.Bytes(), data.Bytes()) != 0 {
			t.Error("expected data to be the same")
		}

		name := form.Value["name"][0]
		if name != "pre-save-test-get-bundle" {
			t.Errorf("expected name to be pre-save-test-get-bundle, got %s", name)
		}

		templateEngine := form.Value["templateEngine"][0]
		if templateEngine != "golang" {
			t.Errorf("expected template engine to be golang, got %s", templateEngine)
		}

		id := form.Value["id"][0]
		if id != info.Id {
			t.Errorf("expected id to be %s, got %s", info.Id, id)
		}
	})
}

func TestListHtmlBundleHandler(t *testing.T) {
	ctx := context.Background()
	endpoint := getEnvOrDefault("S3_ENDPOINT", "localhost:9000")
	accessKey := getEnvOrDefault("S3_ACCESS_KEY", "minio")
	secretKey := getEnvOrDefault("S3_SECRET_KEY", "minio123")
	bucketName := "test-list-html-bundle"

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

	s := &server.Server{}
	s.Serve(ctx)

	files := fileList{
		{Name: "index.html", Content: "<html><body><h1>Test {{ .Name }}</h1></body></html>"},
		{Name: "footer.html", Content: "<footer>Test Footer</footer>"},
	}

	zipBuf := createZipFileBuffer(files)

	info := bundles.BundleInfo{
		Id:             uuid.NewString(),
		Name:           "pre-save-test-list-bundle",
		TemplateEngine: "golang",
		Data:           io.NopCloser(zipBuf),
		Size:           int64(zipBuf.Len()),
		ContentType:    "application/zip",
		FileName:       "bundle.zip",
	}

	_, err = bStore.Save(ctx, info)
	if err != nil {
		t.Fatalf("error saving bundle: %v", err)
	}

	info2 := bundles.BundleInfo{
		Id:             uuid.NewString(),
		Name:           "pre-save-test-list-bundle-2",
		TemplateEngine: "golang",
		Data:           io.NopCloser(zipBuf),
		Size:           int64(zipBuf.Len()),
		ContentType:    "application/zip",
	}
	_, err = bStore.Save(ctx, info2)
	if err != nil {
		t.Fatalf("error saving bundle: %v", err)
	}

	t.Cleanup(func() {
		if err = mc.RemoveObject(ctx, bucketName, "bundles",
			minio.RemoveObjectOptions{ForceDelete: true}); err != nil {
			t.Errorf("error removing object: %v", err)
		}
		if err = mc.RemoveBucket(ctx, bucketName); err != nil {
			t.Errorf("error removing bucket: %v", err)
		}
	})

	t.Run("Should list html bundles info", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/html-bundle", nil)

		resp, err := s.Instance.Test(req, -1)
		if err != nil {
			t.Fatalf("error sending request: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status code to be 200, got %d", resp.StatusCode)
		}

		response := struct {
			Items []bundles.BundleInfo `json:"items"`
		}{}

		_ = json.NewDecoder(resp.Body).Decode(&response)

		if len(response.Items) != 2 {
			t.Fatalf("expected items to be 2, got %d", len(response.Items))
		}

		if !expectedBundleInList(response.Items, info) {
			t.Errorf("expected bundle %s not found in list", info.Name)
		}

		if !expectedBundleInList(response.Items, info2) {
			t.Errorf("expected bundle %s not found in list", info2.Name)
		}
	})

}

func expectedBundleInList(items []bundles.BundleInfo, info bundles.BundleInfo) bool {
	for _, item := range items {
		if item.Id == info.Id && item.Name == info.Name {
			return true
		}
	}
	return false
}

func getEnvOrDefault(key, def string) string {
	value := os.Getenv(key)
	if value != "" {
		return value
	}
	return def

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
		_, _ = f.Write([]byte(file.Content))
	}

	_ = zipWriter.Close()
	return zipBuf
}
