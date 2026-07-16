package operationlog

import (
	"context"
	"time"
)

type Entry struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Username  string    `json:"username"`
	Method    string    `json:"method"`
	Path      string    `json:"path"`
	Status    int       `json:"status"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"userAgent"`
	CreatedAt time.Time `json:"createdAt"`
}

type Repository interface {
	Create(ctx context.Context, entry Entry) error
	List(ctx context.Context, query Query) ([]Entry, int, error)
}

type Query struct {
	StartTime time.Time
	EndTime   time.Time
	Username  string
	Method    string
	Keyword   string
	Status    int
	Page      int
	PageSize  int
}
