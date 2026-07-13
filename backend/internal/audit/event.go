package audit

import "time"

type Event struct {
	EventID    string    `json:"eventId"`
	EventTime  time.Time `json:"eventTime"`
	EventDate  time.Time `json:"eventDate"`
	IngestTime time.Time `json:"ingestTime"`

	EventType string   `json:"eventType"`
	Action    string   `json:"action"`
	Severity  string   `json:"severity"`
	RiskScore uint8    `json:"riskScore"`
	Tags      []string `json:"tags"`

	HostName string `json:"hostName"`
	HostIP   string `json:"hostIp"`
	NodeName string `json:"nodeName"`

	Namespace     string `json:"namespace"`
	PodName       string `json:"podName"`
	ContainerID   string `json:"containerId"`
	ContainerName string `json:"containerName"`
	Image         string `json:"image"`

	PID         uint32 `json:"pid"`
	PPID        uint32 `json:"ppid"`
	ProcessName string `json:"processName"`
	BinaryPath  string `json:"binaryPath"`
	Cmdline     string `json:"cmdline"`
	CWD         string `json:"cwd"`

	ParentProcessName string `json:"parentProcessName"`
	ParentBinaryPath  string `json:"parentBinaryPath"`
	ParentCmdline     string `json:"parentCmdline"`

	UID      uint32 `json:"uid"`
	GID      uint32 `json:"gid"`
	Username string `json:"username"`
	AUID     uint32 `json:"auid"`
	EUID     uint32 `json:"euid"`
	EGID     uint32 `json:"egid"`

	LoginUsername string `json:"loginUsername"`

	FilePath      string `json:"filePath"`
	FileOperation string `json:"fileOperation"`

	SrcIP    string `json:"srcIp"`
	SrcPort  uint16 `json:"srcPort"`
	DstIP    string `json:"dstIp"`
	DstPort  uint16 `json:"dstPort"`
	Protocol string `json:"protocol"`
	Domain   string `json:"domain"`

	RuleIDs   []string `json:"ruleIds"`
	RuleNames []string `json:"ruleNames"`

	RawEvent string `json:"rawEvent"`
}
