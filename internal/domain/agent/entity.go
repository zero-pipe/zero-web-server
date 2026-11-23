// Package agent 定义 zero-agent 与平台之间的控制面模型。
package agent

// Agent 是安装在接入侧的常驻服务实例。
type Agent struct {
	ID         string `json:"id"`
	Hostname   string `json:"hostname"`
	IP         string `json:"ip"`
	Version    string `json:"version"`
	Endpoint   string `json:"endpoint"` // 平台可达的 Agent 地址
	Status     string `json:"status"`   // online | offline
	LastSeen   int64  `json:"lastSeen"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

// Plugin 是 Agent 托管的一个插件实例。
type Plugin struct {
	ID               string   `json:"id"` // 运行时实例 ID，zws_plugin.id
	AgentID          string   `json:"agentId"`
	PluginID         string   `json:"pluginId"`
	Name             string   `json:"name"`
	Vendor           string   `json:"vendor"`
	Version          string   `json:"version"`
	Protocols        []string `json:"protocols"`
	Capabilities     []string `json:"capabilities"`
	StreamTransports []string `json:"streamTransports"`
	EventTypes       []string `json:"eventTypes"`
	ConfigSchema     string   `json:"configSchema"`
	Enabled          bool     `json:"enabled"`
	DesiredState     string   `json:"desiredState"` // enabled | disabled
	ActualState      string   `json:"actualState"`  // discovered | starting | running | stopped | error
	Source           string   `json:"source"`       // platform | manual（manual 即随 Agent 分发的内置/本地插件）
	LastError        string   `json:"lastError"`
	LastSeen         int64    `json:"lastSeen"`
	CreateTime       string   `json:"createTime"`
	UpdateTime       string   `json:"updateTime"`
}

// DesiredState 值与 Plugin.DesiredState 对应。
const (
	StateEnabled  = "enabled"
	StateDisabled = "disabled"
)

type Repository interface {
	UpsertAgent(a *Agent) error
	UpdateAgentSeen(id string, ts int64, online bool) error
	ListAgents() ([]*Agent, error)
	GetAgent(id string) (*Agent, error)

	UpsertPlugin(p *Plugin) error
	ListPlugins(agentID string) ([]*Plugin, error)
	GetPlugin(agentID, pluginID string) (*Plugin, error)
	UpdatePluginReport(agentID, pluginID, actualState, lastError string, ts int64) error
	UpdatePluginDesired(agentID, pluginID, desiredState string) error
	DeletePlugin(agentID, pluginID string) error
}
