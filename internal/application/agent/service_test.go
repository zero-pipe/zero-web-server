package agentapp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	domainagent "zero-web-server/internal/domain/agent"
	domainplugin "zero-web-server/internal/domain/plugin"
)

type fakeAgentRepo struct {
	agents  map[string]*domainagent.Agent
	plugins map[string]*domainagent.Plugin
}

func newFakeAgentRepo() *fakeAgentRepo {
	return &fakeAgentRepo{agents: map[string]*domainagent.Agent{}, plugins: map[string]*domainagent.Plugin{}}
}

func (r *fakeAgentRepo) UpsertAgent(a *domainagent.Agent) error {
	cp := *a
	r.agents[a.ID] = &cp
	return nil
}
func (r *fakeAgentRepo) UpdateAgentSeen(id string, ts int64, online bool) error {
	if a := r.agents[id]; a != nil {
		a.LastSeen = ts
		if online {
			a.Status = "online"
		} else {
			a.Status = "offline"
		}
	}
	return nil
}
func (r *fakeAgentRepo) ListAgents() ([]*domainagent.Agent, error) {
	out := make([]*domainagent.Agent, 0, len(r.agents))
	for _, a := range r.agents {
		out = append(out, a)
	}
	return out, nil
}
func (r *fakeAgentRepo) GetAgent(id string) (*domainagent.Agent, error) { return r.agents[id], nil }
func (r *fakeAgentRepo) UpsertPlugin(p *domainagent.Plugin) error {
	if old := r.plugins[p.ID]; old != nil && old.DesiredState != "" {
		p.DesiredState = old.DesiredState
		p.Enabled = old.DesiredState == domainagent.StateEnabled
	}
	cp := *p
	r.plugins[p.ID] = &cp
	return nil
}
func (r *fakeAgentRepo) ListPlugins(agentID string) ([]*domainagent.Plugin, error) {
	var out []*domainagent.Plugin
	for _, p := range r.plugins {
		if agentID == "" || p.AgentID == agentID {
			out = append(out, p)
		}
	}
	return out, nil
}
func (r *fakeAgentRepo) GetPlugin(agentID, pluginID string) (*domainagent.Plugin, error) {
	for _, p := range r.plugins {
		if p.AgentID == agentID && p.PluginID == pluginID {
			return p, nil
		}
	}
	return nil, nil
}
func (r *fakeAgentRepo) UpdatePluginReport(agentID, pluginID, actualState, lastError string, ts int64) error {
	if p, _ := r.GetPlugin(agentID, pluginID); p != nil {
		p.ActualState, p.LastError, p.LastSeen = actualState, lastError, ts
	}
	return nil
}
func (r *fakeAgentRepo) UpdatePluginDesired(agentID, pluginID, desiredState string) error {
	if p, _ := r.GetPlugin(agentID, pluginID); p != nil {
		p.DesiredState = desiredState
		p.Enabled = desiredState == domainagent.StateEnabled
	}
	return nil
}

func (r *fakeAgentRepo) DeletePlugin(agentID, pluginID string) error {
	if p, _ := r.GetPlugin(agentID, pluginID); p != nil {
		delete(r.plugins, p.ID)
	}
	return nil
}

type fakePluginRepo struct {
	plugins map[string]*domainplugin.Plugin
}

func newFakePluginRepo() *fakePluginRepo {
	return &fakePluginRepo{plugins: map[string]*domainplugin.Plugin{}}
}

func (r *fakePluginRepo) UpsertPlugin(p *domainplugin.Plugin) error {
	cp := *p
	cp.Online = true
	r.plugins[p.ID] = &cp
	return nil
}
func (r *fakePluginRepo) CountDevicesByPlugin(string) (int, error) { return 0, nil }
func (r *fakePluginRepo) UpdatePluginSeen(id string, ts int64, online bool) error {
	if p := r.plugins[id]; p != nil {
		p.Online, p.LastSeen = online, ts
	}
	return nil
}
func (r *fakePluginRepo) MarkPluginOffline(id string) error {
	if p := r.plugins[id]; p != nil {
		p.Online = false
	}
	return nil
}
func (r *fakePluginRepo) DeletePlugin(id string) error {
	delete(r.plugins, id)
	return nil
}
func (r *fakePluginRepo) ListPlugins() ([]*domainplugin.Plugin, error) {
	out := make([]*domainplugin.Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		out = append(out, p)
	}
	return out, nil
}
func (r *fakePluginRepo) GetPlugin(id string) (*domainplugin.Plugin, error) {
	return r.plugins[id], nil
}
func (r *fakePluginRepo) ResolvePlugin(string) (*domainplugin.Plugin, error) { return nil, nil }
func (r *fakePluginRepo) CreateDevice(*domainplugin.Device) error            { return nil }
func (r *fakePluginRepo) UpdateDevice(*domainplugin.Device) error            { return nil }
func (r *fakePluginRepo) DeleteDevice(int) error                             { return nil }
func (r *fakePluginRepo) ListDevices() ([]*domainplugin.Device, error)       { return nil, nil }
func (r *fakePluginRepo) GetDevice(int) (*domainplugin.Device, error)        { return nil, nil }
func (r *fakePluginRepo) GetDeviceByUID(string, string) (*domainplugin.Device, error) {
	return nil, nil
}
func (r *fakePluginRepo) ReplaceChannels(int, []*domainplugin.Channel) error { return nil }
func (r *fakePluginRepo) ListChannels(int) ([]*domainplugin.Channel, error)  { return nil, nil }
func (r *fakePluginRepo) GetChannel(int, string) (*domainplugin.Channel, error) {
	return nil, nil
}

func TestRegisterProjectsRuntimeAndDisable(t *testing.T) {
	ar, pr := newFakeAgentRepo(), newFakePluginRepo()
	svc := NewService(ar, pr, "secret")

	agentSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Agent-Token") != "secret" {
			http.Error(w, "bad token", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"plugins": []map[string]any{{
			"pluginId": "mock-ipc", "desiredState": "disabled", "actualState": "stopped",
		}}})
	}))
	defer agentSrv.Close()

	req := RegisterRequest{
		AgentID: "edge-1", Hostname: "host", Endpoint: agentSrv.URL,
		Plugins: []PluginRequest{{
			InstanceID: "mock-ipc-01", PluginID: "mock-ipc", Name: "Mock",
			Protocols: []string{"mock.v1"}, Capabilities: []string{"live"},
			ActualState: "running", DesiredState: "enabled", Enabled: true,
		}},
	}
	if _, err := svc.Register(context.Background(), req); err != nil {
		t.Fatalf("register: %v", err)
	}
	p := pr.plugins["mock-ipc-01"]
	if p == nil || !p.Online {
		t.Fatalf("runtime plugin not online: %+v", p)
	}
	if p.Endpoint != agentSrv.URL+"/plugins/mock-ipc" {
		t.Fatalf("endpoint=%q", p.Endpoint)
	}

	if _, err := svc.SetPluginState(context.Background(), "edge-1", "mock-ipc", "disable"); err != nil {
		t.Fatalf("disable: %v", err)
	}
	if p.Online {
		t.Fatalf("runtime plugin should be offline after disable")
	}
	ap := ar.plugins["mock-ipc-01"]
	if ap.DesiredState != domainagent.StateDisabled || ap.ActualState != "stopped" {
		t.Fatalf("agent plugin state=%+v", ap)
	}
}
