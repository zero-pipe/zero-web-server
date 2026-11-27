import request from '@/utils/request'

// 通用通道API

export function queryOne(id) {
  return request({
    method: 'get',
    url: '/api/v1/common/channel/one',
    params: {
      id: id
    }
  })
}

// 按统一通道内码取国标通道（统一通道列表进入编辑时用）
export function queryOneByCode(internalCode) {
  return request({
    method: 'get',
    url: '/api/v1/common/channel/one',
    params: {
      internalCode: internalCode
    }
  })
}

export function getIndustryList() {
  return request({
    method: 'get',
    url: '/api/v1/common/channel/industry/list'
  })
}

export function getTypeList() {
  return request({
    method: 'get',
    url: '/api/v1/common/channel/type/list'
  })
}

export function getNetworkIdentificationList() {
  return request({
    method: 'get',
    url: '/api/v1/common/channel/network/identification/list'
  })
}

export function update(data) {
  return request({
    method: 'post',
    url: '/api/v1/common/channel/update',
    data: data
  })
}

export function reset(data) {
  return request({
    method: 'post',
    url: '/api/v1/common/channel/reset',
    data: data
  })
}

export function add(data) {
  return request({
    method: 'post',
    url: '/api/v1/common/channel/add',
    data: data
  })
}

export function getList(params) {
  const { page, count, query, online, hasRecordPlan, channelType, civilCode, parentDeviceId } = params
  return request({
    method: 'get',
    url: '/api/v1/common/channel/list',
    params: {
      page: page,
      count: count,
      channelType: channelType,
      query: query,
      online: online,
      hasRecordPlan: hasRecordPlan,
      civilCode: civilCode,
      parentDeviceId: parentDeviceId
    }
  })
}

export function getCivilCodeList(params) {
  const { page, count, channelType, query, online, civilCode } = params
  return request({
    method: 'get',
    url: '/api/v1/common/channel/civilcode/list',
    params: {
      page: page,
      count: count,
      channelType: channelType,
      query: query,
      online: online,
      civilCode: civilCode
    }
  })
}

export function getUnusualCivilCodeList(params) {
  const { page, count, channelType, query, online } = params
  return request({
    method: 'get',
    url: '/api/v1/common/channel/civilCode/unusual/list',
    params: {
      page: page,
      count: count,
      channelType: channelType,
      query: query,
      online: online
    }
  })
}

export function getUnusualParentList(params) {
  const { page, count, channelType, query, online } = params
  return request({
    method: 'get',
    url: '/api/v1/common/channel/parent/unusual/list',
    params: {
      page: page,
      count: count,
      channelType: channelType,
      query: query,
      online: online
    }
  })
}

export function clearUnusualCivilCodeList(params) {
  const { all, channelIds } = params
  return request({
    method: 'post',
    url: '/api/v1/common/channel/civilCode/unusual/clear',
    data: {
      all: all,
      channelIds: channelIds
    }
  })
}

export function clearUnusualParentList(params) {
  const { all, channelIds } = params
  return request({
    method: 'post',
    url: '/api/v1/common/channel/parent/unusual/clear',
    data: {
      all: all,
      channelIds: channelIds
    }
  })
}

export function getParentList(params) {
  const { page, count, channelType, query, online, groupDeviceId } = params
  return request({
    method: 'get',
    url: '/api/v1/common/channel/parent/list',
    params: {
      page: page,
      count: count,
      channelType: channelType,
      query: query,
      online: online,
      groupDeviceId: groupDeviceId
    }
  })
}

export function addToRegion(params) {
  const { civilCode, channelIds } = params
  return request({
    method: 'post',
    url: '/api/v1/common/channel/region/add',
    data: {
      civilCode: civilCode,
      channelIds: channelIds
    }
  })
}

export function deleteFromRegion(channels) {
  return request({
    method: 'post',
    url: '/api/v1/common/channel/region/delete',
    data: {
      channelIds: channels
    }
  })
}

export function addDeviceToRegion(params) {
  const { civilCode, deviceIds } = params
  return request({
    method: 'post',
    url: '/api/v1/common/channel/region/device/add',
    data: {
      civilCode: civilCode,
      deviceIds: deviceIds
    }
  })
}

export function deleteDeviceFromRegion(deviceIds) {
  return request({
    method: 'post',
    url: '/api/v1/common/channel/region/device/delete',
    data: {
      deviceIds: deviceIds
    }
  })
}

export function addToGroup(params) {
  const { parentId, businessGroup, channelIds } = params
  return request({
    method: 'post',
    url: '/api/v1/common/channel/group/add',
    data: {
      parentId: parentId,
      businessGroup: businessGroup,
      channelIds: channelIds
    }
  })
}

export function deleteFromGroup(channels) {
  return request({
    method: 'post',
    url: '/api/v1/common/channel/group/delete',
    data: {
      channelIds: channels
    }
  })
}

export function addDeviceToGroup(params) {
  const { parentId, businessGroup, deviceIds } = params
  return request({
    method: 'post',
    url: '/api/v1/common/channel/group/device/add',
    data: {
      parentId: parentId,
      businessGroup: businessGroup,
      deviceIds: deviceIds
    }
  })
}

export function deleteDeviceFromGroup(deviceIds) {
  return request({
    method: 'post',
    url: '/api/v1/common/channel/group/device/delete',
    data: {
      deviceIds: deviceIds
    }
  })
}

// 通道操作（播放/云台/预置位/回放）已统一收敛到 /api/v1/resources/{resourceId}/*，
// 见 api/asset.js（resourcePtz/resourcePreset*/resourceCommand）与 api/playback.js。
// 本文件只保留无重复语义的通道管理类接口（list/add/update/map/region/group/civilcode 等）。

export function getAllForMap({ query, online, hasRecordPlan, channelType }) {
  return request({
    method: 'get',
    url: '/api/v1/common/channel/map/list',
    params: {
      query: query,
      online: online,
      hasRecordPlan: hasRecordPlan,
      channelType: channelType
    }
  })
}
export function saveLevel(data) {
  return request({
    method: 'post',
    url: '/api/v1/common/channel/map/save-level',
    data: data
  })
}
export function resetLevel() {
  return request({
    method: 'post',
    url: '/api/v1/common/channel/map/reset-level'
  })
}
export function clearThin(id) {
  return request({
    method: 'get',
    url: '/api/v1/common/channel/map/thin/clear',
    params: {
      id: id
    }
  })
}
export function thinProgress(id) {
  return request({
    method: 'get',
    url: '/api/v1/common/channel/map/thin/progress',
    params: {
      id: id
    }
  })
}
export function saveThin(id) {
  return request({
    method: 'get',
    url: '/api/v1/common/channel/map/thin/save',
    params: {
      id: id
    }
  })
}
export function drawThin(data) {
  return request({
    method: 'post',
    url: '/api/v1/common/channel/map/thin/draw',
    data: data
  })
}
export function test() {
  return request({
    method: 'get',
    url: '/api/v1/sy/camera/list/ids',
    params: {
      deviceIds: ['a', 'b', 'c'].join(','),
      geoCoordSys: 'GCJ02',
      traditional: true
    }
  })
}
