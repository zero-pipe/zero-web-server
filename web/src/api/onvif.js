import request from '@/utils/request'

// ONVIF 专属动作（统一设备管理的 onvif 子组）

export function discoverDevices(timeout = 5) {
  return request({
    url: '/api/v1/devices/onvif/device/discover',
    method: 'post',
    params: { timeout }
  })
}

// 事件订阅开关：与 GB 报警、插件布防在界面上统一为「勾选框 + 事件」
export function setEventSubscribe(id, eventSubscribe) {
  return request({
    url: '/api/v1/devices/onvif/device/event-subscribe',
    method: 'post',
    data: { id: Number(id), eventSubscribe }
  })
}
