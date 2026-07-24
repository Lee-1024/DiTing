package collector

import (
	"context"
	"log/slog"
	"os"
	"strings"

	"diting/backend/internal/audit"
)

type HostMetadata struct {
	ID   string
	Name string
}

// ResolveHostMetadata 解析 Resolve Host Metadata 的最终取值。
func ResolveHostMetadata(configuredID, configuredName string) HostMetadata {
	id := strings.TrimSpace(configuredID)
	name := strings.TrimSpace(configuredName)
	if id == "" {
		id = readMachineID("/etc/machine-id")
	}
	if name == "" {
		name, _ = os.Hostname()
	}
	if id == "" {
		id = name
	}
	if name == "" {
		name = id
	}
	return HostMetadata{ID: id, Name: name}
}

// readMachineID 读取 read Machine ID 数据。
func readMachineID(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

type HostMetadataWriter struct {
	metadata HostMetadata
	next     EventWriter
}

// NewHostMetadataWriter 创建并初始化 New Host Metadata Writer 实例。
func NewHostMetadataWriter(metadata HostMetadata, next EventWriter) *HostMetadataWriter {
	return &HostMetadataWriter{metadata: metadata, next: next}
}

// Write 写入 Write 数据。
func (w *HostMetadataWriter) Write(ctx context.Context, events []audit.Event) error {
	enriched := make([]audit.Event, len(events))
	for i, event := range events {
		event.HostID = w.metadata.ID
		if w.metadata.Name != "" {
			event.HostName = w.metadata.Name
		}
		enriched[i] = event
	}
	if w.metadata.ID == "" {
		slog.Warn("collector host id is empty")
	}
	return w.next.Write(ctx, enriched)
}
