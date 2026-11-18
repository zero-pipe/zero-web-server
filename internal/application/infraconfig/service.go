package infraconfig

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"

	objectstoreapp "zero-web-server/internal/application/objectstore"
	domaininfra "zero-web-server/internal/domain/infraconfig"
	"zero-web-server/internal/infrastructure/config"
	"zero-web-server/internal/port"
	applog "zero-web-server/pkg/log"
)

const maskedSecret = "******"

type Service struct {
	repo        domaininfra.Repository
	bus         port.MessageBus
	cfg         *config.Config
	configPath  string
	mysqlPing   port.PingFunc
	redisPing   port.PingFunc
	objectStore *objectstoreapp.Service
	serverID    string

	mu              sync.Mutex
	restartRequired bool
}

func NewService(
	repo domaininfra.Repository,
	bus port.MessageBus,
	cfg *config.Config,
	configPath string,
	mysqlPing, redisPing port.PingFunc,
	objectStore *objectstoreapp.Service,
	serverID string,
) *Service {
	return &Service{
		repo: repo, bus: bus, cfg: cfg, configPath: configPath,
		mysqlPing: mysqlPing, redisPing: redisPing, objectStore: objectStore, serverID: serverID,
	}
}

// Start 启动 MQTT 消费者（未配置或未启用时跳过）。
func (s *Service) Start(ctx context.Context) {
	if err := s.startMQTT(ctx); err != nil {
		applog.Warn("mqtt consumer not started", "err", err)
	}
}

func (s *Service) Stop() {
	if s.bus != nil {
		_ = s.bus.Stop()
	}
}

func (s *Service) startMQTT(ctx context.Context) error {
	if s.bus == nil {
		return nil
	}
	row, err := s.getMQTT()
	if err != nil {
		return err
	}
	if row == nil || !row.Enabled {
		_ = s.bus.Stop()
		return nil
	}
	clientID := strings.TrimSpace(row.ClientIDPrefix)
	if clientID == "" {
		clientID = "zws"
	}
	if s.serverID != "" {
		clientID += "-" + s.serverID
	}
	clientID += "-consumer"
	cfg := port.MQTTBusConfig{
		Broker: row.Broker, ClientID: clientID, Username: row.Username,
		Password: row.Password, TopicPrefix: row.TopicPrefix, SharedGroup: "zws",
		QoS: byte(row.QoS), KeepAliveSec: row.KeepAliveSec,
		TLSEnabled: row.TLSEnabled, CACert: row.CACert,
		ClientCert: row.ClientCert, ClientKey: row.ClientKey,
	}
	if err := s.bus.Start(ctx, cfg, s.handleMessage); err != nil {
		return err
	}
	applog.Info("mqtt consumer started", "broker", row.Broker, "topic", s.bus.Topic())
	return nil
}

// handleMessage 目前在 Step 1 只做日志；Step 4 接入事件持久化。
func (s *Service) handleMessage(topic string, payload []byte) {
	if len(payload) > 512 {
		payload = payload[:512]
	}
	applog.Info("mqtt event received", "topic", topic, "bytes", len(payload), "payload", string(payload))
}

func (s *Service) getMQTT() (*domaininfra.MQTTConfig, error) {
	if s.repo == nil {
		return domaininfra.Default(), nil
	}
	row, err := s.repo.Get()
	if err != nil {
		if err == domaininfra.ErrNotFound {
			return domaininfra.Default(), nil
		}
		return nil, err
	}
	return row, nil
}

// Overview 汇总四项基础设施状态与配置（密码脱敏）。
func (s *Service) Overview(ctx context.Context) map[string]any {
	return map[string]any{
		"restartRequired": s.RestartRequired(),
		"mysql":           s.mysqlOverview(ctx),
		"redis":           s.redisOverview(ctx),
		"mqtt":            s.mqttOverview(),
		"objectStore":     s.objectStoreOverview(ctx),
	}
}

type status struct {
	Online bool   `json:"online"`
	Error  string `json:"error"`
}

func (s *Service) mysqlOverview(ctx context.Context) map[string]any {
	cfg := s.cfg.MySQL
	online, errMsg := ping(ctx, s.mysqlPing)
	return map[string]any{
		"host": cfg.Host, "port": cfg.Port, "user": cfg.User,
		"password": mask(cfg.Password), "database": cfg.Database, "charset": cfg.Charset,
		"maxIdleConns": cfg.MaxIdleConns, "maxOpenConns": cfg.MaxOpenConns,
		"status": status{Online: online, Error: errMsg},
	}
}

