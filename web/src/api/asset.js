import request from '@/utils/request'

export function queryChannels(params) {
  return request({ url: '/api/v1/channels', method: 'get', params })
}

export function queryAssets(params) {
  return request({ url: '/api/v1/assets', method: 'get', params })
}

export function getAssetDetail(id) {
  return request({ url: `/api/v1/assets/${id}`, method: 'get' })
}

export function syncAsset(id) {
  return request({ url: `/api/v1/assets/${id}/sync`, method: 'post' })
}

export function getResourceDetail(id) {
  return request({ url: `/api/v1/resources/${id}`, method: 'get' })
}

export function startResourceLive(id, profileId) {
  return request({ url: `/api/v1/resources/${id}/live/start`, method: 'post', data: { profileId } })
}

export function stopResourceLive(id) {
  return request({ url: `/api/v1/resources/${id}/live/stop`, method: 'post' })
}
