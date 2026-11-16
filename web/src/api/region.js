import request from '@/utils/request'

// 行政区划API

export function getTreeList(params) {
  const {query, parent, hasChannel} = params
  return request({
    method: 'get',
    url: `/api/v1/region/tree/list`,
    params: {
      query: query,
      parent: parent,
      hasChannel: hasChannel
    }
  })
}

export function deleteRegion(id) {
  return request({
    method: "delete",
    url: `/api/v1/region/delete`,
    params: {
      id: id,
    }
  })
}

export function description(civilCode) {
  return request({
    method: 'get',
    url: `/api/v1/region/description`,
    params: {
      civilCode: civilCode,
    }
  })
}

export function addByCivilCode(civilCode) {
  return request({
    method: 'get',
    url: `/api/v1/region/addByCivilCode`,
    params: {
      civilCode: civilCode,
    }
  })
}

export function queryChildListInBase(parent) {
  return request({
    method: 'get',
    url: "/api/v1/region/base/child/list",
    params: {
      parent: parent,
    }
  })
}

export function update(data) {
  return request({
    method: 'post',
    url: "/api/v1/region/update",
    data: data

  })
}

export function add(data) {
  return request({
    method: 'post',
    url: "/api/v1/region/add",
    data: data
  })
}

export function queryPath(deviceId) {
  return request({
    method: 'get',
    url: `/api/v1/region/path`,
    params: {
      deviceId: deviceId,
    }
  })
}
export function queryTree(params) {
  const { page, count, query } = params
  return request({
    method: 'get',
    url: `/api/v1/region/tree/query`,
    params: {
      query: query,
      page: page,
      count: count
    }
  })
}

