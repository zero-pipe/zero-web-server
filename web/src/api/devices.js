import request from '@/utils/request'

export function listDevices(params) {
  return request({
    url: '/api/devices',
    method: 'get',
    params
  })
}

export function createDevice(data) {
  return request({
    url: '/api/devices',
    method: 'post',
    data
  })
}

export function updateDevice(id, data) {
  return request({
    url: '/api/devices',
    method: 'put',
    params: { id },
    data
  })
}

export function deleteDevice(id) {
  return request({
    url: '/api/devices/delete',
    method: 'post',
    data: { id }
  })
}

export function syncDevice(id) {
  return request({
    url: '/api/devices/sync',
    method: 'post',
    params: { id }
  })
}
