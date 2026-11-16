import request from '@/utils/request'

// 插件实例列表
export function queryPlugins() {
  return request({ url: '/api/v1/plugins', method: 'get' })
}

// 插件设备
export function queryIpcDevices() {
  return request({ url: '/api/v1/ipc/devices', method: 'get' })
}

export function addIpcDevice(data) {
  return request({ url: '/api/v1/ipc/devices', method: 'post', data })
}

export function deleteIpcDevice(id) {
  return request({ url: '/api/v1/ipc/devices', method: 'delete', params: { id } })
}

export function syncIpcDevice(id) {
  return request({ url: '/api/v1/ipc/devices/sync', method: 'post', params: { id } })
}

export function queryIpcChannels(deviceId) {
  return request({ url: '/api/v1/ipc/channels', method: 'get', params: { deviceId } })
}

// 播放
export function startIpcLive(deviceId, channelId) {
  return request({ url: '/api/v1/ipc/play/start', method: 'get', params: { deviceId, channelId } })
}

export function stopIpcLive(deviceId, channelId) {
  return request({ url: '/api/v1/ipc/play/stop', method: 'get', params: { deviceId, channelId } })
}

export function ipcPtz(data) {
  return request({ url: '/api/v1/ipc/ptz', method: 'post', data })
}
