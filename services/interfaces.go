package services

import (
	"context"
	"io"

	"github.com/google/uuid"
	"github.com/lucas-gaitzsch/pdf-turtle/models"
	"github.com/lucas-gaitzsch/pdf-turtle/services/bundles"
)

type AssetsProviderService interface {
	GetMergedCss() *string
	GetCssByKey(key string) (css *string, ok bool)
}

type BundleProviderService interface {
	Provide(bundle *bundles.Bundle) (id uuid.UUID, cleanup bundles.CleanupFunc)
	Remove(id uuid.UUID)
	GetById(id uuid.UUID) (bundles.BundleReader, bool)
	Save(ctx context.Context, info bundles.BundleInfo) (uuid.UUID, error)
	DeleteFromStore(ctx context.Context, id uuid.UUID) error
	GetFromStore(ctx context.Context, id uuid.UUID) (bundles.BundleInfo, error)
}

type RendererBackgroundService interface {
	Init(outerCtx context.Context)
	RenderAndReceive(job models.Job) (io.Reader, error)
	Close()
}
