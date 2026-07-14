import request from '@/utils/request'

export function querySubordinate(params) {
  return request({
    method: 'get',
    url: '/api/subordinate/query',
    params
  })
}

export function addSubordinate(data) {
  return request({
    method: 'post',
    url: '/api/subordinate/add',
    data
  })
}

export function updateSubordinate(id, data) {
  return request({
    method: 'post',
    url: `/api/subordinate/update/${id}`,
    data
  })
}

export function removeSubordinate(id) {
  return request({
    method: 'delete',
    url: `/api/subordinate/delete/${id}`
  })
}
