import request from '@/utils/request'

export function list(params) {
  const { page, count } = params
  return request({
    method: 'get',
    url: '/api/v1/accessToken/list',
    params: { page, count }
  })
}

export function add(params) {
  const { name, token, remark, enabled } = params
  return request({
    method: 'post',
    url: '/api/v1/accessToken/add',
    params: { name, token, remark, enabled: enabled ? 1 : 0 }
  })
}

export function update(params) {
  const { id, name, remark, enabled } = params
  return request({
    method: 'post',
    url: '/api/v1/accessToken/update',
    params: { id, name, remark, enabled: enabled ? 1 : 0 }
  })
}

export function enable(id) {
  return request({ method: 'post', url: `/api/v1/accessToken/enable?id=${id}` })
}

export function disable(id) {
  return request({ method: 'post', url: `/api/v1/accessToken/disable?id=${id}` })
}

export function reset(id) {
  return request({ method: 'post', url: `/api/v1/accessToken/reset?id=${id}` })
}

export function remove(id) {
  return request({ method: 'delete', url: `/api/v1/accessToken/delete?id=${id}` })
}

export function first() {
  return request({ method: 'get', url: '/api/v1/accessToken/first' })
}
