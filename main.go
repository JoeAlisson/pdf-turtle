package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/lucas-gaitzsch/pdf-turtle/config"
	"github.com/lucas-gaitzsch/pdf-turtle/loopback"
	"github.com/lucas-gaitzsch/pdf-turtle/server"
	"github.com/lucas-gaitzsch/pdf-turtle/services"
	"github.com/lucas-gaitzsch/pdf-turtle/services/assetsprovider"
	"github.com/lucas-gaitzsch/pdf-turtle/services/bundles"
	"github.com/lucas-gaitzsch/pdf-turtle/services/renderer"
	"github.com/lucas-gaitzsch/pdf-turtle/utils/logging"

	"github.com/rs/zerolog/log"

	"github.com/alexflint/go-arg"
)

func initConfigByArgs(ctx context.Context) context.Context {
	var c config.Config
	arg.MustParse(&c)

	return config.ContextWithConfig(ctx, c)
}

func initServicesCtx(ctx context.Context) context.Context {
	servicesCtx := ctx

	rendererService := renderer.NewRendererBackgroundService(ctx)
	servicesCtx = context.WithValue(servicesCtx, config.ContextKeyRendererService, rendererService)

	assetsProviderService := assetsprovider.NewAssetsProviderService()
	servicesCtx = context.WithValue(servicesCtx, config.ContextKeyAssetsProviderService, assetsProviderService)

	c := config.Get(ctx)

	bStore, err := bundles.NewMinioStore(bundles.MinioOptions{
		Endpoint:  c.S3Endpoint,
		AccessKey: c.S3AccessKey,
		SecretKey: c.S3SecretKey,
		Bucket:    c.S3Bucket,
		Region:    c.S3Region,
		UseSSL:    c.S3Secure,
	})
	if err != nil {
		log.Warn().Err(err).Msg("could not connect to object store")
	}
	bundleProviderService := bundles.NewBundleProviderService(bStore)
	servicesCtx = context.WithValue(servicesCtx, config.ContextKeyBundleProviderService, bundleProviderService)

	return servicesCtx
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	ctx = initConfigByArgs(ctx)

	logging.InitLogger(ctx)

	log.Info().Msg("Hey dude 👋 .. I am Karl, your turtle for today 🐢")

	// init services
	servicesCtx := initServicesCtx(ctx)

	// init loopback server
	srvLoopback := loopback.Server{}
	srvLoopback.Serve(servicesCtx)

	// init server
	srv := server.Server{}
	srv.Serve(servicesCtx)

	// listen to os signals
	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, syscall.SIGINT, syscall.SIGTERM)
	<-osSignals

	// cleanup resources
	log.Info().Msg("shutting service down...")
	srv.Close(ctx)
	servicesCtx.
		Value(config.ContextKeyRendererService).(services.RendererBackgroundService).
		Close()
	cancel()

	// exit
	log.Info().Msg("🐢 bye bye dude.")
	os.Exit(0)
}
