package clickhouse

import (
	"context"

	"diting/backend/internal/audit"
)

type Batch interface {
	Append(values ...any) error
	Send() error
}

type BatchPreparer interface {
	PrepareBatch(ctx context.Context, query string) (Batch, error)
}

type AuditWriter struct {
	preparer BatchPreparer
}

func NewAuditWriter(preparer BatchPreparer) *AuditWriter {
	return &AuditWriter{preparer: preparer}
}

func (w *AuditWriter) Write(ctx context.Context, events []audit.Event) error {
	if len(events) == 0 {
		return nil
	}

	batch, err := w.preparer.PrepareBatch(ctx, insertAuditEventsSQL)
	if err != nil {
		return err
	}

	for _, event := range events {
		if err := batch.Append(
			event.EventID,
			event.EventTime,
			event.EventDate,
			event.IngestTime,
			event.EventType,
			event.Action,
			event.Severity,
			event.RiskScore,
			event.Tags,
			event.HostName,
			event.HostIP,
			event.NodeName,
			event.Namespace,
			event.PodName,
			event.ContainerID,
			event.ContainerName,
			event.Image,
			event.PID,
			event.PPID,
			event.ProcessName,
			event.BinaryPath,
			event.Cmdline,
			event.CWD,
			event.ParentProcessName,
			event.ParentBinaryPath,
			event.ParentCmdline,
			event.UID,
			event.GID,
			event.Username,
			event.FilePath,
			event.FileOperation,
			event.SrcIP,
			event.SrcPort,
			event.DstIP,
			event.DstPort,
			event.Protocol,
			event.Domain,
			event.RuleIDs,
			event.RuleNames,
			event.RawEvent,
		); err != nil {
			return err
		}
	}

	return batch.Send()
}

const insertAuditEventsSQL = `
INSERT INTO audit_events (
	event_id, event_time, event_date, ingest_time,
	event_type, action, severity, risk_score, tags,
	host_name, host_ip, node_name,
	namespace, pod_name, container_id, container_name, image,
	pid, ppid, process_name, binary_path, cmdline, cwd,
	parent_process_name, parent_binary_path, parent_cmdline,
	uid, gid, username,
	file_path, file_operation,
	src_ip, src_port, dst_ip, dst_port, protocol, domain,
	rule_ids, rule_names, raw_event
)`
