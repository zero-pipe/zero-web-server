import request from '@/utils/request'

export function queryEvents(params) {
  return request({ url: '/api/v1/events', method: 'get', params })
}

export function getEvent(id) {
  return request({ url: `/api/v1/events/${id}`, method: 'get' })
}

export function getEventStats() {
  return request({ url: '/api/v1/events/stats', method: 'get' })
}

export function getEventTypes() {
  return request({ url: '/api/v1/events/types', method: 'get' })
}

// 按时间范围清空事件；all=true 时删除全部（忽略范围）。
export function clearEvents(params) {
  return request({ url: '/api/v1/events/clear', method: 'post', data: params })
}
