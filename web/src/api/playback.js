import request from '@/utils/request'

// 设备回放（统一资源口径，按 resourceId）

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
