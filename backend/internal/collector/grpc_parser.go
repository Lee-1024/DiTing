package collector

import (
	"time"

	"diting/backend/internal/audit"
	tetragon "github.com/cilium/tetragon/api/v1/tetragon"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func ParseTetragonGRPCEvent(response *tetragon.GetEventsResponse) (audit.Event, error) {
	data, err := protojson.Marshal(response)
	if err != nil {
		return audit.Event{}, err
	}
	if event := response.GetProcessExec(); event != nil {
		return parseGRPCProcessExec(response, event, data), nil
	}
	if event := response.GetProcessExit(); event != nil {
		return parseGRPCProcessExit(response, event, data), nil
	}
	return audit.Event{}, ErrUnsupportedEvent
}

func parseGRPCProcessExec(response *tetragon.GetEventsResponse, event *tetragon.ProcessExec, data []byte) audit.Event {
	eventTime := grpcEventTime(response)
	process := event.GetProcess()
	parent := event.GetParent()
	pod := process.GetPod()
	container := pod.GetContainer()

	return audit.Event{
		EventID:           stableID(data),
		EventTime:         eventTime,
		EventDate:         dateOnly(eventTime),
		IngestTime:        time.Now().UTC(),
		EventType:         "process_exec",
		Action:            "exec",
		Severity:          "info",
		RiskScore:         0,
		NodeName:          response.GetNodeName(),
		Namespace:         pod.GetNamespace(),
		PodName:           pod.GetName(),
		ContainerID:       container.GetId(),
		ContainerName:     container.GetName(),
		Image:             imageName(container),
		PID:               uint32Value(process.GetPid()),
		PPID:              uint32Value(parent.GetPid()),
		ProcessName:       processName(process.GetBinary()),
		BinaryPath:        process.GetBinary(),
		Cmdline:           joinCmdline(process.GetBinary(), process.GetArguments()),
		CWD:               process.GetCwd(),
		ParentProcessName: processName(parent.GetBinary()),
		ParentBinaryPath:  parent.GetBinary(),
		ParentCmdline:     joinCmdline(parent.GetBinary(), parent.GetArguments()),
		UID:               uint32Value(process.GetUid()),
		GID:               uint32Value(process.GetProcessCredentials().GetGid()),
		AUID:              uint32Value(process.GetAuid()),
		EUID:              uint32Value(process.GetProcessCredentials().GetEuid()),
		EGID:              uint32Value(process.GetProcessCredentials().GetEgid()),
		RawEvent:          string(data),
	}
}

func parseGRPCProcessExit(response *tetragon.GetEventsResponse, event *tetragon.ProcessExit, data []byte) audit.Event {
	eventTime := grpcEventTime(response)
	if event.GetTime() != nil {
		eventTime = event.GetTime().AsTime()
	}
	process := event.GetProcess()
	parent := event.GetParent()

	return audit.Event{
		EventID:           stableID(data),
		EventTime:         eventTime,
		EventDate:         dateOnly(eventTime),
		IngestTime:        time.Now().UTC(),
		EventType:         "process_exit",
		Action:            "exit",
		Severity:          "info",
		RiskScore:         0,
		NodeName:          response.GetNodeName(),
		PID:               uint32Value(process.GetPid()),
		PPID:              uint32Value(parent.GetPid()),
		ProcessName:       processName(process.GetBinary()),
		BinaryPath:        process.GetBinary(),
		Cmdline:           joinCmdline(process.GetBinary(), process.GetArguments()),
		CWD:               process.GetCwd(),
		ParentProcessName: processName(parent.GetBinary()),
		ParentBinaryPath:  parent.GetBinary(),
		ParentCmdline:     joinCmdline(parent.GetBinary(), parent.GetArguments()),
		UID:               uint32Value(process.GetUid()),
		GID:               uint32Value(process.GetProcessCredentials().GetGid()),
		AUID:              uint32Value(process.GetAuid()),
		EUID:              uint32Value(process.GetProcessCredentials().GetEuid()),
		EGID:              uint32Value(process.GetProcessCredentials().GetEgid()),
		RawEvent:          string(data),
	}
}

func grpcEventTime(response *tetragon.GetEventsResponse) time.Time {
	if response.GetTime() != nil {
		return response.GetTime().AsTime()
	}
	return time.Now().UTC()
}

func uint32Value(value *wrapperspb.UInt32Value) uint32 {
	if value == nil {
		return 0
	}
	return value.Value
}

func imageName(container *tetragon.Container) string {
	if container.GetImage() == nil {
		return ""
	}
	return container.GetImage().GetName()
}
