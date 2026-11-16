import request from '@/utils/request'

export function discoverDevices(timeout = 5) {
  return request({
    url: '/api/v1/onvif/device/discover',
    method: 'post',
    params: { timeout }
  })
}

export function queryDevices(params) {
  return request({
    url: '/api/v1/onvif/device/query',
    method: 'get',
    params
  })
}

export function addDevice(data) {
  return request({
    url: '/api/v1/onvif/device/add',
    method: 'post',
    data
  })
}

export function deleteDevice(id) {
  return request({
    url: `/api/v1/onvif/device/delete/${id}`,
    method: 'delete'
  })
}

export function syncDevice(id) {
  return request({
    url: `/api/v1/onvif/device/sync/${id}`,
    method: 'post'
  })
}

export function probeDevices() {
  return request({
    url: '/api/v1/onvif/device/probe',
    method: 'post'
  })
}

export function queryChannels(params) {
  return request({
    url: '/api/v1/onvif/channel/query',
    method: 'get',
    params
  })
}

export function updateChannel(data) {
  return request({
    url: '/api/v1/onvif/channel/update',
    method: 'post',
    data
  })
}

export function startPlay(channelId, profileToken) {
  return request({
    url: '/api/v1/onvif/play/start',
    method: 'get',
    params: {
      channelId,
      ...(profileToken ? { profileToken } : {})
    }
  })
}

export function stopPlay(channelId, profileToken) {
  return request({
    url: '/api/v1/onvif/play/stop',
    method: 'get',
    params: {
      channelId,
      ...(profileToken ? { profileToken } : {})
    }
  })
}

export function ptzControl(data) {
  return request({
    url: '/api/v1/onvif/ptz/control',
    method: 'post',
    data
  })
}
