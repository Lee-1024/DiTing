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

type Deployment struct {
	ID         string     `json:"id"`
	PolicyID   string     `json:"policyId"`
	HostID     string     `json:"hostId"`
	HostName   string     `json:"hostName"`
	Status     string     `json:"status"`
	Message    string     `json:"message"`
	DeployedAt *time.Time `json:"deployedAt,omitempty"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type Repository interface {
	Create(ctx context.Context, policy Policy) (Policy, error)
	List(ctx context.Context) ([]Policy, error)
	Get(ctx context.Context, id string) (Policy, error)
	Update(ctx context.Context, id string, policy Policy) (Policy, error)
	Delete(ctx context.Context, id string) error
	UpdateDeployment(ctx context.Context, id string, status string, message string) (Policy, error)
	EmergencyDisable(ctx context.Context, message string) (int, error)
	UpsertHostDeployment(ctx context.Context, deployment Deployment) (Deployment, error)
	ListHostDeployments(ctx context.Context, policyID string) ([]Deployment, error)
}

var ErrNotFound = errors.New("enforcement policy not found")

type MemoryRepository struct {
	mu          sync.Mutex
	policies    []Policy
	deployments []Deployment
	next        int
	nextDeploy  int
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{next: 1, nextDeploy: 1, policies: []Policy{}, deployments: []Deployment{}}
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

func (r *MemoryRepository) EmergencyDisable(_ context.Context, message string) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now().UTC()
	count := 0
	for index, policy := range r.policies {
		if !policy.Enabled && policy.Mode == "disabled" && policy.DeploymentStatus == "disabled" {
			continue
		}
		policy.Enabled = false
		policy.Mode = "disabled"
		policy.DeploymentStatus = "disabled"
		policy.DeploymentMessage = message
		policy.UpdatedAt = now
		r.policies[index] = policy
		count++
	}
	return count, nil
}

func (r *MemoryRepository) UpsertHostDeployment(_ context.Context, deployment Deployment) (Deployment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.hasPolicy(deployment.PolicyID) {
		return Deployment{}, ErrNotFound
	}
	now := time.Now().UTC()
	deployment.Status = normalizeDeploymentStatus(deployment.Status)
	for index, existing := range r.deployments {
		if existing.PolicyID == deployment.PolicyID && existing.HostID == deployment.HostID {
			deployment.ID = existing.ID
			deployment.UpdatedAt = now
			if deployment.Status == "deployed" {
				deployment.DeployedAt = &now
			} else {
				deployment.DeployedAt = existing.DeployedAt
			}
			r.deployments[index] = deployment
			return deployment, nil
		}
	}
	deployment.ID = fmt.Sprintf("enforcement-deployment-%d", r.nextDeploy)
	deployment.UpdatedAt = now
	if deployment.Status == "deployed" {
		deployment.DeployedAt = &now
	}
	r.nextDeploy++
	r.deployments = append(r.deployments, deployment)
	return deployment, nil
}

func (r *MemoryRepository) ListHostDeployments(_ context.Context, policyID string) ([]Deployment, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.hasPolicy(policyID) {
		return nil, ErrNotFound
	}
	result := []Deployment{}
	for _, deployment := range r.deployments {
		if deployment.PolicyID == policyID {
			result = append(result, deployment)
		}
	}
	return result, nil
}

func (r *MemoryRepository) hasPolicy(id string) bool {
	for _, policy := range r.policies {
		if policy.ID == id {
			return true
		}
	}
	return false
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
