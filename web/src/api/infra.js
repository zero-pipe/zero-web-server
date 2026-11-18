import request from '@/utils/request'

export function getInfraOverview() {
  return request({ url: '/api/v1/infra/overview', method: 'get' })
}

export function saveMySQLConfig(data) {
  return request({ url: '/api/v1/infra/mysql/save', method: 'post', data })
}

export function saveRedisConfig(data) {
  return request({ url: '/api/v1/infra/redis/save', method: 'post', data })
}

export function getMQTTConfig() {
  return request({ url: '/api/v1/infra/mqtt', method: 'get' })
}

export function saveMQTTConfig(data) {
  return request({ url: '/api/v1/infra/mqtt/save', method: 'post', data })
}

export function checkMQTTHealth() {
  return request({ url: '/api/v1/infra/mqtt/health', method: 'get' })
}

export function getPipelineStatus() {
  return request({ url: '/api/v1/event_pipeline/status', method: 'get' })
}
