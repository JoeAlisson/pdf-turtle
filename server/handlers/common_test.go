package handlers_test

import (
	"archive/zip"
	"bytes"
	"mime/multipart"
	"os"
)

type fileList []struct {
	Name    string
	Content string
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
