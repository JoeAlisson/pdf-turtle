package config

import (
	"context"

	"github.com/lucas-gaitzsch/pdf-turtle/utils"

	"github.com/rs/zerolog/log"
)

type Config struct {
	LogLevelDebug          bool `arg:"--logDebug,env:LOG_LEVEL_DEBUG" default:"false" help:"Debug log level active"`
	LogJsonOutput          bool `arg:"--logJsonOutput,env:LOG_JSON_OUTPUT" default:"false" help:"Json log output"`
	RenderTimeoutInSeconds int  `arg:"--renderTimeout,env:RENDER_TIMEOUT" default:"30" help:"Render timeout in seconds"`
	WorkerInstances        int  `arg:"--workerInstances,env:WORKER_INSTANCES" default:"30"`

	Port                         int    `arg:"env" default:"8000" help:"Server port"`
	GracefulShutdownTimeoutInSec int    `arg:"--GracefulShutdownTimeout,env:GRACEFUL_SHUTDOWN_TIMEOUT" default:"10" help:"Graceful server shutdown timeout in seconds"`
	MaxBodySizeInMb              int    `arg:"--maxBodySize,env:MAX_BODY_SIZE" default:"32" help:"Max body size in megabyte"`
	ServePlayground              bool   `arg:"--servePlayground,env:SERVE_PLAYGROUND" default:"false" help:"Serve playground from path './static-files/playground/'"`
	Secret                       string `arg:"env" default:"" help:"Secret used as bearer token"`
	NoSandbox                    bool   `arg:"--no-sandbox,env:NO_SANDBOX" default:"false" help:"Disable chromium sandbox"`

	PreloadedAssets []string `arg:"env" help:"Preload assets on startup. Example:'bar.js:https://foo.com/bar.js'"` //TODO:!

	LoopbackPort int `arg:"env" default:"8001" help:"Loopback-Server port"`

	S3Endpoint  string `arg:"--s3-endpoint,env:S3_ENDPOINT" default:"" help:"S3 endpoint"`
	S3AccessKey string `arg:"--s3-accessKey,env:S3_ACCESS_KEY" default:"" help:"S3 access key"`
	S3SecretKey string `arg:"--s3-secretKey,env:S3_SECRET_KEY" default:"" help:"S3 secret key"`
	S3Bucket    string `arg:"--s3-bucket,env:S3_BUCKET" default:"" help:"S3 bucket"`
	S3Region    string `arg:"--s3-region,env:S3_REGION" default:"us-east-1" help:"S3 region"`
	S3Secure    bool   `arg:"--s3-secure,env:S3_SECURE" default:"true" help:"S3 secure"`
}

func ContextWithConfig(parentCtx context.Context, config Config) context.Context {
	return context.WithValue(parentCtx, ContextKeyConfig, config)
}

func Get(ctx context.Context) Config {
	c, hasConfig := ctx.Value(ContextKeyConfig).(Config)

	if hasConfig {
		return c
	} else {
		log.Warn().Msg("no config was set -> fallback to default")

		c := &Config{}
		utils.ReflectDefaultValues(c)
		return *c
	}
}
