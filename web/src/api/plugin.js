import request from '@/utils/request'

// 插件实例列表
export function queryPlugins() {
  return request({ url: '/api/v1/plugins', method: 'get' })
}

// 插件设备
export function queryIpcDevices() {
  return request({ url: '/api/v1/devices/plugin/devices', method: 'get' })
}

export function addIpcDevice(data) {
  return request({ url: '/api/v1/devices/plugin/devices', method: 'post', data })
}

export function deleteIpcDevice(id) {
  return request({ url: '/api/v1/devices/plugin/devices', method: 'delete', params: { id } })
}

export function syncIpcDevice(id) {
  return request({ url: '/api/v1/devices/plugin/devices/sync', method: 'post', params: { id } })
}

export function queryIpcChannels(deviceId) {
  return request({ url: '/api/v1/devices/plugin/channels', method: 'get', params: { deviceId } })
}

// 播放
export function startIpcLive(deviceId, channelId) {
  return request({ url: '/api/v1/devices/plugin/play/start', method: 'get', params: { deviceId, channelId } })
}

export function stopIpcLive(deviceId, channelId) {
  return request({ url: '/api/v1/devices/plugin/play/stop', method: 'get', params: { deviceId, channelId } })
}

export function ipcPtz(data) {
  return request({ url: '/api/v1/devices/plugin/ptz', method: 'post', data })
}

// 布防 / 撤防
export function armIpcDevice(deviceId, eventTypes) {
  return request({ url: '/api/v1/devices/plugin/devices/arm', method: 'post', data: { deviceId: Number(deviceId), eventTypes } })
}

export function disarmIpcDevice(deviceId) {
  return request({ url: '/api/v1/devices/plugin/devices/disarm', method: 'post', data: { deviceId: Number(deviceId) } })
}

export function getIpcArmStatus(deviceId) {
  return request({ url: '/api/v1/devices/plugin/devices/arm/status', method: 'get', params: { deviceId } })
}
