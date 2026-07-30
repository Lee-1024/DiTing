package riskanalysis

import "context"

type Repository interface {
	GetByEventIDs(ctx context.Context, eventIDs []string) (map[string]Analysis, error)
	Upsert(ctx context.Context, analysis Analysis) (Analysis, error)
}
