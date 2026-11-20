package pluginpkg

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	idPattern  = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{2,63}$`)
	semverRe   = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	protoRe    = regexp.MustCompile(`^[a-z0-9][a-z0-9._-]{1,63}$`)
	eventRe    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]{0,127}$`)
	goosArchAt = regexp.MustCompile(`^[a-z0-9]+-[a-z0-9]+$`)
	winAbsRe   = regexp.MustCompile(`^[A-Za-z]:`)
)

// capabilitySet 是规范 5.6 定义的合法能力枚举。
var capabilitySet = map[string]bool{
	"login": true, "logout": true, "channels": true, "live": true,
	"playback": true, "record_query": true, "ptz": true, "snapshot": true,
	"arm": true, "disarm": true, "config": true, "audio": true,
}

var transportSet = map[string]bool{
	"rtsp_url": true, "rtp_push": true, "rtmp_push": true,
}

// InspectResult 是上传/安装时的解析结论。
type InspectResult struct {
	Manifest      *Manifest
	ManifestJSON  string
	ConfigSchema  string
	Architectures []string
	SHA256        string
	Size          int64
}

// Inspect 解析 zip 包并返回清单；只做纯内存校验，不落盘。
// 所有安全边界（路径穿越、条目数、体积、符号链接）在此拦截。
func Inspect(size int64, r io.ReaderAt) (*InspectResult, error) {
	if size <= 0 {
		return nil, fmt.Errorf("空文件")
	}
	if size > MaxPackageBytes {
		return nil, fmt.Errorf("包体积超过上限 %d MB", MaxPackageBytes>>20)
	}
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return nil, fmt.Errorf("不是有效的 zip 包: %w", err)
	}
	if len(zr.File) > MaxZipEntries {
		return nil, fmt.Errorf("包内条目超过 %d", MaxZipEntries)
	}

	h := sha256.New()
	sec := io.NewSectionReader(r, 0, size)
	if _, err := io.Copy(h, sec); err != nil {
		return nil, err
	}

	var (
		manifestRaw []byte
		byName      = map[string]*zip.File{}
		total       int64
	)
	for _, f := range zr.File {
		name := f.Name
		if err := checkEntryName(name); err != nil {
			return nil, err
		}
		if f.FileInfo().Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("包内不允许符号链接: %s", name)
		}
		if !f.FileInfo().IsDir() {
			total += int64(f.UncompressedSize64)
			if total > MaxUncompressedBytes {
				return nil, fmt.Errorf("解压后体积超过上限 %d MB", MaxUncompressedBytes>>20)
			}
			byName[path.Clean(name)] = f
		}
		if name == "plugin.yaml" || name == "plugin.yml" {
			rc, err := f.Open()
			if err != nil {
				return nil, fmt.Errorf("读取 plugin.yaml 失败: %w", err)
			}
			manifestRaw, err = io.ReadAll(io.LimitReader(rc, 1<<20))
			_ = rc.Close()
			if err != nil {
				return nil, fmt.Errorf("读取 plugin.yaml 失败: %w", err)
			}
		}
	}
	if manifestRaw == nil {
		return nil, fmt.Errorf("包根目录缺少 plugin.yaml")
	}

	var m Manifest
	if err := yaml.Unmarshal(manifestRaw, &m); err != nil {
		return nil, fmt.Errorf("plugin.yaml 解析失败: %w", err)
	}
	if err := validateManifest(&m); err != nil {
		return nil, err
	}

	schema := strings.TrimSpace(m.ConfigSchema)
	if schema != "" && !strings.HasPrefix(schema, "{") {
		f, ok := byName[path.Clean(schema)]
		if !ok {
			return nil, fmt.Errorf("config_schema 文件不存在: %s", schema)
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("读取 config_schema 失败: %w", err)
		}
		raw, err := io.ReadAll(io.LimitReader(rc, 1<<20))
		_ = rc.Close()
		if err != nil {
			return nil, fmt.Errorf("读取 config_schema 失败: %w", err)
		}
		schema = strings.TrimSpace(string(raw))
	}
	if schema != "" && !json.Valid([]byte(schema)) {
		return nil, fmt.Errorf("config_schema 不是合法 JSON")
	}

	archs := make([]string, 0, len(m.Executable))
	for ga, p := range m.Executable {
		if _, ok := byName[path.Clean(strings.TrimPrefix(p, "./"))]; !ok {
			return nil, fmt.Errorf("executable[%s] 声明的文件不存在: %s", ga, p)
		}
		archs = append(archs, ga)
	}
	sort.Strings(archs)

	// 规范化落库形式：config_schema 统一存内联 JSON，便于前端渲染。
	m.ConfigSchema = schema
	mj, _ := json.Marshal(&m)

	return &InspectResult{
		Manifest:      &m,
		ManifestJSON:  string(mj),
		ConfigSchema:  schema,
		Architectures: archs,
		SHA256:        hex.EncodeToString(h.Sum(nil)),
		Size:          size,
	}, nil
}

