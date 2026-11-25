import request from '@/utils/request'

// 数据字典：类型
export function listDictTypes() {
  return request({ url: '/api/v1/dict/types', method: 'get' })
}
export function createDictType(data) {
  return request({ url: '/api/v1/dict/types', method: 'post', data })
}
export function updateDictType(id, data) {
  return request({ url: `/api/v1/dict/types/${id}`, method: 'put', data })
}
export function deleteDictType(id) {
  return request({ url: `/api/v1/dict/types/${id}`, method: 'delete' })
}

// 数据字典：项
export function listDictItems(type, enabledOnly) {
  return request({ url: '/api/v1/dict/items', method: 'get', params: { type, enabled: enabledOnly ? 1 : 0 } })
}
export function createDictItem(data) {
  return request({ url: '/api/v1/dict/items', method: 'post', data })
}
export function updateDictItem(id, data) {
  return request({ url: `/api/v1/dict/items/${id}`, method: 'put', data })
}
export function deleteDictItem(id) {
  return request({ url: `/api/v1/dict/items/${id}`, method: 'delete' })
}
