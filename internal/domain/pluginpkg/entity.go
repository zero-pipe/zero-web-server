// Package pluginpkg 定义插件安装包（.zwplugin）的清单、校验与安装任务模型。
//
// 规范见 docs/PLUGIN_PLATFORM.md：包是 zip，根目录必须有 plugin.yaml，
// 平台在上传时完成安全与清单校验，Agent 安装时复核清单与哈希。
package pluginpkg

import (
	"errors"
	"time"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("record not found")

// 包与安装状态。
const (
	PackageValid   = "valid"
	PackageInvalid = "invalid"

	InstallPending     = "pending"
	InstallDownloading = "downloading"
	InstallInstalled   = "installed"
	InstallFailed      = "failed"
	InstallUninstalled = "uninstalled"

	// SpecV1 / RPCV1 是当前支持的规范版本。
	SpecV1 = 1
	RPCV1  = "v1"

	// 包上限（与文档 3.1 一致）。
	MaxPackageBytes      = 500 << 20
	MaxZipEntries        = 2000
	MaxUncompressedBytes = 1 << 30
)

// Manifest 是 plugin.yaml 的解析结果（规范第 4 节）。
type Manifest struct {
	Spec             int                `yaml:"spec" json:"spec"`
	ID               string             `yaml:"id" json:"id"`
	Name             string             `yaml:"name" json:"name"`
	Vendor           string             `yaml:"vendor" json:"vendor"`
	Version          string             `yaml:"version" json:"version"`
	Protocols        []string           `yaml:"protocols" json:"protocols"`
	RPC              string             `yaml:"rpc" json:"rpc"`
	Executable       map[string]string  `yaml:"executable" json:"executable"`
	Args             []string           `yaml:"args" json:"args"`
	Listen           string             `yaml:"listen" json:"listen"`
	Health           string             `yaml:"health" json:"health"`
	Capabilities     []string           `yaml:"capabilities" json:"capabilities"`
	CustomCaps       []CustomCapability `yaml:"custom_capabilities" json:"customCapabilities"`
	StreamTransports []string           `yaml:"stream_transports" json:"streamTransports"`
	EventTypes       []string           `yaml:"event_types" json:"eventTypes"`
	ConfigSchema     string             `yaml:"config_schema" json:"configSchema"`
	Env              map[string]string  `yaml:"env" json:"env"`
	MinPlatform      string             `yaml:"min_platform" json:"minPlatform"`
}

// CustomCapability 是插件自定义能力（平台不解释语义，只登记 + 通用 command 透传）。
type CustomCapability struct {
	ID           string `yaml:"id" json:"id"`
	Title        string `yaml:"title" json:"title"`
	ParamsSchema string `yaml:"params_schema" json:"paramsSchema"` // 可选，内联 JSON Schema
	ResultSchema string `yaml:"result_schema" json:"resultSchema"` // 可选，内联 JSON Schema
}

// Package 是平台登记的插件包（zws_plugin_package）。
type Package struct {
	ID               string             `json:"id"`
	PluginID         string             `json:"pluginId"`
	Version          string             `json:"version"`
	Name             string             `json:"name"`
	Vendor           string             `json:"vendor"`
	ObjectKey        string             `json:"objectKey"`
	SHA256           string             `json:"sha256"`
	SizeBytes        int64              `json:"sizeBytes"`
	ManifestJSON     string             `json:"manifestJson"`
	ConfigSchema     string             `json:"configSchema"`
	Protocols        []string           `json:"protocols"`
	Capabilities     []string           `json:"capabilities"`
	CustomCaps       []CustomCapability `json:"customCapabilities"`
	StreamTransports []string           `json:"streamTransports"`
	EventTypes       []string           `json:"eventTypes"`
	Status           string             `json:"status"`
	ErrorMsg         string             `json:"errorMsg"`
	CreateTime       string             `json:"createTime"`
	UpdateTime       string             `json:"updateTime"`
	Manifest         *Manifest          `json:"-"`
	Architectures    []string           `json:"architectures"`
}

// Install 是某 Agent 上的安装任务/状态（zws_plugin_install）。
type Install struct {
	ID         string `json:"id"`
	AgentID    string `json:"agentId"`
	PackageID  string `json:"packageId"`
	PluginID   string `json:"pluginId"`
	Version    string `json:"version"`
	Status     string `json:"status"`
	LastError  string `json:"lastError"`
	CreateTime string `json:"createTime"`
	UpdateTime string `json:"updateTime"`
}

// Repository 是插件包与安装任务的持久化端口。
type Repository interface {
	CreatePackage(p *Package) error
	UpdatePackage(p *Package) error
	GetPackage(id string) (*Package, error)
	GetPackageByPluginVersion(pluginID, version string) (*Package, error)
	ListPackages() ([]*Package, error)
	DeletePackage(id string) error

	CreateInstall(i *Install) error
	UpdateInstall(i *Install) error
	GetInstall(id string) (*Install, error)
	FindInstall(agentID, packageID string) (*Install, error)
	ListInstalls() ([]*Install, error)
	ListInstallsByAgent(agentID string) ([]*Install, error)
	DeleteInstall(id string) error
}

// Now 便于测试替换时间源。
var Now = func() time.Time { return time.Now() }
