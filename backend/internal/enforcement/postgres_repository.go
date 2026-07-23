package enforcement

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) Create(ctx context.Context, policy Policy) (Policy, error) {
	policy = normalize(policy)
	row := r.pool.QueryRow(ctx, `
INSERT INTO diting_enforcement_policies (
    id, name, description, template, mode, enabled, target_hosts, definition, yaml,
    deployment_status, deployment_message, created_at, updated_at
)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW(), NOW())
RETURNING id::text, name, description, template, mode, enabled, target_hosts, definition, yaml, deployment_status, deployment_message, deployed_at, created_at, updated_at
`, policy.Name, policy.Description, policy.Template, policy.Mode, policy.Enabled, policy.TargetHosts, policy.Definition, policy.YAML, policy.DeploymentStatus, policy.DeploymentMessage)
	return scanPolicy(row)
}

func (r *PostgresRepository) List(ctx context.Context) ([]Policy, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, name, description, template, mode, enabled, target_hosts, definition, yaml, deployment_status, deployment_message, deployed_at, created_at, updated_at
FROM diting_enforcement_policies
ORDER BY updated_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	policies := []Policy{}
	for rows.Next() {
		policy, err := scanPolicy(rows)
		if err != nil {
			return nil, err
		}
		policies = append(policies, policy)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return policies, nil
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (Policy, error) {
	row := r.pool.QueryRow(ctx, `
SELECT id::text, name, description, template, mode, enabled, target_hosts, definition, yaml, deployment_status, deployment_message, deployed_at, created_at, updated_at
FROM diting_enforcement_policies
WHERE id = $1
`, id)
	policy, err := scanPolicy(row)
	if err != nil {
		return Policy{}, mapNotFound(err)
	}
	return policy, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id string, policy Policy) (Policy, error) {
	policy = normalize(policy)
	row := r.pool.QueryRow(ctx, `
UPDATE diting_enforcement_policies
SET name = $2,
    description = $3,
    template = $4,
    mode = $5,
    enabled = $6,
    target_hosts = $7,
    definition = $8,
    yaml = $9,
    deployment_status = $10,
    deployment_message = $11,
    updated_at = NOW()
WHERE id = $1
RETURNING id::text, name, description, template, mode, enabled, target_hosts, definition, yaml, deployment_status, deployment_message, deployed_at, created_at, updated_at
`, id, policy.Name, policy.Description, policy.Template, policy.Mode, policy.Enabled, policy.TargetHosts, policy.Definition, policy.YAML, policy.DeploymentStatus, policy.DeploymentMessage)
	updated, err := scanPolicy(row)
	if err != nil {
		return Policy{}, mapNotFound(err)
	}
	return updated, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	commandTag, err := r.pool.Exec(ctx, `DELETE FROM diting_enforcement_policies WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) UpdateDeployment(ctx context.Context, id string, status string, message string) (Policy, error) {
	row := r.pool.QueryRow(ctx, `
UPDATE diting_enforcement_policies
SET deployment_status = $2,
    deployment_message = $3,
    deployed_at = CASE WHEN $2 = 'deployed' THEN NOW() ELSE deployed_at END,
    updated_at = NOW()
WHERE id = $1
RETURNING id::text, name, description, template, mode, enabled, target_hosts, definition, yaml, deployment_status, deployment_message, deployed_at, created_at, updated_at
`, id, normalizeDeploymentStatus(status), message)
	updated, err := scanPolicy(row)
	if err != nil {
		return Policy{}, mapNotFound(err)
	}
	return updated, nil
}

type policyScanner interface {
	Scan(dest ...any) error
}

func scanPolicy(scanner policyScanner) (Policy, error) {
	var policy Policy
	var createdAt time.Time
	var updatedAt time.Time
	var deployedAt *time.Time
	if err := scanner.Scan(
		&policy.ID,
		&policy.Name,
		&policy.Description,
		&policy.Template,
		&policy.Mode,
		&policy.Enabled,
		&policy.TargetHosts,
		&policy.Definition,
		&policy.YAML,
		&policy.DeploymentStatus,
		&policy.DeploymentMessage,
		&deployedAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Policy{}, err
	}
	policy.DeployedAt = deployedAt
	policy.CreatedAt = createdAt
	policy.UpdatedAt = updatedAt
	return policy, nil
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
