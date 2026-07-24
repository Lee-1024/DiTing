package collector

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"time"

	"diting/backend/internal/audit"
)

type tetragonEnvelope struct {
	Time        string            `json:"time"`
	NodeName    string            `json:"node_name"`
	ProcessExec *processExecEvent `json:"process_exec"`
	ProcessExit *processExitEvent `json:"process_exit"`
}

type processExecEvent struct {
	Process processInfo `json:"process"`
	Parent  processInfo `json:"parent"`
}

type processExitEvent struct {
	Process processInfo `json:"process"`
	Parent  processInfo `json:"parent"`
	Time    string      `json:"time"`
}

type processInfo struct {
	ExecID             string             `json:"exec_id"`
	PID                uint32             `json:"pid"`
	UID                uint32             `json:"uid"`
	GID                uint32             `json:"gid"`
	AUID               uint32             `json:"auid"`
	Binary             string             `json:"binary"`
	Arguments          string             `json:"arguments"`
	CWD                string             `json:"cwd"`
	Pod                podInfo            `json:"pod"`
	ProcessCredentials processCredentials `json:"process_credentials"`
}

type processCredentials struct {
	UID  uint32 `json:"uid"`
	GID  uint32 `json:"gid"`
	EUID uint32 `json:"euid"`
	EGID uint32 `json:"egid"`
}

type podInfo struct {
	Namespace string        `json:"namespace"`
	Name      string        `json:"name"`
	Container containerInfo `json:"container"`
}

type containerInfo struct {
	ID    string    `json:"id"`
	Name  string    `json:"name"`
	Image imageInfo `json:"image"`
}

type imageInfo struct {
	Name string `json:"name"`
}

var ErrUnsupportedEvent = errors.New("unsupported tetragon event")

// ParseTetragonEvent 解析 Parse Tetragon Event 并返回结构化结果。
func ParseTetragonEvent(data []byte) (audit.Event, error) {
	var envelope tetragonEnvelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return audit.Event{}, err
	}
	if envelope.ProcessExec != nil {
		return parseProcessExec(envelope, data)
	}
	if envelope.ProcessExit != nil {
		return parseProcessExit(envelope, data)
	}
	return audit.Event{}, ErrUnsupportedEvent
}

// parseProcessExec 解析 parse Process Exec 并返回结构化结果。
func parseProcessExec(envelope tetragonEnvelope, data []byte) (audit.Event, error) {
	eventTime, err := time.Parse(time.RFC3339Nano, envelope.Time)
	if err != nil {
		return audit.Event{}, err
	}

	process := envelope.ProcessExec.Process
	parent := envelope.ProcessExec.Parent
	eventID := stableID(data)

	return audit.Event{
		EventID:           eventID,
		EventTime:         eventTime,
		EventDate:         dateOnly(eventTime),
		IngestTime:        time.Now().UTC(),
		EventType:         "process_exec",
		Action:            "exec",
		Severity:          "info",
		RiskScore:         0,
		NodeName:          envelope.NodeName,
		Namespace:         process.Pod.Namespace,
		PodName:           process.Pod.Name,
		ContainerID:       process.Pod.Container.ID,
		ContainerName:     process.Pod.Container.Name,
		Image:             process.Pod.Container.Image.Name,
		PID:               process.PID,
		PPID:              parent.PID,
		ProcessName:       processName(process.Binary),
		BinaryPath:        process.Binary,
		Cmdline:           joinCmdline(process.Binary, process.Arguments),
		CWD:               process.CWD,
		ParentProcessName: processName(parent.Binary),
		ParentBinaryPath:  parent.Binary,
		ParentCmdline:     joinCmdline(parent.Binary, parent.Arguments),
		UID:               process.UID,
		GID:               process.GID,
		AUID:              process.AUID,
		EUID:              process.ProcessCredentials.EUID,
		EGID:              process.ProcessCredentials.EGID,
		RawEvent:          string(data),
	}, nil
}

// parseProcessExit 解析 parse Process Exit 并返回结构化结果。
func parseProcessExit(envelope tetragonEnvelope, data []byte) (audit.Event, error) {
	eventTimeRaw := envelope.ProcessExit.Time
	if eventTimeRaw == "" {
		eventTimeRaw = envelope.Time
	}
	eventTime, err := time.Parse(time.RFC3339Nano, eventTimeRaw)
	if err != nil {
		return audit.Event{}, err
	}

	process := envelope.ProcessExit.Process
	parent := envelope.ProcessExit.Parent
	eventID := stableID(data)

	return audit.Event{
		EventID:           eventID,
		EventTime:         eventTime,
		EventDate:         dateOnly(eventTime),
		IngestTime:        time.Now().UTC(),
		EventType:         "process_exit",
		Action:            "exit",
		Severity:          "info",
		RiskScore:         0,
		NodeName:          envelope.NodeName,
		PID:               process.PID,
		PPID:              parent.PID,
		ProcessName:       processName(process.Binary),
		BinaryPath:        process.Binary,
		Cmdline:           joinCmdline(process.Binary, process.Arguments),
		CWD:               process.CWD,
		ParentProcessName: processName(parent.Binary),
		ParentBinaryPath:  parent.Binary,
		ParentCmdline:     joinCmdline(parent.Binary, parent.Arguments),
		UID:               process.UID,
		GID:               process.GID,
		AUID:              process.AUID,
		EUID:              process.ProcessCredentials.EUID,
		EGID:              process.ProcessCredentials.EGID,
		RawEvent:          string(data),
	}, nil
}

// joinCmdline 处理 join Cmdline 相关逻辑。
func joinCmdline(binary, arguments string) string {
	if arguments == "" {
		return binary
	}
	if binary == "" {
		return arguments
	}
	return binary + " " + arguments
}

// processName 处理 process Name 相关逻辑。
func processName(binary string) string {
	if binary == "" {
		return ""
	}
	if strings.HasPrefix(binary, "[") && strings.HasSuffix(binary, "]") {
		return binary
	}
	return strings.TrimSuffix(filepath.Base(binary), ".exe")
}

// dateOnly 处理 date Only 相关逻辑。
func dateOnly(value time.Time) time.Time {
	year, month, day := value.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// stableID 处理 stable ID 相关逻辑。
func stableID(data []byte) string {
	sum := sha1.Sum(data)
	return hex.EncodeToString(sum[:])
}
