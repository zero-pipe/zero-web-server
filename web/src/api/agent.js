import request from '@/utils/request'

export function queryAgents() {
  return request({ url: '/api/v1/agents', method: 'get' })
}

export function queryAgentPlugins(agentId) {
  return request({ url: `/api/v1/agents/${agentId}/plugins`, method: 'get' })
}

export function setAgentPluginState(agentId, pluginId, action) {
  return request({ url: `/api/v1/agents/${agentId}/plugins/${pluginId}/${action}`, method: 'post' })
}

// 删除内置/本地插件（须先删设备、再禁用）
export function deleteAgentPlugin(agentId, pluginId) {
  return request({ url: `/api/v1/agents/${agentId}/plugins/${pluginId}`, method: 'delete' })
}
