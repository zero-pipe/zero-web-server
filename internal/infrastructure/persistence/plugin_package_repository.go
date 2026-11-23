package persistence

import (
	"encoding/json"
	"errors"

	domainpkg "zero-web-server/internal/domain/pluginpkg"
	"zero-web-server/internal/infrastructure/persistence/model"

	"gorm.io/gorm"
)

type PluginPackageRepository struct{ db *gorm.DB }

func NewPluginPackageRepository(db *gorm.DB) *PluginPackageRepository {
	return &PluginPackageRepository{db: db}
}

var _ domainpkg.Repository = (*PluginPackageRepository)(nil)

func (r *PluginPackageRepository) CreatePackage(p *domainpkg.Package) error {
	now := nowStr()
	p.CreateTime, p.UpdateTime = now, now
	return r.db.Create(packageToModel(p)).Error
}

func (r *PluginPackageRepository) UpdatePackage(p *domainpkg.Package) error {
	p.UpdateTime = nowStr()
	m := packageToModel(p)
	return r.db.Model(&model.PluginPackage{}).Where("id = ?", p.ID).
		Select("plugin_id", "version", "name", "vendor", "object_key", "sha256", "size_bytes",
			"manifest_json", "config_schema", "protocols", "capabilities", "stream_transports",
			"event_types", "architectures", "status", "error_msg", "update_time").
		Updates(m).Error
}

func (r *PluginPackageRepository) GetPackage(id string) (*domainpkg.Package, error) {
	var m model.PluginPackage
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainpkg.ErrNotFound
		}
		return nil, err
	}
	return packageFromModel(&m), nil
}

func (r *PluginPackageRepository) GetPackageByPluginVersion(pluginID, version string) (*domainpkg.Package, error) {
	var m model.PluginPackage
	if err := r.db.Where("plugin_id = ? AND version = ?", pluginID, version).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainpkg.ErrNotFound
		}
		return nil, err
	}
	return packageFromModel(&m), nil
}

func (r *PluginPackageRepository) ListPackages() ([]*domainpkg.Package, error) {
	var rows []model.PluginPackage
	if err := r.db.Order("update_time DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]*domainpkg.Package, 0, len(rows))
	for i := range rows {
		out = append(out, packageFromModel(&rows[i]))
	}
	return out, nil
}

func (r *PluginPackageRepository) DeletePackage(id string) error {
	return r.db.Where("id = ?", id).Delete(&model.PluginPackage{}).Error
}

func (r *PluginPackageRepository) CreateInstall(i *domainpkg.Install) error {
	now := nowStr()
	i.CreateTime, i.UpdateTime = now, now
	return r.db.Create(installToModel(i)).Error
}

func (r *PluginPackageRepository) UpdateInstall(i *domainpkg.Install) error {
	i.UpdateTime = nowStr()
	return r.db.Model(&model.PluginInstall{}).Where("id = ?", i.ID).
		Select("status", "last_error", "plugin_id", "version", "update_time").Updates(installToModel(i)).Error
}

func (r *PluginPackageRepository) GetInstall(id string) (*domainpkg.Install, error) {
	var m model.PluginInstall
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainpkg.ErrNotFound
		}
		return nil, err
	}
	return installFromModel(&m), nil
}

func (r *PluginPackageRepository) FindInstall(agentID, packageID string) (*domainpkg.Install, error) {
	var m model.PluginInstall
	if err := r.db.Where("agent_id = ? AND package_id = ?", agentID, packageID).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, domainpkg.ErrNotFound
		}
		return nil, err
	}
	return installFromModel(&m), nil
}

func (r *PluginPackageRepository) ListInstalls() ([]*domainpkg.Install, error) {
	var rows []model.PluginInstall
	if err := r.db.Order("update_time DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return installsFromRows(rows), nil
}

func (r *PluginPackageRepository) ListInstallsByAgent(agentID string) ([]*domainpkg.Install, error) {
	var rows []model.PluginInstall
	if err := r.db.Where("agent_id = ?", agentID).Order("update_time DESC").Find(&rows).Error; err != nil {
		return nil, err
	}
	return installsFromRows(rows), nil
}

func (r *PluginPackageRepository) DeleteInstall(id string) error {
	return r.db.Where("id = ?", id).Delete(&model.PluginInstall{}).Error
}

func installsFromRows(rows []model.PluginInstall) []*domainpkg.Install {
	out := make([]*domainpkg.Install, 0, len(rows))
	for i := range rows {
		out = append(out, installFromModel(&rows[i]))
	}
	return out
}

func packageToModel(p *domainpkg.Package) *model.PluginPackage {
	archs, _ := json.Marshal(p.Architectures)
	return &model.PluginPackage{
		ID: p.ID, PluginID: p.PluginID, Version: p.Version, Name: p.Name, Vendor: p.Vendor,
		ObjectKey: p.ObjectKey, SHA256: p.SHA256, SizeBytes: p.SizeBytes,
		ManifestJSON: p.ManifestJSON, ConfigSchema: p.ConfigSchema,
		Protocols: mustJSON(p.Protocols), Capabilities: mustJSON(p.Capabilities),
		StreamTransports: mustJSON(p.StreamTransports), EventTypes: mustJSON(p.EventTypes),
		Architectures: string(archs), Status: p.Status, ErrorMsg: p.ErrorMsg,
		CreateTime: p.CreateTime, UpdateTime: p.UpdateTime,
	}
}

func packageFromModel(m *model.PluginPackage) *domainpkg.Package {
	p := &domainpkg.Package{
		ID: m.ID, PluginID: m.PluginID, Version: m.Version, Name: m.Name, Vendor: m.Vendor,
		ObjectKey: m.ObjectKey, SHA256: m.SHA256, SizeBytes: m.SizeBytes,
		ManifestJSON: m.ManifestJSON, ConfigSchema: m.ConfigSchema,
		Protocols: fromJSON(m.Protocols), Capabilities: fromJSON(m.Capabilities),
		StreamTransports: fromJSON(m.StreamTransports), EventTypes: fromJSON(m.EventTypes),
		Architectures: fromJSON(m.Architectures),
		Status:        m.Status, ErrorMsg: m.ErrorMsg, CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
	if m.ManifestJSON != "" {
		var mf domainpkg.Manifest
		if json.Unmarshal([]byte(m.ManifestJSON), &mf) == nil {
			p.Manifest = &mf
			p.CustomCaps = mf.CustomCaps
		}
	}
	return p
}

func installToModel(i *domainpkg.Install) *model.PluginInstall {
	return &model.PluginInstall{
		ID: i.ID, AgentID: i.AgentID, PackageID: i.PackageID, PluginID: i.PluginID,
		Version: i.Version, Status: i.Status, LastError: i.LastError,
		CreateTime: i.CreateTime, UpdateTime: i.UpdateTime,
	}
}

func installFromModel(m *model.PluginInstall) *domainpkg.Install {
	return &domainpkg.Install{
		ID: m.ID, AgentID: m.AgentID, PackageID: m.PackageID, PluginID: m.PluginID,
		Version: m.Version, Status: m.Status, LastError: m.LastError,
		CreateTime: m.CreateTime, UpdateTime: m.UpdateTime,
	}
}

func mustJSON(v []string) string {
	raw, _ := json.Marshal(v)
	return string(raw)
}

func fromJSON(s string) []string {
	if s == "" {
		return nil
	}
	var out []string
	_ = json.Unmarshal([]byte(s), &out)
	return out
}
