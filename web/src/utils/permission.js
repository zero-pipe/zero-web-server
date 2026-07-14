/**
 * 菜单权限工具：与后端 rbac、layout/menu.js 能力域对齐
 */
import { primaryMenus, legacyMenuAlias } from '@/layout/menu'

const PATH_RULES = [
  { prefix: '/devices', menu: 'access' },
  { prefix: '/device', menu: 'access' },
  { prefix: '/onvifDevice', menu: 'access' },
  { prefix: '/jtDevice', menu: 'access' },
  { prefix: '/channel', menu: 'access' },
  { prefix: '/gbConfig', menu: 'access' },
  { prefix: '/platform', menu: 'access' },
  { prefix: '/mediaServer', menu: 'media' },
  { prefix: '/push', menu: 'media' },
  { prefix: '/proxy', menu: 'media' },
  { prefix: '/live', menu: 'media' },
  { prefix: '/objectStore', menu: 'storage' },
  { prefix: '/recordPlan', menu: 'storage' },
  { prefix: '/cloudRecord', menu: 'storage' },
  { prefix: '/commonChannel', menu: 'org' },
  { prefix: '/map', menu: 'observe' },
  { prefix: '/alarm', menu: 'observe' },
  { prefix: '/dashboard', menu: 'ops' },
  { prefix: '/operations', menu: 'ops' },
  { prefix: '/user', menu: 'user' },
  { prefix: '/role', menu: 'user' }
]

function resolveMenuCode(code) {
  return legacyMenuAlias[code] || code
}

export function hasMenu(menus, code) {
  if (!menus || !menus.length) return false
  if (menus.indexOf('*') >= 0) return true
  const want = resolveMenuCode(code)
  return menus.some(m => resolveMenuCode(m) === want)
}

export function filterMenusByPerm(menus, perms) {
  if (!perms || !perms.length) return []
  if (perms.indexOf('*') >= 0) return menus
  return (menus || []).filter(m => hasMenu(perms, m.id))
}

export function menuCodeByPath(path) {
  const p = (path || '').split('?')[0]
  for (const rule of PATH_RULES) {
    if (p === rule.prefix || p.startsWith(rule.prefix + '/')) {
      return rule.menu
    }
  }
  // fallback：用 primaryMenus 反查
  for (const item of primaryMenus) {
    if (item.path && (p === item.path || p.startsWith(item.path + '/'))) return item.id
    if (item.children) {
      const hit = item.children.find(c => p === c.path || p.startsWith(c.path + '/'))
      if (hit) return item.id
    }
  }
  return null
}

export function canAccessPath(perms, path) {
  if (!perms || !perms.length) return false
  if (perms.indexOf('*') >= 0) return true
  const code = menuCodeByPath(path)
  if (!code) return true
  return hasMenu(perms, code)
}

export function firstAllowedPath(menus) {
  const filtered = filterMenusByPerm(primaryMenus, menus)
  if (!filtered.length) return '/login'
  const first = filtered[0]
  if (first.path) return first.path
  if (first.children && first.children.length) return first.children[0].path
  return '/devices'
}

/** 解析角色 authority 为菜单码数组（展示用，旧码映射到新能力域） */
export function parseAuthorityMenus(authority, roleId) {
  if (roleId === 1 || authority === '*' || authority === '0') {
    return primaryMenus.map(m => m.id)
  }
  if (!authority) return []
  let arr = []
  try {
    const parsed = JSON.parse(authority)
    arr = Array.isArray(parsed) ? parsed : []
  } catch (e) {
    arr = String(authority).split(',').map(s => s.trim()).filter(Boolean)
  }
  const seen = {}
  const out = []
  arr.forEach(c => {
    const code = resolveMenuCode(c)
    if (!seen[code] && primaryMenus.some(m => m.id === code)) {
      seen[code] = true
      out.push(code)
    }
  })
  return out
}
