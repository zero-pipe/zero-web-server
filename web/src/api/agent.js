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
