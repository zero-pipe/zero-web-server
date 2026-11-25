<template>
  <div class="app-container asset-page">
    <!-- 过滤栏 -->
    <el-form :inline="true" size="mini" class="asset-filter">
      <el-form-item>
        <el-input v-model="query" clearable placeholder="设备名称 / 厂家 / 型号 / 通道" prefix-icon="el-icon-search" style="width:260px" />
      </el-form-item>
      <el-form-item label="状态">
        <el-select v-model="status" clearable placeholder="全部" style="width:110px">
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <span class="summary">共 <b>{{ filteredDevices.length }}</b> 台设备 · 在线 <b class="on">{{ onlineCount }}</b></span>
      </el-form-item>
      <el-form-item style="float:right">
        <el-button icon="el-icon-refresh" circle :loading="loading" @click="loadData" />
      </el-form-item>
    </el-form>

    <!-- 设备卡片网格（纯展示） -->
    <div v-loading="loading" class="card-grid">
      <div v-for="d in filteredDevices" :key="d.id" class="dev-card">
        <div class="dc-head">
          <img class="dc-icon" :src="iconSrc(d.deviceType, d.online)" alt="">
          <div class="dc-title">
            <div class="dc-name" :title="d.name">{{ d.name }}</div>
            <div class="dc-sub" :title="vendorModel(d)">{{ vendorModel(d) }}</div>
          </div>
          <span class="dc-status" :class="d.online ? 'on' : 'off'">{{ d.online ? '在线' : '离线' }}</span>
        </div>

        <div class="dc-divider" />

        <div class="dc-row">
          <span class="dc-label">通道数</span>
          <span class="dc-value">{{ d.channels.length }} 路</span>
        </div>
        <div class="dc-chans">
          <template v-if="d.channels.length">
            <div v-for="(ch, i) in d.channels" :key="ch.resourceId || i" class="dc-chan">
              <span class="dc-chan-idx">{{ i + 1 }}</span>
              <span class="dc-chan-name" :title="ch.name">{{ ch.name }}</span>
              <el-tag size="mini" effect="plain" :type="ch.channelType ? '' : 'info'">{{ typeLabel(ch.channelType) }}</el-tag>
            </div>
          </template>
          <div v-else class="dc-empty">暂无通道</div>
        </div>
      </div>

      <el-empty v-if="!loading && filteredDevices.length === 0" description="没有匹配的设备" style="grid-column:1/-1" />
    </div>
  </div>
</template>

<script>
import { queryChannels, queryAssets } from '@/api/asset'
import { listDictItems } from '@/api/dict'

// 设备类型 → 图标底图：枪机/遥控枪机→camera3，球机→camera2，半球/遥控半球→camera1，其余→camera。
const ICON_BASE = {
  bullet: 'camera3', remote_ptz: 'camera3',
  ptz: 'camera2',
  dome: 'camera1', remote_dome: 'camera1'
}

export default {
  name: 'DeviceAssets',
  data() {
    return {
      loading: false,
      query: '',
      status: '',
      devices: [],
      channelTypeMap: {}
    }
  },
  computed: {
    onlineCount() {
      return this.filteredDevices.filter(d => d.online).length
    },
    filteredDevices() {
      const q = this.query.trim().toLowerCase()
      return this.devices.filter(d => {
        if (this.status === 'online' && !d.online) return false
        if (this.status === 'offline' && d.online) return false
        if (!q) return true
        const hay = [d.name, d.vendor, d.model, ...d.channels.map(c => c.name)].join(' ').toLowerCase()
        return hay.includes(q)
      })
    }
  },
  created() {
    this.loadChannelTypeDict()
    this.loadData()
  },
  methods: {
    loadChannelTypeDict() {
      listDictItems('channel_type', true).then(res => {
        const map = {}
        ;((res.data && res.data.list) || []).forEach(it => { map[it.value] = it.label })
        this.channelTypeMap = map
      }).catch(() => {})
    },
    typeLabel(v) {
      if (!v) return '未设'
      return this.channelTypeMap[v] || v
    },
    formIconBase(deviceType) {
      return ICON_BASE[(deviceType || '').trim()] || 'camera'
    },
    iconSrc(deviceType, online) {
      return `static/images/gis/${this.formIconBase(deviceType)}${online ? '' : '-offline'}.png`
    },
    vendorModel(d) {
      const parts = [d.vendor, d.model].filter(x => x && String(x).trim())
      return parts.length ? parts.join(' · ') : '—'
    },
    loadData() {
      this.loading = true
      Promise.all([
        queryAssets({ page: 1, count: 1000 }),
        queryChannels({ page: 1, count: 5000 })
      ]).then(([assetRes, chRes]) => {
        const assets = (assetRes && assetRes.data && assetRes.data.list) || []
        const channels = (chRes && chRes.data && chRes.data.list) || []
        // 通道按设备分组（channels.deviceId == 设备 id）
        const byDevice = {}
        channels.forEach(c => {
          const k = String(c.deviceId || '')
          if (!k) return
          ;(byDevice[k] = byDevice[k] || []).push(c)
        })
        this.devices = assets
          .filter(a => a && a.device)
          .map(a => {
            const dev = a.device
            return {
              id: String(dev.id),
              name: dev.name || '未命名设备',
              deviceType: dev.deviceType || '',
              vendor: dev.vendor || '',
              model: dev.model || '',
              online: !!dev.online,
              channels: byDevice[String(dev.id)] || []
            }
          })
      }).catch(err => {
        this.$message.error((err && err.message) || '加载失败')
      }).finally(() => { this.loading = false })
    }
  }
}
</script>

<style scoped>
.asset-page { display: flex; flex-direction: column; }
.asset-filter { margin-bottom: 6px; }
.summary { color: #909399; font-size: 13px; }
.summary b { color: #303133; }
.summary b.on { color: #34a853; }

.card-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: 14px;
  padding-bottom: 12px;
}

.dev-card {
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 10px;
  padding: 14px 16px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, .05);
  transition: box-shadow .15s, border-color .15s;
}
.dev-card:hover { box-shadow: 0 4px 14px rgba(0, 0, 0, .1); border-color: #d9e2ec; }

.dc-head { display: flex; align-items: center; gap: 12px; }
.dc-icon { width: 44px; height: 44px; flex: 0 0 44px; }
.dc-title { flex: 1; min-width: 0; }
.dc-name {
  font-size: 15px; font-weight: 600; color: #303133;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.dc-sub {
  font-size: 12px; color: #909399; margin-top: 3px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.dc-status { font-size: 12px; padding: 1px 8px; border-radius: 10px; flex: 0 0 auto; }
.dc-status.on { color: #34a853; background: #e7f6ec; }
.dc-status.off { color: #909399; background: #f0f2f5; }

.dc-divider { height: 1px; background: #f0f2f5; margin: 12px 0; }

.dc-row { display: flex; align-items: center; gap: 8px; margin-bottom: 8px; }
.dc-label { color: #909399; font-size: 13px; flex: 0 0 48px; }
.dc-value { color: #303133; font-size: 13px; font-weight: 600; }

.dc-chans { display: flex; flex-direction: column; gap: 6px; }
.dc-chan { display: flex; align-items: center; gap: 8px; font-size: 13px; }
.dc-chan-idx {
  flex: 0 0 18px; height: 18px; line-height: 18px; text-align: center;
  background: #eef2f7; color: #606266; border-radius: 50%; font-size: 11px;
}
.dc-chan-name {
  flex: 1; min-width: 0; color: #303133;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.dc-empty { color: #c0c4cc; font-size: 13px; }
</style>