// checkEntryName 拒绝路径穿越、绝对路径与不安全的条目名。
func checkEntryName(name string) error {
	if name == "" {
		return fmt.Errorf("包内存在空条目名")
	}
	if strings.ContainsRune(name, '\\') {
		return fmt.Errorf("条目名不允许反斜杠: %s", name)
	}
	if strings.HasPrefix(name, "/") || winAbsRe.MatchString(name) {
		return fmt.Errorf("条目名不允许绝对路径: %s", name)
	}
	clean := path.Clean(name)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return fmt.Errorf("条目名存在路径穿越: %s", name)
	}
	return nil
}

// validateManifest 按规范第 4 节校验清单字段。
func validateManifest(m *Manifest) error {
	if m.Spec != SpecV1 {
		return fmt.Errorf("spec 必须为 %d", SpecV1)
	}
	m.ID = strings.TrimSpace(m.ID)
	if !idPattern.MatchString(m.ID) {
		return fmt.Errorf("id 必须匹配 %s", idPattern.String())
	}
	m.Name = strings.TrimSpace(m.Name)
	if m.Name == "" {
		return fmt.Errorf("name 不能为空")
	}
	m.Vendor = strings.TrimSpace(m.Vendor)
	if m.Vendor == "" {
		return fmt.Errorf("vendor 不能为空")
	}
	m.Version = strings.TrimSpace(m.Version)
	if !semverRe.MatchString(m.Version) {
		return fmt.Errorf("version 必须是 x.y.z")
	}
	if m.RPC == "" {
		m.RPC = RPCV1
	}
	if m.RPC != RPCV1 {
		return fmt.Errorf("不支持的 rpc 版本: %s", m.RPC)
	}
	if len(m.Protocols) == 0 {
		return fmt.Errorf("protocols 不能为空")
	}
	for _, p := range m.Protocols {
		if !protoRe.MatchString(p) {
			return fmt.Errorf("protocols 含非法标识: %s", p)
		}
	}
	if len(m.Executable) == 0 {
		return fmt.Errorf("executable 至少声明一个平台")
	}
	for ga := range m.Executable {
		if !goosArchAt.MatchString(ga) {
			return fmt.Errorf("executable 键必须是 goos-goarch: %s", ga)
		}
	}
	if len(m.Capabilities) == 0 {
		return fmt.Errorf("capabilities 不能为空")
	}
	for _, c := range m.Capabilities {
		if !capabilitySet[strings.TrimSpace(c)] {
			return fmt.Errorf("未知能力: %s", c)
		}
	}
	if len(m.StreamTransports) == 0 {
		return fmt.Errorf("stream_transports 不能为空")
	}
	for _, t := range m.StreamTransports {
		if !transportSet[t] {
			return fmt.Errorf("未知取流方式: %s", t)
		}
	}
	for _, e := range m.EventTypes {
		if !eventRe.MatchString(e) {
			return fmt.Errorf("非法事件类型: %s", e)
		}
	}
	if l := strings.TrimSpace(m.Listen); l != "" {
		if _, _, err := splitListen(l); err != nil {
			return err
		}
	}
	return nil
}

// splitListen 校验 host:port 形式，端口允许为 0。
func splitListen(v string) (string, string, error) {
	i := strings.LastIndex(v, ":")
	if i <= 0 || i == len(v)-1 {
		return "", "", fmt.Errorf("listen 必须是 host:port 形式: %s", v)
	}
	return v[:i], v[i+1:], nil
}

// FromBytes 便捷入口（测试用）。
func FromBytes(data []byte) (*InspectResult, error) {
	return Inspect(int64(len(data)), bytes.NewReader(data))
}
