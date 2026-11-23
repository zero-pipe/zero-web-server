import request from '@/utils/request'

// 媒体会话：各在线媒体节点上的实时流只读视图（点播产生，可停止）。
// silent：本页 2s 轮询，失败不弹全局提示。
export function queryMediaSessions() {
  return request({ url: '/api/v1/media-sessions', method: 'get', silent: true })
}

export function stopMediaSession(data) {
  return request({ url: '/api/v1/media-sessions/stop', method: 'post', data })
}

// 新建临时拉流代理（保留的手动运维能力）。
export function addMediaProxy(data) {
  return request({ url: '/api/v1/media-sessions/proxy', method: 'post', data })
}
