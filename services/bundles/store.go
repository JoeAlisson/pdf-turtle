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

type MinioStore struct {
	client *minio.Client
	bucket string
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

	if err = ensureBucketExists(mc, opt); err != nil {
		return MinioStore{}, err

	}
	return MinioStore{
		client: mc,
		bucket: opt.Bucket,
	}, nil
}

func ensureBucketExists(mc *minio.Client, opt MinioOptions) error {
	exists, err := mc.BucketExists(context.Background(), opt.Bucket)
	if err != nil {
		return err
	}
	if !exists {
		err = mc.MakeBucket(context.Background(), opt.Bucket, minio.MakeBucketOptions{
			Region: opt.Region,
		})
	}
	return err
}

func (m MinioStore) Save(ctx context.Context, info Info) (uuid.UUID, error) {
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

func (m MinioStore) Delete(ctx context.Context, id uuid.UUID) error {
	return m.client.RemoveObject(ctx, m.bucket, bundlePath+id.String(), minio.RemoveObjectOptions{})
}

func (m MinioStore) Get(ctx context.Context, id uuid.UUID) (Info, error) {
	obj, err := m.client.GetObject(ctx, m.bucket, bundlePath+id.String(), minio.GetObjectOptions{})
	if err != nil {
		return Info{}, err
	}

	info, err := obj.Stat()
	if err != nil {
		return Info{}, err
	}

	return Info{
		Id:             id.String(),
		Name:           info.UserMetadata["Name"],
		TemplateEngine: info.UserMetadata["Template-Engine"],
		FileName:       info.UserMetadata["File-Name"],
		Data:           obj,
		Size:           info.Size,
		ContentType:    info.ContentType,
	}, nil
}

func (m MinioStore) ListInfo(ctx context.Context) (InfoList, error) {
	infoChan := m.client.ListObjects(ctx, m.bucket, minio.ListObjectsOptions{
		WithMetadata: true,
		Prefix:       bundlePath,
	})

	var list InfoList
	for obj := range infoChan {
		if obj.Err != nil {
			return list, obj.Err
		}
		list.Items = append(list.Items, Info{
			Id:   obj.UserMetadata["X-Amz-Meta-Id"],
			Name: obj.UserMetadata["X-Amz-Meta-Name"],
		})
	}
	return list, nil
}
