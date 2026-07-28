package cache

import (
	"context"
	"sync"
)

type Loader func(context.Context) ([]byte, error)

type Singleflight struct {
	mu       sync.Mutex
	inflight map[string]*call
}

type call struct {
	wg    sync.WaitGroup
	value []byte
	err   error
}

func NewSingleflight() *Singleflight {
	return &Singleflight{inflight: map[string]*call{}}
}

func (s *Singleflight) Do(ctx context.Context, key string, loader Loader) ([]byte, error) {
	s.mu.Lock()
	if existing, ok := s.inflight[key]; ok {
		s.mu.Unlock()
		existing.wg.Wait()
		return append([]byte(nil), existing.value...), existing.err
	}
	current := &call{}
	current.wg.Add(1)
	s.inflight[key] = current
	s.mu.Unlock()

	current.value, current.err = loader(ctx)
	current.value = append([]byte(nil), current.value...)
	current.wg.Done()

	s.mu.Lock()
	delete(s.inflight, key)
	s.mu.Unlock()

	return append([]byte(nil), current.value...), current.err
}
