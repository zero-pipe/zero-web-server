import request from '@/utils/request'

export function getAll() {
  return request({
    method: 'get',
    url: '/api/v1/role/all'
  })
}

export function getMenus() {
  return request({
    method: 'get',
    url: '/api/v1/role/menus'
  })
}

export function addRole(data) {
  return request({
    method: 'post',
    url: '/api/v1/role/add',
    data
  })
}

export function updateRole(data) {
  return request({
    method: 'post',
    url: '/api/v1/role/update',
    data
  })
}

export function deleteRole(id) {
  return request({
    method: 'delete',
    url: '/api/v1/role/delete',
    params: { id }
  })
}
