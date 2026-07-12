<template>
  <div id="log" class="app-container">
    <div style="height: calc(100vh - 124px);">
      <showLog ref="recordVideoPlayer" :remote-url="wsUrl" />
    </div>
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
