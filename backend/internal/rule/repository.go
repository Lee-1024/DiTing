package rule

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Rule struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	EventType   string     `json:"eventType"`
	Enabled     bool       `json:"enabled"`
	Severity    string     `json:"severity"`
	RiskScore   int        `json:"riskScore"`
	MatchExpr   Expression `json:"matchExpr"`
	Tags        []string   `json:"tags"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type Repository interface {
	Create(ctx context.Context, rule Rule) (Rule, error)
	List(ctx context.Context) ([]Rule, error)
	Get(ctx context.Context, id string) (Rule, error)
	Update(ctx context.Context, id string, rule Rule) (Rule, error)
	Delete(ctx context.Context, id string) error
	CountEnabledRules(ctx context.Context) (uint64, error)
}

var ErrNotFound = errors.New("rule not found")

type MemoryRepository struct {
	mu    sync.Mutex
	rules []Rule
	next  int
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{next: 1, rules: []Rule{}}
}

func (r *MemoryRepository) Create(_ context.Context, rule Rule) (Rule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	rule.ID = fmt.Sprintf("rule-%d", r.next)
	rule.CreatedAt = now
	rule.UpdatedAt = now
	if !rule.Enabled {
		rule.Enabled = true
	}
	r.next++
	r.rules = append(r.rules, rule)
	return rule, nil
}

func (r *MemoryRepository) List(_ context.Context) ([]Rule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]Rule, len(r.rules))
	copy(result, r.rules)
	return result, nil
}

func (r *MemoryRepository) Get(_ context.Context, id string) (Rule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, rule := range r.rules {
		if rule.ID == id {
			return rule, nil
		}
	}
	return Rule{}, ErrNotFound
}

func (r *MemoryRepository) Update(_ context.Context, id string, next Rule) (Rule, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index, existing := range r.rules {
		if existing.ID != id {
			continue
		}
		next.ID = id
		next.CreatedAt = existing.CreatedAt
		next.UpdatedAt = time.Now().UTC()
		r.rules[index] = next
		return next, nil
	}
	return Rule{}, ErrNotFound
}

func (r *MemoryRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index, rule := range r.rules {
		if rule.ID != id {
			continue
		}
		r.rules = append(r.rules[:index], r.rules[index+1:]...)
		return nil
	}
	return ErrNotFound
}

func (r *MemoryRepository) CountEnabledRules(_ context.Context) (uint64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var count uint64
	for _, rule := range r.rules {
		if rule.Enabled {
			count++
		}
	}
	return count, nil
}
