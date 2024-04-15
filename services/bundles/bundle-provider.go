package bundles

import (
	"context"
	"io"
	"sync"

	"github.com/google/uuid"
)

type CleanupFunc = func()

type InfoList struct {
	Items []Info
}

type InfoReader interface {
	io.Reader
	io.ReaderAt
}

type Info struct {
	Id             string     `json:"id,omitempty"`
	Name           string     `json:"name,omitempty"`
	TemplateEngine string     `json:"templateEngine,omitempty"`
	Data           InfoReader `json:"data,omitempty"`
	Size           int64      `json:"size,omitempty"`
	ContentType    string     `json:"contentType,omitempty"`
	FileName       string     `json:"fileName,omitempty"`
}

type Store interface {
	Save(ctx context.Context, info Info) (uuid.UUID, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Get(ctx context.Context, id uuid.UUID) (Info, error)
	ListInfo(ctx context.Context) (InfoList, error)
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

func (bps *BundleProviderService) Save(ctx context.Context, info Info) (uuid.UUID, error) {
	return bps.Store.Save(ctx, info)
}

func (bps *BundleProviderService) DeleteFromStore(ctx context.Context, id uuid.UUID) error {
	return bps.Store.Delete(ctx, id)
}

func (bps *BundleProviderService) GetFromStore(ctx context.Context, id uuid.UUID) (Info, error) {
	return bps.Store.Get(ctx, id)
}

func (bps *BundleProviderService) ListInfoFromStore(ctx context.Context) (InfoList, error) {
	return bps.Store.ListInfo(ctx)
}
