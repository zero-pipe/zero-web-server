/**
 * 菜单权限工具：与后端 rbac 菜单码、layout/menu.js 的 id 对齐
 */
import { primaryMenus } from '@/layout/menu'

const PATH_RULES = [
  { prefix: '/map', menu: 'map' },
  { prefix: '/live', menu: 'live' },
  { prefix: '/device', menu: 'device' },
  { prefix: '/onvifDevice', menu: 'device' },
  { prefix: '/jtDevice', menu: 'device' },
  { prefix: '/channel', menu: 'device' },
  { prefix: '/push', menu: 'device' },
  { prefix: '/proxy', menu: 'device' },
  { prefix: '/commonChannel', menu: 'org' },
  { prefix: '/recordPlan', menu: 'record' },
  { prefix: '/cloudRecord', menu: 'record' },
  { prefix: '/alarm', menu: 'alarm' },
  { prefix: '/dashboard', menu: 'ops' },
  { prefix: '/operations', menu: 'ops' },
  { prefix: '/mediaServer', menu: 'system' },
  { prefix: '/gbConfig', menu: 'system' },
  { prefix: '/platform', menu: 'system' },
  { prefix: '/user', menu: 'user' },
  { prefix: '/role', menu: 'user' }
]

export function hasMenu(menus, code) {
  if (!menus || !menus.length) return false
  if (menus.indexOf('*') >= 0) return true
  return menus.indexOf(code) >= 0
}

export function filterMenusByPerm(menus, perms) {
  if (!perms || !perms.length) return []
  if (perms.indexOf('*') >= 0) return menus
  return (menus || []).filter(m => hasMenu(perms, m.id))
}

export function menuCodeByPath(path) {
  const p = (path || '').split('?')[0]
  for (let i = 0; i < PATH_RULES.length; i++) {
    const r = PATH_RULES[i]
    if (p === r.prefix || p.startsWith(r.prefix + '/')) {
      return r.menu
    }
  }
  return null
}

export function canAccessPath(menus, path) {
  const code = menuCodeByPath(path)
  if (!code) return true // 无映射的页面不拦（如 404）
  return hasMenu(menus, code)
}

export function firstAllowedPath(menus) {
  const filtered = filterMenusByPerm(primaryMenus, menus)
  if (!filtered.length) return '/login'
  const first = filtered[0]
  if (first.path) return first.path
  if (first.children && first.children.length) return first.children[0].path
  return '/map'
}

/** 解析角色 authority 为菜单码数组（展示用） */
export function parseAuthorityMenus(authority, roleId) {
  if (roleId === 1 || authority === '*' || authority === '0') {
    return primaryMenus.map(m => m.id)
  }
  if (!authority) return []
  try {
    const arr = JSON.parse(authority)
    return Array.isArray(arr) ? arr : []
  } catch (e) {
    return String(authority).split(',').map(s => s.trim()).filter(Boolean)
  }
}