func (s *Service) redisOverview(ctx context.Context) map[string]any {
	cfg := s.cfg.Redis
	online, errMsg := ping(ctx, s.redisPing)
	return map[string]any{
		"host": cfg.Host, "port": cfg.Port, "password": mask(cfg.Password),
		"database": cfg.Database, "poolSize": cfg.PoolSize,
		"status": status{Online: online, Error: errMsg},
	}
}

func (s *Service) mqttOverview() map[string]any {
	row, err := s.getMQTT()
	if err != nil || row == nil {
		row = domaininfra.Default()
	}
	connected := false
	errMsg := ""
	if s.bus != nil {
		connected = s.bus.Connected()
		errMsg = s.bus.LastError()
	}
	return map[string]any{
		"enabled": row.Enabled, "broker": row.Broker, "clientIdPrefix": row.ClientIDPrefix,
		"username": row.Username, "password": mask(row.Password),
		"topicPrefix": row.TopicPrefix, "qos": row.QoS, "keepAliveSec": row.KeepAliveSec,
		"tlsEnabled": row.TLSEnabled, "caCert": row.CACert,
		"clientCert": row.ClientCert, "clientKey": row.ClientKey,
		"topic":  s.mqttTopic(),
		"status": status{Online: connected, Error: errMsg},
	}
}

func (s *Service) objectStoreOverview(ctx context.Context) map[string]any {
	if s.objectStore == nil {
		return map[string]any{"enabled": false, "provider": "noop", "status": status{Error: "未初始化"}}
	}
	row, err := s.objectStore.GetOrEmpty()
	if err != nil {
		return map[string]any{"enabled": false, "provider": "noop", "status": status{Error: err.Error()}}
	}
	online := false
	errMsg := ""
	if row.Enabled {
		c, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if err := s.objectStore.Health(c); err != nil {
			errMsg = err.Error()
		} else {
			online = true
		}
	}
	return map[string]any{
		"enabled": row.Enabled, "provider": row.Provider,
		"endpoint": row.Endpoint, "bucket": row.Bucket, "publicBase": row.PublicBase,
		"status": status{Online: online, Error: errMsg},
	}
}

func (s *Service) SaveMySQL(in config.MySQLConfig) (bool, string, error) {
	in.Host = strings.TrimSpace(in.Host)
	in.User = strings.TrimSpace(in.User)
	in.Database = strings.TrimSpace(in.Database)
	if in.Host == "" || in.User == "" || in.Database == "" || in.Port <= 0 {
		return false, "", fmt.Errorf("host/port/user/database 不能为空")
	}
	if strings.TrimSpace(in.Charset) == "" {
		in.Charset = "utf8mb4"
	}
	if in.Password == maskedSecret {
		in.Password = s.cfg.MySQL.Password
	}
	err := config.UpdateSection(s.configPath, "mysql", map[string]any{
		"host": in.Host, "port": in.Port, "user": in.User, "password": in.Password,
		"database": in.Database, "charset": in.Charset,
		"max_idle_conns": in.MaxIdleConns, "max_open_conns": in.MaxOpenConns,
	})
	if err != nil {
		return false, "", err
	}
	s.cfg.MySQL = in
	s.setRestartRequired()
	return true, s.localOverrideWarning(), nil
}

func (s *Service) SaveRedis(in config.RedisConfig) (bool, string, error) {
	in.Host = strings.TrimSpace(in.Host)
	if in.Host == "" || in.Port <= 0 {
		return false, "", fmt.Errorf("host/port 不能为空")
	}
	if in.Password == maskedSecret {
		in.Password = s.cfg.Redis.Password
	}
	err := config.UpdateSection(s.configPath, "redis", map[string]any{
		"host": in.Host, "port": in.Port, "password": in.Password,
		"database": in.Database, "pool_size": in.PoolSize,
	})
	if err != nil {
		return false, "", err
	}
	s.cfg.Redis = in
	s.setRestartRequired()
	return true, s.localOverrideWarning(), nil
}

