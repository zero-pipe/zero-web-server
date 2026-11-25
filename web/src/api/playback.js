import request from '@/utils/request'

// 回放流播放API

export function play([deviceId, channelId, startTime, endTime]) {
  return request({
    method: 'get',
    url: '/api/v1/playback/start/' + deviceId + '/' + channelId + '?startTime=' + startTime + '&endTime=' + endTime
  })
}
export function resume(streamId) {
  return request({
    method: 'get',
    url: '/api/v1/playback/resume/' + streamId
  })
}
export function pause(streamId) {
  return request({
    method: 'get',
    url: '/api/v1/playback/pause/' + streamId
  })
}
export function setSpeed([streamId, speed]) {
  return request({
    method: 'get',
    url: `/api/v1/playback/speed/${streamId}/${speed}`
  })
}
export function stop(deviceId, channelId, streamId) {
  return request({
    method: 'get',
    url: '/api/v1/playback/stop/' + deviceId + '/' + channelId + '/' + streamId
  })
}
export function seek(streamId, seekTime) {
  return request({
    method: 'get',
    url: `/api/v1/playback/seek/${streamId}/${seekTime}`
  })
}

// ---- 统一资源口径（按 resourceId，前端推荐用这套）----
// 录像查询：GB 返回 RecordInfo{sumNum,recordList:[{startTime,endTime,...}]}
export function queryRecords(resourceId, { startTime, endTime, streamType } = {}) {
  return request({
    method: 'post',
    url: `/api/v1/resources/${resourceId}/playback/records`,
    data: { startTime, endTime, streamType: streamType || 0 }
  })
}
// 起回放流：返回 StreamContent（含 stream 与 flv/ws_flv/hls/rtc 等多出口）
export function startPlayback(resourceId, { startTime, endTime, stream } = {}) {
  return request({
    method: 'post',
    url: `/api/v1/resources/${resourceId}/playback/start`,
    data: { startTime, endTime, stream: stream || 'main' }
  })
}
export function stopPlayback(resourceId) {
  return request({
    method: 'post',
    url: `/api/v1/resources/${resourceId}/playback/stop`
  })
}
