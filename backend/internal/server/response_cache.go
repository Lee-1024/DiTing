package server

import (
	"bytes"
	"net/http"
	"time"

	"diting/backend/internal/cache"
)

func responseCache(store cache.Cache, namespace string, ttl time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if store == nil || ttl <= 0 {
			return next
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				next.ServeHTTP(w, r)
				return
			}
			key := cache.Key(namespace, map[string]string{
				"path":  r.URL.Path,
				"query": r.URL.RawQuery,
			})
			if data, ok, err := store.Get(r.Context(), key); err == nil && ok {
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("X-DiTing-Cache", "HIT")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
				return
			}

			recorder := &cacheRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(recorder, r)
			if recorder.status >= http.StatusOK && recorder.status < http.StatusMultipleChoices {
				_ = store.Set(r.Context(), key, recorder.body.Bytes(), ttl)
			}
		})
	}
}

type cacheRecorder struct {
	http.ResponseWriter
	status int
	body   bytes.Buffer
}

func (r *cacheRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
	r.ResponseWriter.WriteHeader(statusCode)
}

func (r *cacheRecorder) Write(data []byte) (int, error) {
	r.body.Write(data)
	return r.ResponseWriter.Write(data)
}
