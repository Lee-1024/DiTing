package audit

import "time"

type OperationGroup struct {
	GroupID        string    `json:"groupId"`
	Representative Event     `json:"representative"`
	EventCount     uint64    `json:"eventCount"`
	EventTypes     []string  `json:"eventTypes"`
	FilePaths      []string  `json:"filePaths"`
	Tags           []string  `json:"tags"`
	MaxSeverity    string    `json:"maxSeverity"`
	FirstSeen      time.Time `json:"firstSeen"`
	LastSeen       time.Time `json:"lastSeen"`
}
