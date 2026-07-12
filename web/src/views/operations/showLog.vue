<template>
  <div id="log" style="height: 100%">
    <el-form :inline="true" size="mini">
      <el-form-item label="过滤">
        <el-input v-model="filter" size="mini" placeholder="请输入过滤关键字" style="width: 20vw" />
      </el-form-item>
      <el-form-item v-if="statusText">
        <span :style="{ color: statusOk ? '#67c23a' : '#f56c6c', fontSize: '12px' }">{{ statusText }}</span>
      </el-form-item>
      <el-form-item style="float: right;">
        <el-button size="mini" icon="el-icon-download" @click="downloadFile()">下载</el-button>
      </el-form-item>
    </el-form>
    <log-viewer :log="logData" :auto-scroll="true" :height="winHeight" style="height: calc(100% - 60px);" />
  </div>
</template>

<script>

import moment from 'moment/moment'
import logViewer from '@femessage/log-viewer'
import stripAnsi from 'strip-ansi'
import request from '@/utils/request'

export default {
  name: 'Log',
  components: { logViewer },
  props: ['fileUrl', 'remoteUrl', 'loadEnd'],
  data() {
    return {
      loading: true,
      winHeight: window.innerHeight - 200,
      data: [],
      filter: '',
      logData: '',
      websocket: null,
      statusText: '',
      statusOk: false
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
  created() {
    this.data = []
    if (this.fileUrl || this.remoteUrl) {
      this.initData()
    }
  },
  beforeDestroy() {
    this.closeSocket()
  },
  methods: {
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
    appendLine(line) {
      if (line == null || line === '') {
        return
      }
      this.data.push(String(line))
      this.refreshLogData()
    },
    initData() {
      this.loading = true
      this.data = []
      this.logData = ''
      this.statusText = ''
      this.closeSocket()

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

      // 不用 JWT 当 Sec-WebSocket-Protocol（易握手失败）；改 query 传 token
      const token = this.$store.getters.token
      let url = this.remoteUrl
      if (token) {
        url += (url.indexOf('?') >= 0 ? '&' : '?') + 'access-token=' + encodeURIComponent(token)
      }
      console.log('realtime log ws:', url)

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
        if (!this.statusOk) {
          this.statusText = `连接关闭 code=${e.code}`
        } else {
          this.statusText = '连接已断开'
          this.statusOk = false
        }
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
        a.download = `zero-web-kit-${this.filter || 'all'}-${moment().format('YYYY-MM-DD')}.log`
        a.href = e.target.result
        document.body.appendChild(a)
        a.click()
        document.body.removeChild(a)
      }
    }
  }
}
</script>
