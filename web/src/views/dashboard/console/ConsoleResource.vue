<template>
  <div class="console-card">
    <div class="console-card__head">
      <div class="console-card__title">
        <i class="el-icon-s-data console-card__icon is-resource" />
        <span>接入资源</span>
      </div>
      <div class="console-card__hint">在线 / 总数</div>
    </div>
    <div class="console-card__body console-resource">
      <div class="resource-item">
        <div class="resource-item__label"><i class="el-icon-video-camera" />设备</div>
        <el-progress
          type="circle"
          :width="88"
          :stroke-width="8"
          :percentage="pct(deviceInfo)"
          :color="pct(deviceInfo) > 0 ? '#1565c0' : '#c0c4cc'"
        />
        <div class="resource-item__meta">{{ deviceInfo.online }} / {{ deviceInfo.total }}</div>
      </div>
      <div class="resource-item">
        <div class="resource-item__label"><i class="el-icon-picture-outline" />通道</div>
        <el-progress
          type="circle"
          :width="88"
          :stroke-width="8"
          :percentage="pct(channelInfo)"
          :color="pct(channelInfo) > 0 ? '#1565c0' : '#c0c4cc'"
        />
        <div class="resource-item__meta">{{ channelInfo.online }} / {{ channelInfo.total }}</div>
      </div>
      <div class="resource-item">
        <div class="resource-item__label"><i class="el-icon-upload2" />推流</div>
        <el-progress
          type="circle"
          :width="88"
          :stroke-width="8"
          :percentage="pct(pushInfo)"
          :color="pct(pushInfo) > 0 ? '#00897b' : '#c0c4cc'"
        />
        <div class="resource-item__meta">{{ pushInfo.online }} / {{ pushInfo.total }}</div>
      </div>
      <div class="resource-item">
        <div class="resource-item__label"><i class="el-icon-download" />拉流代理</div>
        <el-progress
          type="circle"
          :width="88"
          :stroke-width="8"
          :percentage="pct(proxyInfo)"
          :color="pct(proxyInfo) > 0 ? '#00897b' : '#c0c4cc'"
        />
        <div class="resource-item__meta">{{ proxyInfo.online }} / {{ proxyInfo.total }}</div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'ConsoleResource',
  data() {
    return {
      deviceInfo: { total: 0, online: 0 },
      channelInfo: { total: 0, online: 0 },
      pushInfo: { total: 0, online: 0 },
      proxyInfo: { total: 0, online: 0 }
    }
  },
  methods: {
    pct(info) {
      if (!info || !info.total) return 0
      return Math.floor((info.online / info.total) * 100)
    },
    setData(data) {
      if (!data) return
      this.deviceInfo = data.device || this.deviceInfo
      this.channelInfo = data.channel || this.channelInfo
      this.pushInfo = data.push || this.pushInfo
      this.proxyInfo = data.proxy || this.proxyInfo
    }
  }
}
</script>

<style scoped>
.console-card {
  width: 100%;
  height: 100%;
  background: #fff;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}
.console-card__head {
  height: 44px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 14px;
  border-bottom: 1px solid #eef2f7;
}
.console-card__title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
}
.console-card__icon {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  background: #e8eaf6;
  color: #3949ab;
}
.console-card__hint {
  font-size: 12px;
  color: #94a3b8;
}
.console-card__body {
  flex: 1;
  min-height: 0;
}
.console-resource {
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-rows: 1fr 1fr;
  padding: 8px 4px 12px;
}
.resource-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
}
.resource-item__label {
  font-size: 12px;
  color: #64748b;
  display: flex;
  align-items: center;
  gap: 4px;
}
.resource-item__meta {
  font-size: 12px;
  color: #334155;
  font-weight: 600;
}
.console-resource ::v-deep .el-progress__text {
  font-size: 16px !important;
}
</style>
