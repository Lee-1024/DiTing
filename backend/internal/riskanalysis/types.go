package riskanalysis

import "time"

type Analysis struct {
	EventID     string    `json:"eventId"`
	AISeverity  string    `json:"aiSeverity"`
	Verdict     string    `json:"verdict"`
	Confidence  int       `json:"confidence"`
	Reason      string    `json:"reason"`
	Evidence    []string  `json:"evidence"`
	Suggestion  string    `json:"suggestion"`
	Model       string    `json:"model"`
	RawResponse string    `json:"rawResponse,omitempty"`
	AnalyzedAt  time.Time `json:"analyzedAt"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

const (
	VerdictTruePositive  = "true_positive"
	VerdictSuspicious    = "suspicious"
	VerdictFalsePositive = "false_positive"
	VerdictNeedsReview   = "needs_review"
)
