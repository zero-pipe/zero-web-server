import request from '@/utils/request'

export function discoverDevices(timeout = 5) {
  return request({
    url: '/api/v1/devices/onvif/device/discover',
    method: 'post',
    params: { timeout }
  })
}

export function queryDevices(params) {
  return request({
    url: '/api/v1/devices/onvif/device/query',
    method: 'get',
    params
  })
}

export function addDevice(data) {
  return request({
    url: '/api/v1/devices/onvif/device/add',
    method: 'post',
    data
  })
}

export function deleteDevice(id) {
  return request({
    url: `/api/v1/devices/onvif/device/delete/${id}`,
    method: 'delete'
  })
}

export function syncDevice(id) {
  return request({
    url: `/api/v1/devices/onvif/device/sync/${id}`,
    method: 'post'
  })
}

export function probeDevices() {
  return request({
    url: '/api/v1/devices/onvif/device/probe',
    method: 'post'
  })
}

// 事件订阅开关：与 GB 报警、插件布防在界面上统一为「勾选框 + 事件」
export function getEventSubscribe(id) {
  return request({
    url: '/api/v1/devices/onvif/device/event-subscribe',
    method: 'get',
    params: { id: Number(id) }
  })
}

export function setEventSubscribe(id, eventSubscribe) {
  return request({
    url: '/api/v1/devices/onvif/device/event-subscribe',
    method: 'post',
    data: { id: Number(id), eventSubscribe }
  })
}

export function queryChannels(params) {
  return request({
    url: '/api/v1/devices/onvif/channel/query',
    method: 'get',
    params
  })
}

export function updateChannel(data) {
  return request({
    url: '/api/v1/devices/onvif/channel/update',
    method: 'post',
    data
  })
}

export function startPlay(channelId, profileToken) {
  return request({
    url: '/api/v1/devices/onvif/play/start',
    method: 'get',
    params: {
      channelId,
      ...(profileToken ? { profileToken } : {})
    }
  })
}

export function stopPlay(channelId, profileToken) {
  return request({
    url: '/api/v1/devices/onvif/play/stop',
    method: 'get',
    params: {
      channelId,
      ...(profileToken ? { profileToken } : {})
    }
  })
}

export function ptzControl(data) {
  return request({
    url: '/api/v1/devices/onvif/ptz/control',
    method: 'post',
    data
  })
}
