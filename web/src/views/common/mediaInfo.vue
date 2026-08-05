<template>
  <div id="mediaInfo" class="media-info-panel">
    <div class="mi-toolbar">
      <span class="mi-title">流媒体概况</span>
      <el-button class="mi-refresh" icon="el-icon-refresh-right" circle size="mini" @click="getMediaInfo" />
    </div>

    <template v-if="hasInfo">
      <div class="mi-stats">
        <div class="mi-stat">
          <span class="mi-stat-value">{{ info.readerCount != null ? info.readerCount : '—' }}</span>
          <span class="mi-stat-label">观看人数</span>
        </div>
        <div class="mi-stat">
          <span class="mi-stat-value">{{ formatByteSpeed() }}</span>
          <span class="mi-stat-label">网络码率</span>
        </div>
        <div class="mi-stat">
          <span class="mi-stat-value">{{ formatAliveSecond() }}</span>
          <span class="mi-stat-label">持续时间</span>
        </div>
      </div>

      <div v-if="info.videoCodec" class="mi-section">
        <div class="mi-section-title">视频</div>
        <div class="mi-kv">
          <div class="mi-kv-item">
            <span class="mi-k">编码</span>
            <span class="mi-v">{{ info.videoCodec }}</span>
          </div>
          <div class="mi-kv-item">
            <span class="mi-k">分辨率</span>
            <span class="mi-v">{{ formatResolution() }}</span>
          </div>
          <div class="mi-kv-item">
            <span class="mi-k">帧率</span>
            <span class="mi-v">{{ info.fps != null ? info.fps + ' FPS' : '—' }}</span>
          </div>
          <div class="mi-kv-item">
            <span class="mi-k">丢包率</span>
            <span class="mi-v">{{ info.loss != null && info.loss !== '' ? info.loss : '—' }}</span>
          </div>
        </div>
      </div>

      <div v-if="info.audioCodec" class="mi-section">
        <div class="mi-section-title">音频</div>
        <div class="mi-kv">
          <div class="mi-kv-item">
            <span class="mi-k">编码</span>
            <span class="mi-v">{{ info.audioCodec }}</span>
          </div>
          <div class="mi-kv-item">
            <span class="mi-k">采样率</span>
            <span class="mi-v">{{ formatSampleRate() }}</span>
          </div>
          <div v-if="info.channels" class="mi-kv-item">
            <span class="mi-k">声道</span>
            <span class="mi-v">{{ info.channels }}</span>
          </div>
        </div>
      </div>
    </template>

    <div v-else class="mi-empty">等待流媒体信息…</div>
  </div>
</template>

<script>
export default {
  name: 'MediaInfo',
  props: ['app', 'stream', 'mediaServerId'],
  data() {
    return {
      info: {},
      task: null
    }
  },
  computed: {
    hasInfo() {
      return this.info && (this.info.videoCodec || this.info.audioCodec || this.info.readerCount != null || this.info.bytesSpeed != null)
    }
  },
  created() {
    this.getMediaInfo()
  },
  methods: {
    getMediaInfo() {
      if (!this.app || !this.stream || !this.mediaServerId) {
        return
      }
      this.$store.dispatch('server/getMediaInfo', {
        app: this.app,
        stream: this.stream,
        mediaServerId: this.mediaServerId
      })
        .then(data => {
          this.info = data || {}
        })
        .catch(() => {})
    },
    startTask() {
      this.stopTask()
      this.getMediaInfo()
      this.task = setInterval(this.getMediaInfo, 1000)
    },
    stopTask() {
      if (this.task) {
        window.clearInterval(this.task)
        this.task = null
      }
    },
    formatByteSpeed() {
      const bytesSpeed = this.info.bytesSpeed
      if (bytesSpeed == null || bytesSpeed === '') return '—'
      const num = 1024.0
      if (bytesSpeed < num) return bytesSpeed + ' B/s'
      if (bytesSpeed < Math.pow(num, 2)) return (bytesSpeed / num).toFixed(1) + ' KB/s'
      if (bytesSpeed < Math.pow(num, 3)) return (bytesSpeed / Math.pow(num, 2)).toFixed(2) + ' MB/s'
      return (bytesSpeed / Math.pow(num, 3)).toFixed(2) + ' GB/s'
    },
    formatAliveSecond() {
      const aliveSecond = Number(this.info.aliveSecond)
      if (!aliveSecond && aliveSecond !== 0) return '—'
      const h = Math.floor(aliveSecond / 3600)
      const minute = Math.floor((aliveSecond / 60) % 60)
      const second = Math.floor(aliveSecond % 60)
      if (h > 0) {
        return `${h}:${String(minute).padStart(2, '0')}:${String(second).padStart(2, '0')}`
      }
      return `${String(minute).padStart(2, '0')}:${String(second).padStart(2, '0')}`
    },
    formatSampleRate() {
      const rate = this.info.audioSampleRate || this.info.sampleRate
      if (!rate) return '—'
      return rate >= 1000 ? `${rate} Hz` : String(rate)
    },
    formatResolution() {
      const w = Number(this.info.width) || 0
      const h = Number(this.info.height) || 0
      if (w <= 0 || h <= 0) return '—'
      return `${w}×${h}`
    }
  }
}
</script>

<style scoped>
.media-info-panel {
  position: relative;
}
.mi-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 12px;
}
.mi-title {
  font-size: 13px;
  font-weight: 600;
  color: #303133;
}
.mi-stats {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 8px;
  margin-bottom: 14px;
}
.mi-stat {
  background: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 10px 8px;
  text-align: center;
}
.mi-stat-value {
  display: block;
  font-size: 15px;
  font-weight: 600;
  color: #303133;
  font-variant-numeric: tabular-nums;
}
.mi-stat-label {
  display: block;
  margin-top: 4px;
  font-size: 11px;
  color: #909399;
}
.mi-section {
  background: #f5f7fa;
  border: 1px solid #e4e7ed;
  border-radius: 6px;
  padding: 10px 12px;
  margin-bottom: 10px;
}
.mi-section-title {
  font-size: 11px;
  font-weight: 600;
  color: #909399;
  letter-spacing: 0.06em;
  margin-bottom: 8px;
}
.mi-kv {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 6px 12px;
}
.mi-kv-item {
  display: flex;
  justify-content: space-between;
  gap: 8px;
  font-size: 12px;
  line-height: 1.6;
}
.mi-k { color: #909399; }
.mi-v { color: #303133; font-weight: 500; }
.mi-empty {
  color: #909399;
  font-size: 12px;
  text-align: center;
  padding: 28px 8px;
}
</style>
