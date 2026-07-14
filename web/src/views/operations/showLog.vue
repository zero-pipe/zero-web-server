<template>
  <div class="log-shell">
    <div ref="panel" class="log-panel">
      <div class="log-toolbar">
        <span
          v-if="isRealtime"
          class="log-status-dot"
          :class="statusOk ? 'is-online' : 'is-offline'"
          :title="statusTip"
        />
        <el-input
          v-model="filter"
          size="mini"
          clearable
          prefix-icon="el-icon-search"
          placeholder="过滤关键字"
          class="log-filter-input"
        />
        <button
          type="button"
          class="log-scroll-btn"
          title="回到顶部"
          @click="scrollToTop"
        >
          <i class="el-icon-arrow-up" />
        </button>
        <button
          type="button"
          class="log-scroll-btn"
          title="滚到底部"
          @click="scrollToBottom"
        >
          <i class="el-icon-arrow-down" />
        </button>
      </div>
      <log-viewer
        v-if="winHeight > 0"
        ref="logViewer"
        :log="logData"
        :auto-scroll="true"
        :height="winHeight"
        class="log-viewer-panel"
      />
    </div>
    <div class="log-actions">
      <el-button size="mini" icon="el-icon-download" @click="downloadFile()">下载</el-button>
    </div>
  </div>
</template>

<script>
import moment from 'moment/moment'
import logViewer from '@femessage/log-viewer'
import stripAnsi from 'strip-ansi'
import request from '@/utils/request'

/** 实时日志页面只保留最近 N 行，避免刷屏 */
const REALTIME_KEEP_LINES = 50

export default {
  name: 'Log',
  components: { logViewer },
  props: ['fileUrl', 'remoteUrl', 'loadEnd'],
  data() {
    return {
      loading: true,
      winHeight: 0,
      data: [],
      filter: '',
      logData: '',
      websocket: null,
      statusText: '',
      statusOk: false,
      resizeObserver: null
    }
  },
  computed: {
    isRealtime() {
      return !!this.remoteUrl
    },
    statusTip() {
      if (this.statusOk) return '已连接'
      return this.statusText || '未连接'
    }
  },
  watch: {
    remoteUrl() {
      this.initData()
    },
    fileUrl() {
      this.initData()
    },
    filter() {
      this.refreshLogData()
    }
  },
  mounted() {
    this.bindPanelResize()
    this.$nextTick(() => this.updateHeight())
  },
  created() {
    this.data = []
    if (this.fileUrl || this.remoteUrl) {
      this.initData()
    }
  },
  beforeDestroy() {
    if (this.resizeObserver) {
      this.resizeObserver.disconnect()
      this.resizeObserver = null
    }
    window.removeEventListener('resize', this.updateHeight)
    this.closeSocket()
  },
  methods: {
    bindPanelResize() {
      window.addEventListener('resize', this.updateHeight)
      if (typeof ResizeObserver !== 'undefined' && this.$refs.panel) {
        this.resizeObserver = new ResizeObserver(() => this.updateHeight())
        this.resizeObserver.observe(this.$refs.panel)
      }
    },
    updateHeight() {
      const panel = this.$refs.panel
      if (!panel) return
      // 与黑框同高，避免虚拟列表高度大于容器导致右侧滚动条「悬空」
      const h = Math.floor(panel.clientHeight)
      if (h > 0 && h !== this.winHeight) {
        this.winHeight = h
      }
    },
    closeSocket() {
      if (this.websocket) {
        try {
          this.websocket.onopen = null
          this.websocket.onmessage = null
          this.websocket.onerror = null
          this.websocket.onclose = null
          this.websocket.close()
        } catch (e) { /* ignore */ }
        this.websocket = null
      }
    },
    refreshLogData() {
      this.logData = this.getLogData()
    },
    trimRealtimeBuffer() {
      if (!this.isRealtime) return
      if (this.data.length > REALTIME_KEEP_LINES) {
        this.data = this.data.slice(this.data.length - REALTIME_KEEP_LINES)
      }
    },
    appendLine(line) {
      if (line == null || line === '') {
        return
      }
      this.data.push(String(line))
      this.trimRealtimeBuffer()
      this.refreshLogData()
    },
    initData() {
      this.loading = true
      this.data = []
      this.logData = ''
      this.statusText = ''
      this.statusOk = false
      this.closeSocket()
      this.$nextTick(() => this.updateHeight())

      if (this.fileUrl) {
        request({
          method: 'get',
          url: this.fileUrl
        }).then((res) => {
          const text = typeof res === 'string' ? res : ''
          text.split('\n').forEach(item => {
            if (item !== '') {
              this.data.push(item)
            }
          })
          this.refreshLogData()
          this.loading = false
          if (this.loadEnd && typeof this.loadEnd === 'function') {
            this.loadEnd()
          }
        }).catch((error) => {
          console.log(error)
          this.statusText = '日志文件加载失败'
          this.statusOk = false
        })
        return
      }

      if (!this.remoteUrl) {
        return
      }

      const token = this.$store.getters.token
      let url = this.remoteUrl
      if (token) {
        url += (url.indexOf('?') >= 0 ? '&' : '?') + 'access-token=' + encodeURIComponent(token)
      }

      let ws
      try {
        ws = new WebSocket(url)
      } catch (e) {
        this.statusText = 'WebSocket 创建失败: ' + (e && e.message)
        this.statusOk = false
        return
      }
      this.websocket = ws
      this.statusText = '连接中…'
      this.statusOk = false

      ws.onopen = () => {
        this.statusText = '已连接'
        this.statusOk = true
        this.loading = false
      }
      ws.onmessage = (e) => {
        this.loading = false
        this.appendLine(e.data)
      }
      ws.onerror = () => {
        this.statusText = '连接异常（请确认后端已启动，开发环境需直连 :18080）'
        this.statusOk = false
      }
      ws.onclose = (e) => {
        this.statusOk = false
        this.statusText = this.statusText && this.statusText.indexOf('异常') >= 0
          ? this.statusText
          : (`连接已断开` + (e && e.code ? ` (${e.code})` : ''))
      }
    },
    getLogData() {
      if (this.data.length === 0) {
        return ''
      }
      let result = ''
      for (let i = 0; i < this.data.length; i++) {
        if (!this.filter || this.data[i].indexOf(this.filter) > -1) {
          result += this.data[i] + '\r\n'
        }
      }
      return result
    },
    getLogDataWithOutAnsi() {
      if (this.data.length === 0) {
        return ''
      }
      let result = ''
      for (let i = 0; i < this.data.length; i++) {
        if (!this.filter || this.data[i].indexOf(this.filter) > -1) {
          result += stripAnsi(this.data[i]) + '\r\n'
        }
      }
      return result
    },
    downloadFile() {
      const blob = new Blob([this.getLogDataWithOutAnsi()], {
        type: 'text/plain; charset=utf-8'
      })
      const reader = new FileReader()
      reader.readAsDataURL(blob)
      reader.onload = (e) => {
        const a = document.createElement('a')
        a.download = `zero-web-server-${this.filter || 'all'}-${moment().format('YYYY-MM-DD')}.log`
        a.href = e.target.result
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
      }
    },
    scrollToTop() {
      const viewer = this.$refs.logViewer
      if (viewer && typeof viewer.setScrollTop === 'function') {
        viewer.setScrollTop(0)
      }
    },
    scrollToBottom() {
      const viewer = this.$refs.logViewer
      if (viewer && typeof viewer.setScrollTop === 'function') {
        const count = (viewer.linesCount != null) ? viewer.linesCount : this.data.length
        viewer.setScrollTop(count)
      }
    }
  }
}
</script>

