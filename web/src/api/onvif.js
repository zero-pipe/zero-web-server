import request from '@/utils/request'

export function discoverDevices(timeout = 5) {
  return request({
    url: '/api/onvif/device/discover',
    method: 'post',
    params: { timeout }
  })
}

export function queryDevices(params) {
  return request({
    url: '/api/onvif/device/query',
    method: 'get',
    params
  })
}

export function addDevice(data) {
  return request({
    url: '/api/onvif/device/add',
    method: 'post',
    data
  })
}

export function deleteDevice(id) {
  return request({
    url: `/api/onvif/device/delete/${id}`,
    method: 'delete'
  })
}

export function syncDevice(id) {
  return request({
    url: `/api/onvif/device/sync/${id}`,
    method: 'post'
  })
}

export function probeDevices() {
  return request({
    url: '/api/onvif/device/probe',
    method: 'post'
  })
}

export function queryChannels(params) {
  return request({
    url: '/api/onvif/channel/query',
    method: 'get',
    params
  })
}

export function startPlay(channelId) {
  return request({
    url: '/api/onvif/play/start',
    method: 'get',
    params: { channelId }
  })
}

export function stopPlay(channelId) {
  return request({
    url: '/api/onvif/play/stop',
    method: 'get',
    params: { channelId }
  })
}

export function ptzControl(data) {
  return request({
    url: '/api/onvif/ptz/control',
    method: 'post',
    data
  })
}
