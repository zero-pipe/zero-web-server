import request from '@/utils/request'

// 插件注册表（非设备维度）
export function queryPlugins() {
  return request({ url: '/api/v1/plugins', method: 'get' })
}

// 插件设备专属动作（统一设备管理的 plugin 子组）
export function addIpcDevice(data) {
  return request({ url: '/api/v1/devices/plugin/devices', method: 'post', data })
}

export function syncIpcDevice(id) {
  return request({ url: '/api/v1/devices/plugin/devices/sync', method: 'post', params: { id } })
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