<style scoped>
.log-shell {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  height: 100%;
  overflow: hidden;
}

.log-panel {
  position: relative;
  flex: 1;
  min-height: 0;
  border-radius: 8px;
  overflow: hidden;
  background: #1e1e1e;
  border: 1px solid #1e293b;
}

.log-toolbar {
  position: absolute;
  top: 10px;
  /* 预留滚动条宽度，确保整块工具条都在黑框内 */
  right: 40px;
  z-index: 6;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 6px 4px 8px;
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.82);
  border: 1px solid rgba(148, 163, 184, 0.22);
  backdrop-filter: blur(6px);
  max-width: calc(100% - 52px);
  box-sizing: border-box;
}

.log-status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.log-status-dot.is-online {
  background: #22c55e;
  box-shadow: 0 0 0 2px rgba(34, 197, 94, 0.28);
}

.log-status-dot.is-offline {
  background: #eab308;
  box-shadow: 0 0 0 2px rgba(234, 179, 8, 0.28);
}

.log-filter-input {
  width: 160px;
  min-width: 0;
  flex: 1 1 auto;
}

.log-filter-input >>> .el-input__inner {
  background: rgba(255, 255, 255, 0.08);
  border-color: rgba(255, 255, 255, 0.16);
  color: #e2e8f0;
}

.log-filter-input >>> .el-input__inner::placeholder {
  color: #94a3b8;
}

.log-filter-input >>> .el-input__prefix,
.log-filter-input >>> .el-input__suffix {
  color: #94a3b8;
}

.log-scroll-btn {
  width: 28px;
  height: 28px;
  padding: 0;
  border: 1px solid rgba(148, 163, 184, 0.35);
  border-radius: 6px;
  background: rgba(30, 41, 59, 0.95);
  color: #e2e8f0;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

.log-scroll-btn:hover {
  background: #1565c0;
  border-color: #1565c0;
  color: #fff;
}

.log-viewer-panel {
  width: 100%;
  height: 100%;
  box-sizing: border-box;
}

/* 让虚拟列表滚动条落在黑框内并铺满高度 */
.log-panel >>> .log-viewer {
  height: 100% !important;
  max-height: 100%;
  box-sizing: border-box;
  padding: 12px 0;
  background: transparent;
}

.log-actions {
  display: flex;
  justify-content: flex-end;
  flex-shrink: 0;
  padding-top: 8px;
}
</style>
