package persistence

import (
	"encoding/json"
	"time"

	domainagent "zero-web-server/internal/domain/agent"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type AgentRepository struct{ db *gorm.DB }

func NewAgentRepository(db *gorm.DB) *AgentRepository { return &AgentRepository{db: db} }

var _ domainagent.Repository = (*AgentRepository)(nil)

func (r *AgentRepository) UpsertAgent(a *domainagent.Agent) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	var existing model.Agent
	err := r.db.Where("id = ?", a.ID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(&model.Agent{
			ID: a.ID, Hostname: a.Hostname, IP: a.IP, Version: a.Version,
			Endpoint: a.Endpoint, Status: a.Status, LastSeen: a.LastSeen,
			CreateTime: now, UpdateTime: now,
		}).Error
	}
	if err != nil {
		return err
	}
	return r.db.Model(&model.Agent{}).Where("id = ?", a.ID).Updates(map[string]any{
		"hostname":    a.Hostname,
		"ip":          a.IP,
		"version":     a.Version,
		"endpoint":    a.Endpoint,
		"status":      a.Status,
		"last_seen":   a.LastSeen,
		"update_time": now,
	}).Error
}

func (r *AgentRepository) UpdateAgentSeen(id string, ts int64, online bool) error {
	status := "offline"
	if online {
		status = "online"
	}
	return r.db.Model(&model.Agent{}).Where("id = ?", id).
		Updates(map[string]any{"last_seen": ts, "status": status}).Error
}

func (r *AgentRepository) ListAgents() ([]*domainagent.Agent, error) {
	var rows []model.Agent
	if err := r.db.Order("update_time DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainagent.Agent, 0, len(rows))
	for i := range rows {
		out = append(out, &domainagent.Agent{
			ID: rows[i].ID, Hostname: rows[i].Hostname, IP: rows[i].IP, Version: rows[i].Version,
			Endpoint: rows[i].Endpoint, Status: rows[i].Status, LastSeen: rows[i].LastSeen,
			CreateTime: rows[i].CreateTime, UpdateTime: rows[i].UpdateTime,
		})
	}
	return out, nil
}

func (r *AgentRepository) GetAgent(id string) (*domainagent.Agent, error) {
	var row model.Agent
	if err := r.db.Where("id = ?", id).First(&row).Error; err != nil {
		return nil, err
	}
	return &domainagent.Agent{
		ID: row.ID, Hostname: row.Hostname, IP: row.IP, Version: row.Version,
		Endpoint: row.Endpoint, Status: row.Status, LastSeen: row.LastSeen,
		CreateTime: row.CreateTime, UpdateTime: row.UpdateTime,
	}, nil
}

func (r *AgentRepository) UpsertPlugin(p *domainagent.Plugin) error {
	now := time.Now().Format("2006-01-02 15:04:05")
	protocols, _ := json.Marshal(p.Protocols)
	caps, _ := json.Marshal(p.Capabilities)
	transports, _ := json.Marshal(p.StreamTransports)
	events, _ := json.Marshal(p.EventTypes)
	var existing model.AgentPlugin
	err := r.db.Where("id = ?", p.ID).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		return r.db.Create(&model.AgentPlugin{
			ID: p.ID, AgentID: p.AgentID, PluginID: p.PluginID, Name: p.Name,
			Vendor: p.Vendor, Version: p.Version,
			Protocols: string(protocols), Capabilities: string(caps),
			StreamTransports: string(transports), EventTypes: string(events),
			ConfigSchema: p.ConfigSchema, Enabled: p.Enabled,
			DesiredState: p.DesiredState, ActualState: p.ActualState, Source: p.Source,
			LastError: p.LastError, LastSeen: p.LastSeen, CreateTime: now, UpdateTime: now,
		}).Error
	}
	if err != nil {
		return err
	}
	updates := map[string]any{
		"agent_id":          p.AgentID,
		"plugin_id":         p.PluginID,
		"name":              p.Name,
		"vendor":            p.Vendor,
		"version":           p.Version,
		"protocols":         string(protocols),
		"capabilities":      string(caps),
		"stream_transports": string(transports),
		"event_types":       string(events),
		"config_schema":     p.ConfigSchema,
		"enabled":           p.Enabled,
		"actual_state":      p.ActualState,
		"source":            p.Source,
		"last_error":        p.LastError,
		"last_seen":         p.LastSeen,
		"update_time":       now,
	}
	// 平台下发过 desired state 时，以平台为准，避免 Agent 重启后被清单默认值覆盖。
	if existing.DesiredState == "" && p.DesiredState != "" {
		updates["desired_state"] = p.DesiredState
	}
	return r.db.Model(&model.AgentPlugin{}).Where("id = ?", p.ID).Updates(updates).Error
}

func (r *AgentRepository) ListPlugins(agentID string) ([]*domainagent.Plugin, error) {
	q := r.db.Order("plugin_id ASC")
	if agentID != "" {
		q = q.Where("agent_id = ?", agentID)
	}
	var rows []model.AgentPlugin
	if err := q.Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainagent.Plugin, 0, len(rows))
	for i := range rows {
		out = append(out, toDomainAgentPlugin(&rows[i]))
	}
	return out, nil
}

func (r *AgentRepository) GetPlugin(agentID, pluginID string) (*domainagent.Plugin, error) {
	var row model.AgentPlugin
	if err := r.db.Where("agent_id = ? AND plugin_id = ?", agentID, pluginID).First(&row).Error; err != nil {
		return nil, err
	}
	return toDomainAgentPlugin(&row), nil
}

func (r *AgentRepository) UpdatePluginReport(agentID, pluginID, actualState, lastError string, ts int64) error {
	return r.db.Model(&model.AgentPlugin{}).
		Where("agent_id = ? AND plugin_id = ?", agentID, pluginID).
		Updates(map[string]any{
			"actual_state": actualState, "last_error": lastError, "last_seen": ts,
			"update_time": time.Now().Format("2006-01-02 15:04:05"),
		}).Error
}

func (r *AgentRepository) UpdatePluginDesired(agentID, pluginID, desiredState string) error {
	enabled := desiredState == domainagent.StateEnabled
	return r.db.Model(&model.AgentPlugin{}).
		Where("agent_id = ? AND plugin_id = ?", agentID, pluginID).
		Updates(map[string]any{
			"desired_state": desiredState, "enabled": enabled,
			"update_time": time.Now().Format("2006-01-02 15:04:05"),
		}).Error
}

// DeletePlugin 删除某 Agent 上的插件记录（内置插件被移除后清理）。
func (r *AgentRepository) DeletePlugin(agentID, pluginID string) error {
	return r.db.Where("agent_id = ? AND plugin_id = ?", agentID, pluginID).
		Delete(&model.AgentPlugin{}).Error
}

func toDomainAgentPlugin(row *model.AgentPlugin) *domainagent.Plugin {
	return &domainagent.Plugin{
		ID: row.ID, AgentID: row.AgentID, PluginID: row.PluginID, Name: row.Name,
		Vendor: row.Vendor, Version: row.Version,
		Protocols: decodeStrings(row.Protocols), Capabilities: decodeStrings(row.Capabilities),
		StreamTransports: decodeStrings(row.StreamTransports), EventTypes: decodeStrings(row.EventTypes),
		ConfigSchema: row.ConfigSchema, Enabled: row.Enabled,
		DesiredState: row.DesiredState, ActualState: row.ActualState, Source: row.Source,
		LastError: row.LastError, LastSeen: row.LastSeen,
		CreateTime: row.CreateTime, UpdateTime: row.UpdateTime,
	}
}
