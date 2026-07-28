package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	"diting/backend/internal/cache"
)

func responseCache(store cache.Cache, flights *cache.Singleflight, namespace string, ttl time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if store == nil || ttl <= 0 {
			return next
		}
		if flights == nil {
			flights = cache.NewSingleflight()
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
				if writeCachedResponse(w, data) {
					w.Header().Set("X-DiTing-Cache", "HIT")
				}
				return
			}

			result, err := flights.Do(r.Context(), key, func(ctx context.Context) ([]byte, error) {
				if data, ok, err := store.Get(ctx, key); err == nil && ok {
					return data, nil
				}
				recorder := newCacheRecorder()
				next.ServeHTTP(recorder, r)
				response := cachedResponse{
					Status: recorder.status,
					Header: recorder.header,
					Body:   recorder.body.Bytes(),
				}
				encoded, err := json.Marshal(response)
				if err != nil {
					return nil, err
				}
				if recorder.status >= http.StatusOK && recorder.status < http.StatusMultipleChoices {
					_ = store.Set(ctx, key, encoded, ttl)
				}
				return encoded, nil
			})
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}
			if !writeCachedResponse(w, result) {
				next.ServeHTTP(w, r)
			}
		})
	}
}

func writeCachedResponse(w http.ResponseWriter, data []byte) bool {
	var response cachedResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return false
	}
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	if response.Status >= http.StatusOK && response.Status < http.StatusMultipleChoices {
		w.Header().Set("X-DiTing-Cache", "HIT")
	}
	if response.Status == 0 {
		response.Status = http.StatusOK
	}
	w.WriteHeader(response.Status)
	_, _ = w.Write(response.Body)
	return true
}

type cachedResponse struct {
	Status int
	Header http.Header
	Body   []byte
}

type cacheRecorder struct {
	header http.Header
	status int
	body   bytes.Buffer
}

func newCacheRecorder() *cacheRecorder {
	return &cacheRecorder{header: http.Header{}, status: http.StatusOK}
}

func (r *cacheRecorder) Header() http.Header {
	return r.header
}

func (r *cacheRecorder) WriteHeader(statusCode int) {
	r.status = statusCode
}

func (r *cacheRecorder) Write(data []byte) (int, error) {
	return r.body.Write(data)
}
