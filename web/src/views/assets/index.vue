<template>
  <div class="app-container">
    <el-form :inline="true" size="mini">
      <el-form-item label="搜索">
        <el-input v-model="query" clearable placeholder="名称 / 内码 / 序列号" style="width:200px" @input="reload" />
      </el-form-item>
      <el-form-item label="协议">
        <el-select v-model="protocol" clearable placeholder="全部" style="width:130px" @change="reload">
          <el-option label="GB28181" value="gb28181" />
          <el-option label="ONVIF" value="onvif" />
          <el-option label="插件" value="plugin" />
        </el-select>
      </el-form-item>
      <el-form-item label="状态">
        <el-select v-model="status" clearable placeholder="全部" style="width:110px" @change="reload">
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button icon="el-icon-refresh" circle :loading="loading" @click="loadList" />
      </el-form-item>
    </el-form>

    <el-table v-loading="loading" :data="list" size="small" border height="calc(100vh - 220px)">
      <el-table-column prop="device.name" label="名称" min-width="150" />
      <el-table-column label="协议" width="100">
        <template slot-scope="{ row }">
          <el-tag v-if="row.binding" size="mini">{{ row.binding.protocol }}</el-tag>
          <span v-else class="empty">—</span>
        </template>
      </el-table-column>
      <el-table-column label="接入" width="90">
        <template slot-scope="{ row }">
          <el-tag v-if="row.binding" size="mini" type="info">{{ row.binding.mode }}</el-tag>
          <span v-else class="empty">—</span>
        </template>
      </el-table-column>
      <el-table-column label="地址 / 外部 ID" min-width="180" show-overflow-tooltip>
        <template slot-scope="{ row }">{{ row.binding ? (row.binding.endpoint || row.binding.externalId) : '—' }}</template>
      </el-table-column>
      <el-table-column prop="device.vendor" label="厂商" width="100" />
      <el-table-column prop="resourceCount" label="资源数" width="80" />
      <el-table-column label="状态" width="90">
        <template slot-scope="{ row }">
          <el-tag :type="row.device.online ? 'success' : 'info'" size="mini">{{ row.device.online ? '在线' : '离线' }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template slot-scope="{ row }">
          <el-button type="text" size="mini" @click="openDetail(row)">详情</el-button>
          <el-button type="text" size="mini" @click="sync(row)">同步资源</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      style="text-align:right;margin-top:8px"
      layout="total, sizes, prev, pager, next"
      :total="total"
      :page-size="count"
      :current-page.sync="page"
      :page-sizes="[15,25,35,50]"
      @current-change="loadList"
      @size-change="onSize"
    />

    <el-drawer :visible.sync="detailVisible" size="62%" title="设备资产详情" @close="stopLive">
      <div v-if="detail" class="detail">
        <el-descriptions :column="2" size="small" border>
          <el-descriptions-item label="名称">{{ detail.device.name }}</el-descriptions-item>
          <el-descriptions-item label="内码">{{ detail.device.internalCode }}</el-descriptions-item>
          <el-descriptions-item label="厂商/型号">{{ detail.device.vendor }} {{ detail.device.model }}</el-descriptions-item>
          <el-descriptions-item label="序列号">{{ detail.device.serialNumber || '—' }}</el-descriptions-item>
          <el-descriptions-item label="接入协议">{{ detail.binding ? detail.binding.protocol : '—' }}</el-descriptions-item>
          <el-descriptions-item label="接入地址">{{ detail.binding ? (detail.binding.endpoint || detail.binding.externalId) : '—' }}</el-descriptions-item>
        </el-descriptions>

        <div class="section-title">能力（设备级）</div>
        <div>
          <el-tag v-for="c in detail.capabilities" :key="c.id" size="mini" style="margin:0 4px 4px 0"
                  :type="c.available ? 'success' : 'info'">{{ c.operation }}</el-tag>
        </div>

        <div class="section-title">资源</div>
        <el-table :data="detail.resources" size="small" border>
          <el-table-column prop="name" label="名称" min-width="140" />
          <el-table-column prop="kind" label="类型" width="130" />
          <el-table-column prop="externalKey" label="外部键" min-width="120" show-overflow-tooltip />
          <el-table-column prop="status" label="状态" width="80" />
          <el-table-column label="码流" width="150">
            <template slot-scope="{ row }">
              <el-select v-model="row._profileId" size="mini" placeholder="默认" style="width:120px">
                <el-option v-for="p in row._profiles" :key="p.id" :label="p.name || p.role" :value="p.id" />
              </el-select>
            </template>
          </el-table-column>
          <el-table-column label="能力" min-width="160">
            <template slot-scope="{ row }">
              <el-tag v-for="c in row._caps" :key="c.id" size="mini" style="margin-right:3px">{{ c.operation }}</el-tag>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="90">
            <template slot-scope="{ row }">
              <el-button v-if="row.kind === 'video_sensor'" type="text" size="mini" @click="play(row)">播放</el-button>
              <span v-else class="empty">—</span>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </el-drawer>

    <el-dialog :title="playing ? ('播放 - ' + playing.name) : '播放'" :visible.sync="playerVisible" width="820px" @closed="stopLive">
      <jessibuca v-if="playerVisible && videoUrl" :video-url="videoUrl" height="440px" />
      <div v-if="streamInfo" class="stream-links">
        <div v-if="streamInfo.flv">FLV: {{ streamInfo.flv }}</div>
        <div v-if="streamInfo.wsFlv">WS-FLV: {{ streamInfo.wsFlv }}</div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import jessibuca from '@/views/common/jessibuca'
import { queryAssets, getAssetDetail, syncAsset, getResourceDetail, startResourceLive, stopResourceLive } from '@/api/asset'

export default {
  name: 'DeviceAssets',
  components: { jessibuca },
  data() {
    return {
      loading: false,
      list: [],
      total: 0,
      page: 1,
      count: 15,
      query: '',
      protocol: '',
      status: '',
      detailVisible: false,
      detail: null,
      playerVisible: false,
      playing: null,
      streamInfo: null,
      videoUrl: ''
    }
  },
  created() { this.loadList() },
  beforeDestroy() { this.stopLive() },
  methods: {
    reload() { this.page = 1; this.loadList() },
    onSize(n) { this.count = n; this.page = 1; this.loadList() },
    loadList() {
      this.loading = true
      queryAssets({ page: this.page, count: this.count, query: this.query, protocol: this.protocol, status: this.status })
        .then(res => {
          this.list = (res.data && res.data.list) || []
          this.total = (res.data && res.data.total) || 0
        }).finally(() => { this.loading = false })
    },
    openDetail(row) {
      getAssetDetail(row.device.id).then(res => {
        this.detail = res.data
        this.loadResourceMeta()
        this.detailVisible = true
      })
    },
    loadResourceMeta() {
      if (!this.detail || !this.detail.resources) return
      this.detail.resources.forEach(r => {
        this.$set(r, '_profiles', [])
        this.$set(r, '_profileId', '')
        this.$set(r, '_caps', [])
        getResourceDetail(r.id).then(res => {
          const d = res.data || {}
          this.$set(r, '_profiles', d.profiles || [])
          this.$set(r, '_caps', d.capabilities || [])
          const def = (d.profiles || []).find(p => p.isDefault)
          this.$set(r, '_profileId', def ? def.id : ((d.profiles || [])[0] || {}).id || '')
        })
      })
    },
    sync(row) {
      syncAsset(row.device.id).then(() => {
        this.$message.success('资源已同步')
        this.loadList()
        if (this.detail && this.detail.device.id === row.device.id) this.openDetail(row)
      })
    },
    play(row) {
      startResourceLive(row.id, row._profileId || '').then(res => {
        const content = res.data
        if (!content) { this.$message.error('未返回播放地址'); return }
        this.playing = row
        this.streamInfo = content
        this.videoUrl = content.wsFlv || content.flv || ''
        this.playerVisible = true
      })
    },
    stopLive() {
      if (this.playing) {
        stopResourceLive(this.playing.id).catch(() => {})
      }
      this.playerVisible = false
      this.videoUrl = ''
      this.streamInfo = null
      this.playing = null
    }
  }
}
</script>

<style scoped>
.detail { padding: 0 12px 12px; }
.section-title { margin: 16px 0 8px; font-weight: 600; color: #303133; }
.empty { color: #c0c4cc; }
.stream-links { margin-top: 10px; color: #606266; font-size: 12px; word-break: break-all; }
</style>
