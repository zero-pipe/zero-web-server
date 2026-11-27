import request from '@/utils/request'

export function queryChannels(params) {
  return request({ url: '/api/v1/channels', method: 'get', params })
}

export function queryAssets(params) {
  return request({ url: '/api/v1/assets', method: 'get', params })
}

export function queryAssetTree(params) {
  return request({ url: '/api/v1/assets/tree', method: 'get', params })
}

export function updateAsset(id, data) {
  return request({ url: `/api/v1/assets/${id}`, method: 'put', data })
}

export function updateChannel(code, data) {
  return request({ url: `/api/v1/channels/${code}`, method: 'put', data })
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

// ---- 通道操作统一族（按 resourceId）----
// 云台方向/变倍：command=方向或变倍指令，速度上限 水平/垂直 255、变倍 16。
export function resourcePtz(id, { command, horizonSpeed, verticalSpeed, zoomSpeed } = {}) {
  return request({
    url: `/api/v1/resources/${id}/ptz`,
    method: 'get',
    params: { command, horizonSpeed, verticalSpeed, zoomSpeed }
  })
}

// 预置位（presetId 一律字符串）。查询为只读辅助功能：失败不弹全局错误，由组件内联提示。
export function resourcePresetQuery(id) {
  return request({ url: `/api/v1/resources/${id}/preset/query`, method: 'get', silent: true })
}
export function resourcePresetCall(id, presetId) {
  return request({ url: `/api/v1/resources/${id}/preset/call`, method: 'get', params: { presetId } })
}
export function resourcePresetAdd(id, presetId) {
  return request({ url: `/api/v1/resources/${id}/preset/add`, method: 'get', params: { presetId } })
}
export function resourcePresetDelete(id, presetId) {
  return request({ url: `/api/v1/resources/${id}/preset/delete`, method: 'get', params: { presetId } })
}

// 通用命令透传（聚焦/光圈等镜头能力，platform 不解析 params 语义，见接口契约 §4.3）。
export function resourceCommand(id, cmd, params) {
  return request({ url: `/api/v1/resources/${id}/command/${cmd}`, method: 'post', data: { params } })
}

// ---- 云端录像回放（按 resourceId + 绝对毫秒）----
export function resourceCloudSegments(id, { start, end } = {}) {
  return request({ url: `/api/v1/resources/${id}/playback/cloud/segments`, method: 'get', params: { start, end } })
}
export function resourceCloudStart(id, { recordId, startTime } = {}) {
  return request({ url: `/api/v1/resources/${id}/playback/cloud/start`, method: 'get', params: { recordId, startTime } })
}
export function resourceCloudSeek(id, { recordId, seekTime } = {}) {
  return request({ url: `/api/v1/resources/${id}/playback/cloud/seek`, method: 'get', params: { recordId, seekTime } })
}
export function resourceCloudStop(id, { recordId } = {}) {
  return request({ url: `/api/v1/resources/${id}/playback/cloud/stop`, method: 'get', params: { recordId } })
}
