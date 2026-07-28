package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
	"time"
)

type Cache interface {
	Get(ctx context.Context, key string) ([]byte, bool, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	DeletePrefix(ctx context.Context, prefix string) error
}

func Key(namespace string, values map[string]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var builder strings.Builder
	builder.WriteString(namespace)
	for _, key := range keys {
		builder.WriteByte('\n')
		builder.WriteString(key)
		builder.WriteByte('=')
		builder.WriteString(values[key])
	}
	sum := sha256.Sum256([]byte(builder.String()))
	return namespace + ":" + hex.EncodeToString(sum[:])
}
