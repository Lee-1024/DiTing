package queryguard

import (
	"context"
	"fmt"
	"time"
)

type Limits struct {
	MaxRange       time.Duration
	DefaultTimeout time.Duration
	MaxPageSize    int
	MaxExportRows  int
	HotWindow      time.Duration
}

func NormalizePage(page int) int {
	if page <= 0 {
		return 1
	}
	return page
}

func NormalizePageSize(pageSize int, max int, fallback int) int {
	if fallback <= 0 {
		fallback = 10
	}
	if max <= 0 {
		max = fallback
	}
	if pageSize <= 0 {
		return fallback
	}
	if pageSize > max {
		return max
	}
	return pageSize
}

func ValidateTimeRange(start time.Time, end time.Time, maxRange time.Duration) error {
	if end.Before(start) {
		return fmt.Errorf("end_time must be after start_time")
	}
	if maxRange > 0 && end.Sub(start) > maxRange {
		return fmt.Errorf("time range %s exceeds maximum %s", end.Sub(start), maxRange)
	}
	return nil
}

func WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, timeout)
}
