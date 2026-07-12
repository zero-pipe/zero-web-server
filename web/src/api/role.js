import request from '@/utils/request'

export function getAll() {
  return request({
    method: 'get',
    url: '/api/role/all'
  })
}

export function getMenus() {
  return request({
    method: 'get',
    url: '/api/role/menus'
  })
}

export function addRole(data) {
  return request({
    method: 'post',
    url: '/api/role/add',
    data
  })
}

export function updateRole(data) {
  return request({
    method: 'post',
    url: '/api/role/update',
    data
  })
}

export function deleteRole(id) {
  return request({
    method: 'delete',
    url: '/api/role/delete',
    params: { id }
  })
}
