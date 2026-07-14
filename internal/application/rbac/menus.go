package rbac

import (
	"encoding/json"
	"strings"
)

// 菜单权限码，与 web/src/layout/menu.js 的 primary id 对齐（中台能力域）。
const (
	MenuAccess  = "access"  // 接入
	MenuMedia   = "media"   // 媒体
	MenuStorage = "storage" // 存储
	MenuOrg     = "org"     // 组织
	MenuApp     = "app"     // 应用（地图/分屏/报警）
	MenuOps     = "ops"     // 运维
	MenuUser    = "user"    // 用户

	// 兼容旧码（角色 authority 历史数据）
	MenuMap     = "map"
	MenuLive    = "live"
	MenuDevice  = "device"
	MenuRecord  = "record"
	MenuAlarm   = "alarm"
	MenuSystem  = "system"
	MenuObserve = "observe" // 曾用名「观察/监控」
)

// AllMenus 全部一级菜单权限（新能力域）。
var AllMenus = []string{
	MenuApp, MenuAccess, MenuMedia, MenuStorage, MenuOrg, MenuOps, MenuUser,
}

// legacyAlias 旧一级码映射到新能力码。
var legacyAlias = map[string]string{
	MenuDevice:  MenuAccess,
	MenuSystem:  MenuAccess,
	MenuRecord:  MenuStorage,
	MenuLive:    MenuApp,
	MenuMap:     MenuApp,
	MenuAlarm:   MenuApp,
	MenuObserve: MenuApp,
}

// MenuDefs 供前端角色配置勾选。
var MenuDefs = []struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}{
	{MenuApp, "应用管理"},
	{MenuAccess, "接入管理"},
	{MenuMedia, "媒体管理"},
	{MenuStorage, "存储管理"},
	{MenuOrg, "组织管理"},
	{MenuOps, "运维管理"},
	{MenuUser, "用户管理"},
}

const (
	AuthorityAll    = "*"
	AuthorityLegacy = "0" // 兼容旧 admin
)

// IsFullAccess authority 是否表示全部菜单。
func IsFullAccess(authority string) bool {
	a := strings.TrimSpace(authority)
	return a == AuthorityAll || a == AuthorityLegacy
}

// ParseMenus 解析角色 authority → 菜单码列表。
func ParseMenus(roleID int, authority string) []string {
	if roleID == 1 || IsFullAccess(authority) {
		out := make([]string, len(AllMenus))
		copy(out, AllMenus)
		return out
	}
	a := strings.TrimSpace(authority)
	if a == "" {
		return []string{}
	}
	var codes []string
	if err := json.Unmarshal([]byte(a), &codes); err != nil {
		parts := strings.Split(a, ",")
		codes = codes[:0]
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				codes = append(codes, p)
			}
		}
	}
	return Normalize(codes)
}

// EncodeMenus 菜单码 → 存库字符串；全选存 *。
func EncodeMenus(codes []string) string {
	codes = Normalize(codes)
	if len(codes) == 0 {
		return "[]"
	}
	if len(codes) >= len(AllMenus) {
		all := true
		set := map[string]bool{}
		for _, c := range codes {
			set[c] = true
		}
		for _, m := range AllMenus {
			if !set[m] {
				all = false
				break
			}
		}
		if all {
			return AuthorityAll
		}
	}
	b, err := json.Marshal(codes)
	if err != nil {
		return "[]"
	}
	return string(b)
}

// Normalize 去重、旧码映射，只保留合法菜单码。
func Normalize(codes []string) []string {
	valid := map[string]bool{}
	for _, m := range AllMenus {
		valid[m] = true
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		c = strings.TrimSpace(c)
		if alias, ok := legacyAlias[c]; ok {
			c = alias
		}
		if !valid[c] || seen[c] {
			continue
		}
		seen[c] = true
		out = append(out, c)
	}
	return out
}

// HasMenu 是否拥有指定菜单。
func HasMenu(menus []string, code string) bool {
	if alias, ok := legacyAlias[code]; ok {
		code = alias
	}
	for _, m := range menus {
		if m == AuthorityAll || m == code {
			return true
		}
	}
	return false
}

// HasAny 是否拥有任一菜单。
func HasAny(menus []string, codes ...string) bool {
	for _, c := range codes {
		if HasMenu(menus, c) {
			return true
		}
	}
	return false
}
