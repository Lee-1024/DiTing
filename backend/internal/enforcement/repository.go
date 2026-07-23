package enforcement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

type Policy struct {
	ID                string          `json:"id"`
	Name              string          `json:"name"`
	Description       string          `json:"description"`
	Template          string          `json:"template"`
	Mode              string          `json:"mode"`
	Enabled           bool            `json:"enabled"`
	TargetHosts       []string        `json:"targetHosts"`
	Definition        json.RawMessage `json:"definition"`
	YAML              string          `json:"yaml"`
	DeploymentStatus  string          `json:"deploymentStatus"`
	DeploymentMessage string          `json:"deploymentMessage"`
	DeployedAt        *time.Time      `json:"deployedAt,omitempty"`
	CreatedAt         time.Time       `json:"createdAt"`
	UpdatedAt         time.Time       `json:"updatedAt"`
}

type Repository interface {
	Create(ctx context.Context, policy Policy) (Policy, error)
	List(ctx context.Context) ([]Policy, error)
	Get(ctx context.Context, id string) (Policy, error)
	Update(ctx context.Context, id string, policy Policy) (Policy, error)
	Delete(ctx context.Context, id string) error
	UpdateDeployment(ctx context.Context, id string, status string, message string) (Policy, error)
}

var ErrNotFound = errors.New("enforcement policy not found")

type MemoryRepository struct {
	mu       sync.Mutex
	policies []Policy
	next     int
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{next: 1, policies: []Policy{}}
}

func (r *MemoryRepository) Create(_ context.Context, policy Policy) (Policy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	policy.ID = fmt.Sprintf("enforcement-policy-%d", r.next)
	policy.CreatedAt = now
	policy.UpdatedAt = now
	policy = normalize(policy)
	r.next++
	r.policies = append(r.policies, policy)
	return policy, nil
}

func (r *MemoryRepository) List(_ context.Context) ([]Policy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	result := make([]Policy, len(r.policies))
	copy(result, r.policies)
	return result, nil
}

func (r *MemoryRepository) Get(_ context.Context, id string) (Policy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, policy := range r.policies {
		if policy.ID == id {
			return policy, nil
		}
	}
	return Policy{}, ErrNotFound
}

func (r *MemoryRepository) Update(_ context.Context, id string, next Policy) (Policy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index, existing := range r.policies {
		if existing.ID != id {
			continue
		}
		next.ID = id
		next.CreatedAt = existing.CreatedAt
		next.UpdatedAt = time.Now().UTC()
		next.DeployedAt = existing.DeployedAt
		next = normalize(next)
		r.policies[index] = next
		return next, nil
	}
	return Policy{}, ErrNotFound
}

func (r *MemoryRepository) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index, policy := range r.policies {
		if policy.ID != id {
			continue
		}
		r.policies = append(r.policies[:index], r.policies[index+1:]...)
		return nil
	}
	return ErrNotFound
}

func (r *MemoryRepository) UpdateDeployment(_ context.Context, id string, status string, message string) (Policy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for index, policy := range r.policies {
		if policy.ID != id {
			continue
		}
		now := time.Now().UTC()
		policy.DeploymentStatus = normalizeDeploymentStatus(status)
		policy.DeploymentMessage = message
		policy.UpdatedAt = now
		if policy.DeploymentStatus == "deployed" {
			policy.DeployedAt = &now
		}
		r.policies[index] = policy
		return policy, nil
	}
	return Policy{}, ErrNotFound
}

func normalize(policy Policy) Policy {
	if policy.Mode == "" {
		policy.Mode = "audit"
	}
	if policy.DeploymentStatus == "" {
		policy.DeploymentStatus = "draft"
	}
	if len(policy.Definition) == 0 {
		policy.Definition = json.RawMessage(`{}`)
	}
	policy.DeploymentStatus = normalizeDeploymentStatus(policy.DeploymentStatus)
	return policy
}

func normalizeDeploymentStatus(status string) string {
	switch status {
	case "draft", "deployed", "failed", "disabled":
		return status
	default:
		return "draft"
	}
}
