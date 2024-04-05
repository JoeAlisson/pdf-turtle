package bundles

import (
	"context"
	"io"
	"sync"

	"github.com/google/uuid"
)

type CleanupFunc = func()

type BundleInfo struct {
	Id             string
	Name           string
	TemplateEngine string
	Data           io.ReadCloser
	Size           int64
	ContentType    string
	FileName       string
}

type Store interface {
	Save(ctx context.Context, info BundleInfo) (uuid.UUID, error)
	DeleteFromStore(ctx context.Context, id uuid.UUID) error
	GetFromStore(ctx context.Context, id uuid.UUID) (BundleInfo, error)
}

func NewBundleProviderService(s Store) *BundleProviderService {
	bps := &BundleProviderService{
		bundles: make(map[uuid.UUID]*Bundle),
		Store:   s,
	}
	return bps
}

type BundleProviderService struct {
	bundles map[uuid.UUID]*Bundle
	lock    sync.RWMutex
	Store   Store
}

func (bps *BundleProviderService) Provide(bundle *Bundle) (id uuid.UUID, cleanup CleanupFunc) {
	if bundle == nil {
		bundle = &Bundle{}
	}

	bps.lock.Lock()
	defer bps.lock.Unlock()

	id = uuid.New()
	bps.bundles[id] = bundle

	cleanup = func() {
		bps.Remove(id)
	}

	return
}

func (bps *BundleProviderService) Remove(id uuid.UUID) {
	bps.lock.Lock()
	defer bps.lock.Unlock()

	delete(bps.bundles, id)
}

func (bps *BundleProviderService) GetById(id uuid.UUID) (BundleReader, bool) {
	bps.lock.RLock()
	defer bps.lock.RUnlock()

	b, ok := bps.bundles[id]

	return b, ok
}

func (bps *BundleProviderService) Save(ctx context.Context, info BundleInfo) (uuid.UUID, error) {
	return bps.Store.Save(ctx, info)
}

func (bps *BundleProviderService) DeleteFromStore(ctx context.Context, id uuid.UUID) error {
	return bps.Store.DeleteFromStore(ctx, id)
}

func (bps *BundleProviderService) GetFromStore(ctx context.Context, id uuid.UUID) (BundleInfo, error) {
	return bps.Store.GetFromStore(ctx, id)
}
