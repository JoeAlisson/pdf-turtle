package services

import (
	"context"
	"io"

	"github.com/lucas-gaitzsch/pdf-turtle/models"
	"github.com/lucas-gaitzsch/pdf-turtle/services/bundles"

	"github.com/google/uuid"
)

type AssetsProviderService interface {
	GetMergedCss() *string
	GetCssByKey(key string) (css *string, ok bool)
}

type BundleProviderService interface {
	Provide(bundle *bundles.Bundle) (id uuid.UUID, cleanup bundles.CleanupFunc)
	Remove(id uuid.UUID)
	GetById(id uuid.UUID) (bundles.BundleReader, bool)
	Save(ctx context.Context, info bundles.Info) (uuid.UUID, error)
	DeleteFromStore(ctx context.Context, id uuid.UUID) error
	GetFromStore(ctx context.Context, id uuid.UUID) (bundles.Info, error)
	ListInfoFromStore(ctx context.Context) (bundles.InfoList, error)
}

type RendererBackgroundService interface {
	Init(outerCtx context.Context)
	RenderAndReceive(job models.Job) (io.Reader, error)
	Close()
}
