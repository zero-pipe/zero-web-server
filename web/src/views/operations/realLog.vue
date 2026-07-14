<template>
  <div class="real-log-page app-container">
    <showLog ref="recordVideoPlayer" :remote-url="wsUrl" />
  </div>
</template>

<script>
import showLog from './showLog.vue'

export default {
  name: 'OperationsRealLog',
  components: { showLog },
  data() {
    return {
      wsUrl: this.buildWsUrl()
    }
  },
  methods: {
    buildWsUrl() {
      // 开发环境直连后端：webpack 代理会破坏 WebSocket 帧（RSV1 错误）
      if (process.env.NODE_ENV === 'development') {
        const port = process.env.VUE_APP_BACKEND_PORT || '18080'
        return `ws://127.0.0.1:${port}/channel/log`
      }
      if (location.protocol === 'https:') {
        return `wss://${window.location.host}/channel/log`
      }
      return `ws://${window.location.host}/channel/log`
    }
  }
}
</script>

<style scoped>
.real-log-page {
  height: calc(100vh - 84px);
  max-height: calc(100vh - 84px);
  padding: 12px 16px 12px;
  box-sizing: border-box;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}
</style>
