import request from '@/utils/request'

// 存储节点列表（供录像计划「录制目标」下拉）
export function listStorageNodes() {
  return request({
    method: 'get',
    url: '/api/v1/storage/node/list',
    silent: true
  })
}
