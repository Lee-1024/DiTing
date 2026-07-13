package rule

import (
	"context"
	"encoding/json"
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

func (r *PostgresRepository) Create(ctx context.Context, rule Rule) (Rule, error) {
	matchExpr, tags, err := marshalRuleJSON(rule)
	if err != nil {
		return Rule{}, err
	}

	row := r.pool.QueryRow(ctx, `
INSERT INTO diting_audit_rules (id, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at)
VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6, $7::jsonb, $8::jsonb, NOW(), NOW())
RETURNING id::text, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at
`, rule.Name, rule.Description, rule.EventType, rule.Enabled, rule.Severity, rule.RiskScore, string(matchExpr), string(tags))

	return scanRule(row)
}

func (r *PostgresRepository) List(ctx context.Context) ([]Rule, error) {
	rows, err := r.pool.Query(ctx, `
SELECT id::text, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at
FROM diting_audit_rules
ORDER BY updated_at DESC
`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rules := []Rule{}
	for rows.Next() {
		rule, err := scanRule(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rules, nil
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (Rule, error) {
	row := r.pool.QueryRow(ctx, `
SELECT id::text, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at
FROM diting_audit_rules
WHERE id = $1
`, id)
	rule, err := scanRule(row)
	if err != nil {
		return Rule{}, mapNotFound(err)
	}
	return rule, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id string, rule Rule) (Rule, error) {
	matchExpr, tags, err := marshalRuleJSON(rule)
	if err != nil {
		return Rule{}, err
	}
	row := r.pool.QueryRow(ctx, `
UPDATE diting_audit_rules
SET name = $2,
    description = $3,
    event_type = $4,
    enabled = $5,
    severity = $6,
    risk_score = $7,
    match_expr = $8::jsonb,
    tags = $9::jsonb,
    updated_at = NOW()
WHERE id = $1
RETURNING id::text, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at
`, id, rule.Name, rule.Description, rule.EventType, rule.Enabled, rule.Severity, rule.RiskScore, string(matchExpr), string(tags))
	updated, err := scanRule(row)
	if err != nil {
		return Rule{}, mapNotFound(err)
	}
	return updated, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id string) error {
	commandTag, err := r.pool.Exec(ctx, `DELETE FROM diting_audit_rules WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *PostgresRepository) CountEnabledRules(ctx context.Context) (uint64, error) {
	var count uint64
	if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM diting_audit_rules WHERE enabled = true`).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

type ruleScanner interface {
	Scan(dest ...any) error
}

func scanRule(scanner ruleScanner) (Rule, error) {
	var rule Rule
	var matchExpr []byte
	var tags []byte
	var createdAt time.Time
	var updatedAt time.Time
	if err := scanner.Scan(
		&rule.ID,
		&rule.Name,
		&rule.Description,
		&rule.EventType,
		&rule.Enabled,
		&rule.Severity,
		&rule.RiskScore,
		&matchExpr,
		&tags,
		&createdAt,
		&updatedAt,
	); err != nil {
		return Rule{}, err
	}
	if err := json.Unmarshal(matchExpr, &rule.MatchExpr); err != nil {
		return Rule{}, err
	}
	if err := json.Unmarshal(tags, &rule.Tags); err != nil {
		return Rule{}, err
	}
	rule.CreatedAt = createdAt
	rule.UpdatedAt = updatedAt
	return rule, nil
}

func marshalRuleJSON(rule Rule) ([]byte, []byte, error) {
	matchExpr, err := json.Marshal(rule.MatchExpr)
	if err != nil {
		return nil, nil, err
	}
	tags := rule.Tags
	if tags == nil {
		tags = []string{}
	}
	tagsJSON, err := json.Marshal(tags)
	if err != nil {
		return nil, nil, err
	}
	return matchExpr, tagsJSON, nil
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