// MQTTConfigRaw 返回含密码的配置，仅供内部布防下发使用。
func (s *Service) MQTTConfigRaw() (*domaininfra.MQTTConfig, error) {
	return s.getMQTT()
}

func (s *Service) GetMQTT() (*domaininfra.MQTTConfig, error) {
	row, err := s.getMQTT()
	if err != nil {
		return nil, err
	}
	if row.Password != "" {
		row.Password = maskedSecret
	}
	return row, nil
}

func (s *Service) SaveMQTT(in *domaininfra.MQTTConfig) error {
	if in == nil {
		return fmt.Errorf("参数错误")
	}
	if in.Enabled {
		in.Broker = strings.TrimSpace(in.Broker)
		if in.Broker == "" {
			return fmt.Errorf("启用时必须填写 Broker")
		}
	}
	if in.QoS < 0 || in.QoS > 2 {
		return fmt.Errorf("QoS 仅支持 0/1/2")
	}
	if in.KeepAliveSec <= 0 {
		in.KeepAliveSec = 30
	}
	if strings.TrimSpace(in.ClientIDPrefix) == "" {
		in.ClientIDPrefix = "zws"
	}
	in.TopicPrefix = strings.Trim(strings.TrimSpace(in.TopicPrefix), "/")
	if in.TopicPrefix == "" {
		in.TopicPrefix = "zws/events"
	}
	if in.Password == maskedSecret {
		old, err := s.getMQTT()
		if err != nil {
			return err
		}
		in.Password = old.Password
	}
	if err := s.repo.Save(in); err != nil {
		return err
	}
	if err := s.restartMQTT(context.Background()); err != nil {
		return fmt.Errorf("配置已保存，但 MQTT 连接失败: %w", err)
	}
	return nil
}

func (s *Service) restartMQTT(ctx context.Context) error {
	if s.bus != nil {
		_ = s.bus.Stop()
	}
	return s.startMQTT(ctx)
}

func (s *Service) MQTTStatus() (bool, string) {
	if s.bus == nil {
		return false, "未初始化"
	}
	return s.bus.Connected(), s.bus.LastError()
}

func (s *Service) PipelineStatus(ctx context.Context) map[string]any {
	mysqlOnline, _ := ping(ctx, s.mysqlPing)
	redisOnline, _ := ping(ctx, s.redisPing)
	mqttConnected, mqttErr := s.MQTTStatus()
	row, _ := s.getMQTT()
	mqttEnabled := row != nil && row.Enabled

	objEnabled, objHealthy, objErr := false, false, ""
	if s.objectStore != nil {
		if cfg, err := s.objectStore.GetOrEmpty(); err == nil {
			objEnabled = cfg.Enabled
			if cfg.Enabled {
				c, cancel := context.WithTimeout(ctx, 3*time.Second)
				if err := s.objectStore.Health(c); err != nil {
					objErr = err.Error()
				} else {
					objHealthy = true
				}
				cancel()
			}
		}
	}
	ready := objEnabled && objHealthy && mqttEnabled && mqttConnected
	return map[string]any{
		"ready": ready,
		"mysql": map[string]any{"online": mysqlOnline},
		"redis": map[string]any{"online": redisOnline},
		"objectStore": map[string]any{
			"enabled": objEnabled, "healthy": objHealthy, "error": objErr,
		},
		"mqtt": map[string]any{
			"enabled": mqttEnabled, "connected": mqttConnected, "error": mqttErr,
		},
	}
}

func ping(ctx context.Context, fn port.PingFunc) (bool, string) {
	if fn == nil {
		return false, "未初始化"
	}
	c, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := fn(c); err != nil {
		return false, err.Error()
	}
	return true, ""
}

func (s *Service) mqttTopic() string {
	if s.bus == nil {
		return ""
	}
	return s.bus.Topic()
}

func (s *Service) RestartRequired() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.restartRequired
}

func (s *Service) setRestartRequired() {
	s.mu.Lock()
	s.restartRequired = true
	s.mu.Unlock()
}

func (s *Service) localOverrideWarning() string {
	local := filepath.Join(filepath.Dir(s.configPath), "config.local.yaml")
	return "配置文件已更新；如存在 " + local + "，其同名配置会覆盖本次修改。重启服务后生效。"
}

func mask(secret string) string {
	if strings.TrimSpace(secret) == "" {
		return ""
	}
	return maskedSecret
}
