import request from '@/utils/request'

export function getList(params) {
  return request({
    url: '/api/mock/table/list',
    method: 'get',
    params
  })
}
