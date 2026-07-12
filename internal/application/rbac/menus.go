package rbac

import (
	"encoding/json"
	"strings"
)

// 菜单权限码，与 web/src/layout/menu.js 的 primary id 对齐。
const (
	MenuMap    = "map"
	MenuLive   = "live"
	MenuDevice = "device"
	MenuOrg    = "org"
	MenuRecord = "record"
	MenuAlarm  = "alarm"
	MenuOps    = "ops"
	MenuSystem = "system"
	MenuUser   = "user"
)

// AllMenus 全部一级菜单权限。
var AllMenus = []string{
	MenuMap, MenuLive, MenuDevice, MenuOrg, MenuRecord,
	MenuAlarm, MenuOps, MenuSystem, MenuUser,
}

// MenuDefs 供前端角色配置勾选。
var MenuDefs = []struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}{
	{MenuMap, "电子地图"},
	{MenuLive, "分屏监控"},
	{MenuDevice, "设备管理"},
	{MenuOrg, "组织管理"},
	{MenuRecord, "录像管理"},
	{MenuAlarm, "报警管理"},
	{MenuOps, "运维管理"},
	{MenuSystem, "系统管理"},
	{MenuUser, "用户管理"},
}

const (
	AuthorityAll   = "*"
	AuthorityLegacy = "0" // 兼容旧 admin
)

// IsFullAccess authority 是否表示全部菜单。
func IsFullAccess(authority string) bool {
	a := strings.TrimSpace(authority)
	return a == AuthorityAll || a == AuthorityLegacy
}

// ParseMenus 解析角色 authority → 菜单码列表。
// roleID==1 或 * / 0：返回全部；否则解析 JSON 数组。
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
		// 兼容逗号分隔
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

// Normalize 去重并只保留合法菜单码。
func Normalize(codes []string) []string {
	valid := map[string]bool{}
	for _, m := range AllMenus {
		valid[m] = true
	}
	seen := map[string]bool{}
	out := make([]string, 0, len(codes))
	for _, c := range codes {
		c = strings.TrimSpace(c)
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
