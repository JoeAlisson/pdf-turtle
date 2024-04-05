package bundles

import (
	"context"
	"strconv"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const bundlePath = "bundles/"

type MinioOptions struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Region    string
	UseSSL    bool
}

func NewMinioStore(opt MinioOptions) (MinioStore, error) {
	mc, err := minio.New(opt.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(opt.AccessKey, opt.SecretKey, ""),
		Secure: opt.UseSSL,
		Region: opt.Region,
	})
	if err != nil {
		return MinioStore{}, err
	}
	return MinioStore{
		client: mc,
		bucket: opt.Bucket,
	}, nil
}

type MinioStore struct {
	client *minio.Client
	bucket string
}

func (m MinioStore) Save(ctx context.Context, info BundleInfo) (uuid.UUID, error) {
	id := parseId(info.Id)
	opt := minio.PutObjectOptions{
		ContentType: info.ContentType,
		UserMetadata: map[string]string{
			"Id":              id.String(),
			"Name":            info.Name,
			"File-Name":       info.FileName,
			"Size":            strconv.FormatInt(info.Size, 10),
			"Template-Engine": info.TemplateEngine,
		},
	}

	_, err := m.client.PutObject(ctx, m.bucket, bundlePath+id.String(), info.Data, info.Size, opt)
	if err != nil {
		return uuid.Nil, err
	}
	return id, nil
}

func parseId(id string) uuid.UUID {
	if r, err := uuid.Parse(id); err == nil {
		return r
	}
	return uuid.New()
}

func (m MinioStore) DeleteFromStore(ctx context.Context, id uuid.UUID) error {
	return m.client.RemoveObject(ctx, m.bucket, bundlePath+id.String(), minio.RemoveObjectOptions{})
}

func (m MinioStore) GetFromStore(ctx context.Context, id uuid.UUID) (BundleInfo, error) {
	obj, err := m.client.GetObject(ctx, m.bucket, bundlePath+id.String(), minio.GetObjectOptions{})
	if err != nil {
		return BundleInfo{}, err
	}

	info, err := obj.Stat()
	if err != nil {
		return BundleInfo{}, err
	}

	return BundleInfo{
		Id:             id.String(),
		Name:           info.UserMetadata["Name"],
		TemplateEngine: info.UserMetadata["Template-Engine"],
		FileName:       info.UserMetadata["File-Name"],
		Data:           obj,
		Size:           info.Size,
		ContentType:    info.ContentType,
	}, nil
}
