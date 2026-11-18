import request from '@/utils/request'

export function queryEvents(params) {
  return request({ url: '/api/v1/events', method: 'get', params })
}

export function getEvent(id) {
  return request({ url: `/api/v1/events/${id}`, method: 'get' })
}
