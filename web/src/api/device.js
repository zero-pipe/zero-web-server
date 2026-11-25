import request from '@/utils/request'

// 国标设备 API（统一设备管理的 gb 子组）

export function queryDeviceSyncStatus(deviceId) {
  return request({
    method: 'get',
    url: `/api/v1/devices/gb/query/sync_status`,
    params: {
      deviceId: deviceId
    }
  })
}

export function queryDevices(params) {
  const { page, count, query, status } = params
  return request({
    method: 'get',
    url: `/api/v1/devices/gb/query/devices`,
    params: {
      page: page,
      count: count,
      query: query,
      status: status
    }
  })
}

export function sync(deviceId) {
  return request({
    method: 'get',
    url: `/api/v1/devices/gb/query/devices/${deviceId}/sync`
  })
}

export function setGuard(deviceId) {
  return request({
    method: 'get',
    url: `/api/v1/devices/gb/control/guard`,
    params: {
      deviceId: deviceId,
      guardCmd: 'SetGuard'
    }
  })
}

export function resetGuard(deviceId) {
  return request({
    method: 'get',
    url: `/api/v1/devices/gb/control/guard`,
    params: {
      deviceId: deviceId,
      guardCmd: 'ResetGuard'
    }
  })
}

export function subscribeCatalog(params) {
  const { id, cycle } = params
  return request({
    method: 'get',
    url: `/api/v1/devices/gb/query/subscribe/catalog`,
    params: {
      id: id,
      cycle: cycle
    }
  })
}

export function subscribeMobilePosition(params) {
  const { id, cycle, interval } = params
  return request({
    method: 'get',
    url: `/api/v1/devices/gb/query/subscribe/mobile-position`,
    params: {
      id: id,
      cycle: cycle,
      interval: interval
    }
  })
}

export function subscribeForAlarm(params) {
  const { id, cycle } = params
  return request({
    method: 'get',
    url: '/api/v1/devices/gb/query/subscribe/alarm',
    params: {
      id: id,
      cycle: cycle
    }
  })
}

export function queryHasStreamChannels(params) {
  const { page, count, query } = params
  return request({
    method: 'get',
    url: `/api/v1/devices/gb/query/streams`,
    params: {
      page: page,
      count: count,
      query: query
    }
  })
}

export function getKeepaliveTimeStatistics({ deviceId, count }) {
  return request({
    method: 'get',
    url: '/api/v1/devices/gb/query/statistics/keepalive',
    params: {
      deviceId: deviceId,
      count: count
    }
  })
}

export function getRegisterTimeStatistics({ deviceId, count }) {
  return request({
    method: 'get',
    url: '/api/v1/devices/gb/query/statistics/register',
    params: {
      deviceId: deviceId,
      count: count
    }
  })
}

export function dragZoomIn(params) {
  return request({
    method: 'get',
    url: '/api/v1/devices/gb/control/drag_zoom/zoom_in',
    params
  })
}

export function dragZoomOut(params) {
  return request({
    method: 'get',
    url: '/api/v1/devices/gb/control/drag_zoom/zoom_out',
    params
  })
}
