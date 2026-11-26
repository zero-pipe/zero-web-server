import request from '@/utils/request'

// 自定义事件：平台不理解语义，仅按基础字段与 raw 展示/查询。
export function queryCustomEvents(params) {
  return request({ url: '/api/v1/custom-events', method: 'get', params })
}

export function getCustomEvent(id) {
  return request({ url: `/api/v1/custom-events/${id}`, method: 'get' })
}

export function getCustomEventTypes(deviceName) {
  return request({ url: '/api/v1/custom-events/types', method: 'get', params: { deviceName } })
}

// 按时间范围清空自定义事件；all=true 时删除全部（忽略范围）。
export function clearCustomEvents(params) {
  return request({ url: '/api/v1/custom-events/clear', method: 'post', data: params })
}
