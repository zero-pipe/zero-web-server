import request from '@/utils/request'

// 分组API

export function update(data) {
  return request({
    method: 'post',
    url: '/api/v1/group/update',
    data: data
  })
}
export function add(data) {
  return request({
    method: 'post',
    url: '/api/v1/group/add',
    data: data
  })
}
export function getTreeList({ query, parent, hasChannel, page, count }) {
  return request({
    method: 'get',
    url: `/api/v1/group/tree/list`,
    params: {
      query: query,
      parent: parent,
      hasChannel: hasChannel,
      page: page,
      count: count
    }
  })
}
export function deleteGroup(id) {
  return request({
    method: 'delete',
    url: `/api/v1/group/delete`,
    params: {
      id: id
    }
  })
}
export function getPath(params) {
  const { deviceId, businessGroup } = params
  return request({
    method: 'get',
    url: `/api/v1/group/path`,
    params: {
      deviceId: deviceId,
      businessGroup: businessGroup
    }
  })
}
export function queryTree(params) {
  const { page, count, query } = params
  return request({
    method: 'get',
    url: `/api/v1/group/tree/query`,
    params: {
      query: query,
      page: page,
      count: count
    }
  })
}
