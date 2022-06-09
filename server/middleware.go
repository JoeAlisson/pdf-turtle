package server

import (
	"context"
	"encoding/json"
	"net/http"
	"pdf-turtle/config"
	"pdf-turtle/models/dto"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

func maxBodySizeMiddleware(maxBodySizeMb int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.Body = http.MaxBytesReader(w, r.Body, int64(maxBodySizeMb)*1024*1024)
			next.ServeHTTP(w, r)
		})
	}
}

func NewResponseWriterMiddleware(w http.ResponseWriter) *responseWriterMiddleware {
	return &responseWriterMiddleware{
		w:          w,
		statusCode: http.StatusOK,
	}
}

type responseWriterMiddleware struct {
	w          http.ResponseWriter
	statusCode int
}

func (wm *responseWriterMiddleware) Header() http.Header {
	return wm.w.Header()
}

func (wm *responseWriterMiddleware) Write(b []byte) (int, error) {
	return wm.w.Write(b)
}

func (wm *responseWriterMiddleware) WriteHeader(statusCode int) {
	wm.statusCode = statusCode
	wm.w.WriteHeader(statusCode)
}

func (wm *responseWriterMiddleware) GetStatus() int {
	if wm.statusCode == 0 {
		return http.StatusOK
	}
	return wm.statusCode
}

func loggingMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			wm := NewResponseWriterMiddleware(w)
			ctx := r.Context()
			path := r.URL.EscapedPath()

			requestUUID := uuid.New()

			requestLogger := log.With().
				Str("method", r.Method).
				Str("path", path).
				Stringer(string(config.ContextKeyRequestId), requestUUID).
				Logger()

			requestLogger.Debug().
				Int64("contentLength", r.ContentLength).
				Msg("start request")

			ctx = requestLogger.WithContext(ctx)
			ctx = context.WithValue(ctx, config.ContextKeyRequestId, requestUUID)

			defer func(begin time.Time) {
				status := wm.GetStatus()
				dur := time.Since(begin)

				requestLogger.Info().
					Dur("executionTime", dur).
					Int("statusCode", status).
					Msgf(
						"[%d %s] %s %s -> took %d ms",
						status,
						http.StatusText(status),
						r.Method,
						path,
						dur.Milliseconds(),
					)
			}(time.Now())

			next.ServeHTTP(wm, r.WithContext(ctx))
		})
	}
}

func recoverMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					ctx := r.Context()

					err, ok := rec.(error)

					errMsg := ""
					if ok {
						errMsg = err.Error()
					} else if errStr, ok := rec.(string); ok {
						errMsg = errStr
					}

					log.Ctx(ctx).
						Error().
						Err(err).
						Stack().
						Interface("err", rec).
						Msg("err during request")

					w.WriteHeader(http.StatusInternalServerError)

					w.Header().Set("Content-Type", "application/json")

					json.NewEncoder(w).Encode(dto.RequestError{
						Msg:       "err during request",
						Err:       errMsg,
						RequestId: ctx.Value(config.ContextKeyRequestId).(uuid.UUID).String(),
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
