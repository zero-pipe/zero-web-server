import request from '@/utils/request'

// 插件包列表
export function queryPluginPackages() {
  return request({ url: '/api/v1/plugin-packages', method: 'get' })
}

export function getPluginPackage(id) {
  return request({ url: `/api/v1/plugin-packages/${id}`, method: 'get' })
}

// 上传插件包（multipart）
export function uploadPluginPackage(file) {
  const form = new FormData()
  form.append('file', file)
  return request({
    url: '/api/v1/plugin-packages',
    method: 'post',
    data: form,
    timeout: 120000
  })
}

export function deletePluginPackage(id) {
  return request({ url: `/api/v1/plugin-packages/${id}`, method: 'delete' })
}

// 安装到指定 Agent
export function installPluginPackage(id, agentId) {
  return request({ url: `/api/v1/plugin-packages/${id}/install`, method: 'post', params: { agentId } })
}

// 安装记录
export function queryPluginInstalls(agentId) {
  return request({ url: '/api/v1/plugin-installs', method: 'get', params: { agentId } })
}

export function uninstallPlugin(installId) {
  return request({ url: `/api/v1/plugin-installs/${installId}`, method: 'delete' })
}
