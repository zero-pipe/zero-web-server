import Cookies from 'js-cookie'

const TokenKey = 'zws_token'
const NameKey = 'zws_username'
const serverIdKey = 'zws_server_id'
const MenusKey = 'zws_menus'
const RoleIdKey = 'zws_role_id'
const expires = 30

export function getToken() {
  return Cookies.get(TokenKey)
}

export function setToken(token) {
  return Cookies.set(TokenKey, token, { expires: expires })
}

export function removeToken() {
  return Cookies.remove(TokenKey)
}

export function getName() {
  return Cookies.get(NameKey)
}

export function setName(name) {
  return Cookies.set(NameKey, name, { expires: expires })
}

export function removeName() {
  return Cookies.remove(NameKey)
}

export function getServerId() {
  return Cookies.get(serverIdKey)
}

export function setServerId(serverId) {
  return Cookies.set(serverIdKey, serverId, { expires: expires })
}

export function removeServerId() {
  return Cookies.remove(serverIdKey)
}

export function getMenus() {
  try {
    const raw = localStorage.getItem(MenusKey)
    return raw ? JSON.parse(raw) : []
  } catch (e) {
    return []
  }
}

export function setMenus(menus) {
  localStorage.setItem(MenusKey, JSON.stringify(menus || []))
}

export function removeMenus() {
  localStorage.removeItem(MenusKey)
}

export function getRoleId() {
  const v = localStorage.getItem(RoleIdKey)
  return v ? parseInt(v, 10) : 0
}

export function setRoleId(id) {
  localStorage.setItem(RoleIdKey, String(id || 0))
}

export function removeRoleId() {
  localStorage.removeItem(RoleIdKey)
}
