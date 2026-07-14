<template>
  <div
    id="operationsForHistoryLog"
    v-loading="loading"
    class="app-container history-log-page"
  >
    <div class="history-log-shell">
      <div class="history-log-shell-head">
        <div class="history-log-shell-title">历史日志</div>
        <div class="history-log-shell-desc">每个运行周期一个日志文件</div>
        <el-button
          class="history-log-refresh"
          size="mini"
          icon="el-icon-refresh"
          circle
          :loading="loading"
          @click="getFileList"
        />
      </div>

      <div v-if="!loading && fileList.length === 0" class="history-log-empty">
        暂无日志文件
      </div>

      <div class="history-log-grid">
        <div
          v-for="file in fileList"
          :key="file.fileName"
          class="history-log-card"
        >
          <div class="history-log-card-head">
            <i class="el-icon-document history-log-card-icon" />
            <div class="history-log-card-name" :title="file.fileName">{{ file.fileName }}</div>
            <el-tag size="mini" type="info" effect="plain">{{ formatFileSize(file.fileSize) }}</el-tag>
          </div>

          <div class="history-log-card-body">
            <div class="history-log-field">
              <span class="history-log-label">开始时间</span>
              <span class="history-log-value">{{ formatTimeStamp(file.startTime) }}</span>
            </div>
            <div class="history-log-field">
              <span class="history-log-label">关机时间</span>
              <span class="history-log-value">{{ formatTimeStamp(file.endTime) }}</span>
            </div>
          </div>

          <div class="history-log-card-footer">
            <el-button
              type="text"
              icon="el-icon-view"
              @click="showLogView(file)"
            >查看</el-button>
            <el-button
              type="text"
              icon="el-icon-download"
              @click="downloadFile(file)"
            >下载</el-button>
          </div>
        </div>
      </div>
    </div>

    <el-dialog
      top="10vh"
      :title="playerTitle"
      :visible.sync="showLog"
      width="90%"
      append-to-body
    >
      <div class="history-log-viewer">
        <showLog ref="recordVideoPlayer" :file-url="fileUrl" :load-end="loadEnd" />
      </div>
    </el-dialog>
  </div>
</template>

<script>
import showLog from './showLog.vue'
import moment from 'moment'
import { getToken } from '@/utils/auth'

export default {
  name: 'OperationsHistoryLog',
  components: {
    showLog
  },
  data() {
    return {
      showLog: false,
      playerTitle: '',
      fileUrl: '',
      file: null,
      fileList: [],
      loading: false
    }
  },
  mounted() {
    this.getFileList()
  },
  destroyed() {
    this.$destroy('recordVideoPlayer')
  },
  methods: {
    getFileList() {
      this.loading = true
      this.$store.dispatch('log/queryList', {})
        .then((data) => {
          this.fileList = data || []
        })
        .catch((error) => {
          console.log(error)
        })
        .finally(() => {
          this.loading = false
        })
    },
    showLogView(file) {
      this.playerTitle = '正在加载日志...'
      this.fileUrl = `/api/log/file/${file.fileName}`
      this.showLog = true
      this.file = file
    },
    downloadFile(file) {
      const fileUrl = ((process.env.NODE_ENV === 'development') ? process.env.VUE_APP_BASE_API : window.baseUrl) + `/api/log/file/${file.fileName}`
      const headers = new Headers()
      headers.append('access-token', getToken())
      fetch(fileUrl, {
        method: 'GET',
        headers: headers
      })
        .then(response => response.blob())
        .then(blob => {
          const link = document.createElement('a')
          link.target = '_blank'
          link.href = window.URL.createObjectURL(blob)
          link.download = file.fileName
          document.body.appendChild(link)
          link.click()
          document.body.removeChild(link)
          this.$message.success('下载成功')
        })
        .catch(error => {
          console.error('下载失败：', error)
          this.$message.error('下载失败')
        })
    },
    loadEnd() {
      if (this.file) {
        this.playerTitle = this.file.fileName
      }
    },
    formatTimeStamp(time) {
      if (!time) return '—'
      return moment.unix(time / 1000).format('YYYY-MM-DD HH:mm:ss')
    },
    formatFileSize(fileSize) {
      if (fileSize == null || fileSize < 0) return '—'
      if (fileSize < 1024) {
        return fileSize + ' B'
      }
      if (fileSize < (1024 * 1024)) {
        return (fileSize / 1024).toFixed(2) + ' KB'
      }
      if (fileSize < (1024 * 1024 * 1024)) {
        return (fileSize / (1024 * 1024)).toFixed(2) + ' MB'
      }
      return (fileSize / (1024 * 1024 * 1024)).toFixed(2) + ' GB'
    }
  }
}
</script>

<style scoped>
.history-log-page {
  padding: 16px 20px 24px;
  min-height: calc(100vh - 124px);
  background: #f0f4f8;
}

.history-log-shell {
  max-width: 1180px;
  margin: 0 auto;
}

.history-log-shell-head {
  position: relative;
  margin-bottom: 14px;
  padding: 16px 18px;
  background: #fff;
  border: 1px solid #e3ebf5;
  border-radius: 10px;
  box-shadow: 0 8px 24px rgba(15, 40, 80, 0.05);
}

.history-log-shell-title {
  font-size: 18px;
  font-weight: 650;
  color: #1e293b;
  line-height: 1.3;
}

.history-log-shell-desc {
  margin-top: 4px;
  font-size: 13px;
  color: #64748b;
}

.history-log-refresh {
  position: absolute;
  top: 14px;
  right: 16px;
}

.history-log-empty {
  padding: 48px 16px;
  text-align: center;
  color: #94a3b8;
  background: #fff;
  border: 1px dashed #dbe4ef;
  border-radius: 10px;
}

.history-log-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 14px;
}

.history-log-card {
  display: flex;
  flex-direction: column;
  background: #fff;
  border: 1px solid #e3ebf5;
  border-radius: 10px;
  box-shadow: 0 6px 18px rgba(15, 40, 80, 0.04);
  overflow: hidden;
}

.history-log-card-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 14px 16px 10px;
  border-bottom: 1px solid #eef2f7;
}

.history-log-card-icon {
  color: #1565c0;
  font-size: 18px;
  flex-shrink: 0;
}

.history-log-card-name {
  flex: 1;
  min-width: 0;
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.history-log-card-body {
  display: grid;
  gap: 10px;
  padding: 14px 16px;
  flex: 1;
}

.history-log-field {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.history-log-label {
  font-size: 12px;
  color: #94a3b8;
}

.history-log-value {
  font-size: 13px;
  font-weight: 550;
  color: #1e293b;
}

.history-log-card-footer {
  display: flex;
  justify-content: flex-end;
  gap: 4px;
  padding: 8px 12px 10px;
  border-top: 1px solid #f1f5f9;
  background: #fafbfc;
}

.history-log-viewer {
  height: 600px;
}
</style>
