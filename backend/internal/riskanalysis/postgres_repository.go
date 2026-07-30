package riskanalysis

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetByEventIDs(ctx context.Context, eventIDs []string) (map[string]Analysis, error) {
	result := map[string]Analysis{}
	if len(eventIDs) == 0 {
		return result, nil
	}
	rows, err := r.pool.Query(ctx, `
SELECT event_id, ai_severity, verdict, confidence, reason, evidence, suggestion, model, raw_response, analyzed_at, created_at, updated_at
FROM diting_ai_risk_analyses
WHERE event_id = ANY($1)
`, eventIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		analysis, err := scanAnalysis(rows)
		if err != nil {
			return nil, err
		}
		result[analysis.EventID] = analysis
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *PostgresRepository) Upsert(ctx context.Context, analysis Analysis) (Analysis, error) {
	evidence, err := json.Marshal(analysis.Evidence)
	if err != nil {
		return Analysis{}, err
	}
	if analysis.AnalyzedAt.IsZero() {
		analysis.AnalyzedAt = time.Now().UTC()
	}
	row := r.pool.QueryRow(ctx, `
INSERT INTO diting_ai_risk_analyses (event_id, ai_severity, verdict, confidence, reason, evidence, suggestion, model, raw_response, analyzed_at, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6::jsonb, $7, $8, $9, $10, NOW(), NOW())
ON CONFLICT (event_id) DO UPDATE
SET ai_severity = EXCLUDED.ai_severity,
    verdict = EXCLUDED.verdict,
    confidence = EXCLUDED.confidence,
    reason = EXCLUDED.reason,
    evidence = EXCLUDED.evidence,
    suggestion = EXCLUDED.suggestion,
    model = EXCLUDED.model,
    raw_response = EXCLUDED.raw_response,
    analyzed_at = EXCLUDED.analyzed_at,
    updated_at = NOW()
RETURNING event_id, ai_severity, verdict, confidence, reason, evidence, suggestion, model, raw_response, analyzed_at, created_at, updated_at
`, analysis.EventID, analysis.AISeverity, analysis.Verdict, analysis.Confidence, analysis.Reason, string(evidence), analysis.Suggestion, analysis.Model, analysis.RawResponse, analysis.AnalyzedAt)
	return scanAnalysis(row)
}

type analysisScanner interface {
	Scan(dest ...any) error
}

func scanAnalysis(scanner analysisScanner) (Analysis, error) {
	var analysis Analysis
	var evidenceData []byte
	if err := scanner.Scan(
		&analysis.EventID,
		&analysis.AISeverity,
		&analysis.Verdict,
		&analysis.Confidence,
		&analysis.Reason,
		&evidenceData,
		&analysis.Suggestion,
		&analysis.Model,
		&analysis.RawResponse,
		&analysis.AnalyzedAt,
		&analysis.CreatedAt,
		&analysis.UpdatedAt,
	); err != nil {
		return Analysis{}, err
	}
	if len(evidenceData) > 0 {
		_ = json.Unmarshal(evidenceData, &analysis.Evidence)
	}
	if analysis.Evidence == nil {
		analysis.Evidence = []string{}
	}
	return analysis, nil
}
