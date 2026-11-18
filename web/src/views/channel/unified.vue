<template>
  <div class="app-container unified-channel">
    <el-card shadow="never">
      <div slot="header" class="card-header">
        <span>通道列表（统一内码视图）</span>
        <div style="float:right">
          <el-button size="mini" icon="el-icon-refresh-right" @click="load">刷新</el-button>
        </div>
      </div>

      <el-form :inline="true" size="mini">
        <el-form-item label="搜索">
          <el-input
            v-model="query"
            clearable
            style="width: 15rem"
            placeholder="名称 / 设备内码 / 通道内码 / 通道标识"
            prefix-icon="el-icon-search"
            @keyup.enter.native="search"
            @clear="search"
          />
        </el-form-item>
        <el-form-item label="协议">
          <el-select v-model="protocol" style="width: 9rem" clearable placeholder="全部" @change="search">
            <el-option label="全部" value="" />
            <el-option label="GB28181" value="gb28181" />
            <el-option label="ONVIF" value="onvif" />
            <el-option label="插件" value="plugin" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="deviceInternalCode">
          <el-tag size="mini" closable @close="clearDevice">设备：{{ deviceInternalCode }}</el-tag>
        </el-form-item>
      </el-form>

      <el-table v-loading="loading" :data="list" size="small" border>
        <el-table-column prop="name" label="通道名称" min-width="160" />
        <el-table-column prop="internalCode" label="通道内码" width="200" />
        <el-table-column label="所属设备" min-width="170">
          <template slot-scope="{ row }">
            <div>{{ row.deviceName }}</div>
            <div class="sub-text">{{ row.deviceInternalCode }}</div>
          </template>
        </el-table-column>
        <el-table-column label="协议" width="100">
          <template slot-scope="{ row }">
            <el-tag size="mini" effect="plain">{{ protocolLabel(row.protocol) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="通道标识" min-width="170">
          <template slot-scope="{ row }">
            <span v-if="row.gbCode" title="国标通道编码">{{ row.gbCode }}</span>
            <span v-else>{{ row.externalKey }}</span>
          </template>
        </el-table-column>
        <el-table-column label="在线" width="80">
          <template slot-scope="{ row }">
            <el-tag size="mini" :type="row.online ? 'success' : 'info'">{{ row.online ? '在线' : '离线' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="140" fixed="right">
          <template slot-scope="{ row }">
            <el-button
              type="text"
              size="mini"
              :disabled="!row.online"
              :loading="row._loading"
              @click="play(row)"
            >播放</el-button>
            <el-button type="text" size="mini" @click="openDevice(row)">设备</el-button>
          </template>
        </el-table-column>
      </el-table>

      <el-pagination
        style="margin-top: 12px; text-align: right"
        background
        :current-page.sync="page"
        :page-size="count"
        :total="total"
        layout="total, prev, pager, next"
        @current-change="load"
      />
    </el-card>

    <el-dialog
      :title="'播放 - ' + (playingRow ? playingRow.name : '')"
      :visible.sync="playerVisible"
      width="820px"
      @closed="stopPlay"
    >
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
import { queryChannels, startResourceLive, stopResourceLive } from '@/api/asset'

export default {
  name: 'UnifiedChannelList',
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
      deviceInternalCode: '',
      playerVisible: false,
      playingRow: null,
      streamInfo: null,
      videoUrl: ''
    }
  },
  mounted() {
    // 支持 /channel/unified?deviceInternalCode=...&protocol=plugin 直达某设备/某协议
    const q = this.$route && this.$route.query ? this.$route.query : {}
    this.deviceInternalCode = q.deviceInternalCode || ''
    this.protocol = q.protocol || ''
    this.query = q.query || ''
    this.load()
  },
  methods: {
    protocolLabel(p) {
      return { gb28181: 'GB28181', onvif: 'ONVIF', plugin: '插件' }[p] || (p || '-')
    },
    load() {
      this.loading = true
      queryChannels({
        page: this.page,
        count: this.count,
        query: this.query,
        protocol: this.protocol,
        deviceInternalCode: this.deviceInternalCode
      }).then(res => {
        const data = (res && res.data) || {}
        this.list = (data.list || []).map(item => ({ ...item, _loading: false }))
        this.total = data.total || 0
      }).finally(() => { this.loading = false })
    },
    search() {
      this.page = 1
      this.load()
    },
    clearDevice() {
      this.deviceInternalCode = ''
      this.search()
    },
    openDevice(row) {
      this.$router.push({ path: '/assets', query: { query: row.deviceInternalCode } })
    },
    play(row) {
      row._loading = true
      startResourceLive(row.resourceId).then(res => {
        const content = (res && res.data) || null
        if (!content) {
          this.$message.error('未返回播放地址')
          return
        }
        this.playingRow = row
        this.streamInfo = content
        this.videoUrl = content.wsFlv || content.flv || ''
        this.playerVisible = true
      }).catch(err => {
        this.$message.error((err && err.message) || '播放失败')
      }).finally(() => { row._loading = false })
    },
    stopPlay() {
      if (this.playingRow) {
        stopResourceLive(this.playingRow.resourceId).catch(() => {})
      }
      this.playerVisible = false
      this.videoUrl = ''
      this.streamInfo = null
      this.playingRow = null
    }
  }
}
</script>

<style scoped>
.unified-channel .card-header { display: flex; justify-content: space-between; align-items: center; }
.unified-channel .sub-text { color: #909399; font-size: 12px; }
.unified-channel .stream-links { margin-top: 8px; font-size: 12px; color: #606266; word-break: break-all; }
</style>
