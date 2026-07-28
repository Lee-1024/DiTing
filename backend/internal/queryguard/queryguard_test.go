package queryguard

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNormalizePageAndPageSize(t *testing.T) {
	if got := NormalizePage(0); got != 1 {
		t.Fatalf("expected page 1, got %d", got)
	}
	if got := NormalizePageSize(0, 100, 20); got != 20 {
		t.Fatalf("expected fallback page size 20, got %d", got)
	}
	if got := NormalizePageSize(500, 100, 20); got != 100 {
		t.Fatalf("expected capped page size 100, got %d", got)
	}
}

func TestValidateTimeRange(t *testing.T) {
	start := time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC)
	end := start.Add(25 * time.Hour)

	if err := ValidateTimeRange(start, end, 24*time.Hour); err == nil {
		t.Fatal("expected range exceeding max to fail")
	}
	if err := ValidateTimeRange(end, start, 24*time.Hour); err == nil {
		t.Fatal("expected inverted range to fail")
	}
	if err := ValidateTimeRange(start, start.Add(12*time.Hour), 24*time.Hour); err != nil {
		t.Fatalf("expected valid range, got %v", err)
	}
}

func TestWithTimeoutFallsBackToParentForNonPositiveTimeout(t *testing.T) {
	ctx := context.Background()
	next, cancel := WithTimeout(ctx, 0)
	defer cancel()

	if next != ctx {
		t.Fatal("expected non-positive timeout to return parent context")
	}
}

func TestWithTimeoutCancelsContext(t *testing.T) {
	ctx, cancel := WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	select {
	case <-ctx.Done():
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected context to be canceled by timeout")
	}
	if !errors.Is(ctx.Err(), context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", ctx.Err())
	}
}
