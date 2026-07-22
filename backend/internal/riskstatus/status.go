package riskstatus

import (
	"context"
	"errors"
	"time"
)

const (
	StatusOpen          = "open"
	StatusConfirmed     = "confirmed"
	StatusFalsePositive = "false_positive"
	StatusIgnored       = "ignored"
	StatusClosed        = "closed"
	StatusIgnoreSimilar = "ignore_similar"
)

type Disposition struct {
	EventID     string     `json:"eventId"`
	Status      string     `json:"status"`
	Note        string     `json:"note"`
	Scope       string     `json:"scope"`
	Fingerprint string     `json:"fingerprint"`
	HandledBy   string     `json:"handledBy"`
	HandledAt   *time.Time `json:"handledAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type Repository interface {
	ListByEventIDs(ctx context.Context, eventIDs []string) (map[string]Disposition, error)
	ListByFingerprints(ctx context.Context, fingerprints []string) (map[string]Disposition, error)
	Upsert(ctx context.Context, disposition Disposition) (Disposition, error)
}

var ErrInvalidStatus = errors.New("invalid risk status")

func NormalizeStatus(status string) (string, error) {
	switch status {
	case "", StatusOpen:
		return StatusOpen, nil
	case StatusConfirmed:
		return StatusConfirmed, nil
	case StatusFalsePositive:
		return StatusFalsePositive, nil
	case StatusIgnored:
		return StatusIgnored, nil
	case StatusClosed:
		return StatusClosed, nil
	case StatusIgnoreSimilar:
		return StatusIgnoreSimilar, nil
	default:
		return "", ErrInvalidStatus
	}
}
