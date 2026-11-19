<template>
  <div class="unified-channel">
    <el-form :inline="true" size="mini">
      <el-form-item v-if="embedded" style="margin-right: 2rem">
        <el-page-header content="通道列表" @back="$emit('back')" />
      </el-form-item>
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
      <el-form-item v-if="!lockedProtocol" label="协议">
        <el-select v-model="protocolFilter" style="width: 9rem" clearable placeholder="全部" @change="search">
          <el-option label="全部" value="" />
          <el-option label="GB28181" value="gb28181" />
          <el-option label="ONVIF" value="onvif" />
          <el-option label="插件" value="plugin" />
        </el-select>
      </el-form-item>
      <el-form-item v-if="deviceInternalCode && !deviceLocked">
        <el-tag size="mini" closable @close="clearDevice">设备：{{ deviceInternalCode }}</el-tag>
      </el-form-item>
      <el-form-item>
        <el-button size="mini" icon="el-icon-refresh-right" @click="load">刷新</el-button>
      </el-form-item>
    </el-form>

    <el-table v-loading="loading" :data="list" size="small" border>
      <el-table-column prop="name" label="通道名称" min-width="160" />
      <el-table-column prop="internalCode" label="通道内码" width="200" />
      <el-table-column label="类型" width="110">
        <template slot-scope="{ row }">{{ kindLabel(row.kind) }}</template>
      </el-table-column>
      <el-table-column label="经纬度" min-width="150">
        <template slot-scope="{ row }">
          <span v-if="row.longitude || row.latitude">{{ row.longitude }}, {{ row.latitude }}</span>
          <span v-else class="sub-text">—</span>
        </template>
      </el-table-column>
      <el-table-column label="所属设备" min-width="180">
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

    <!-- 与标准接入一致的播放器（编码信息/实时视频；PTZ 暂不对插件开放） -->
    <device-player ref="devicePlayer" @closed="onPlayerClosed" />
  </div>
</template>

<script>
import DevicePlayer from '@/views/device/dialog/devicePlayer.vue'
import { queryChannels, startResourceLive, stopResourceLive } from '@/api/asset'

export default {
  name: 'UnifiedChannelPanel',
  components: { DevicePlayer },
  props: {
    // 固定设备内码（设备详情下钻时传入）
    deviceInternalCode: { type: String, default: '' },
    // 限定协议（gb28181 | onvif | plugin）
    protocol: { type: String, default: '' },
    // 嵌入到接入页时显示「返回」页头
    embedded: { type: Boolean, default: false },
    // 设备内码是否锁定（不允许在页面内清除）
    deviceLocked: { type: Boolean, default: false }
  },
  data() {
    return {
      loading: false,
      list: [],
      total: 0,
      page: 1,
      count: 15,
      query: '',
      protocolFilter: this.protocol,
      deviceCode: this.deviceInternalCode,
      playingRow: null
    }
  },
  computed: {
    lockedProtocol() {
      return !!this.protocol
    }
  },
  watch: {
    deviceInternalCode(v) {
      this.deviceCode = v
      this.page = 1
      this.load()
    },
    protocol(v) {
      this.protocolFilter = v
      this.page = 1
      this.load()
    }
  },
  mounted() {
    this.load()
  },
  methods: {
    protocolLabel(p) {
      return { gb28181: 'GB28181', onvif: 'ONVIF', plugin: '插件' }[p] || (p || '-')
    },
    kindLabel(kind) {
      return {
        video_sensor: '视频通道',
        audio_input: '音频输入',
        audio_output: '音频输出',
        alarm_input: '报警输入',
        relay_output: '继电器',
        metadata_source: '数据通道'
      }[kind] || kind || '-'
    },
    load() {
      this.loading = true
      queryChannels({
        page: this.page,
        count: this.count,
        query: this.query,
        protocol: this.protocolFilter,
        deviceInternalCode: this.deviceCode
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
      this.deviceCode = ''
      this.search()
    },
    openDevice(row) {
      this.$router.push({ path: '/assets', query: { query: row.deviceInternalCode } })
    },
    play(row) {
      const player = this.$refs.devicePlayer
      if (!player) return
      // 先打开播放器（loading 态），取流成功后再把流内容交给播放器。
      player.openDialog('media', '', '', { pending: true, hasAudio: !!row.hasAudio, ptzType: 0, showPtz: false })
      row._loading = true
      this.playingRow = row
      startResourceLive(row.resourceId).then(res => {
        const content = (res && res.data) || null
        if (!content) {
          player.onStreamError('未返回播放地址')
          return
        }
        player.onStreamReady(content, !!row.hasAudio)
      }).catch(err => {
        player.onStreamError((err && err.message) || '取流失败')
      }).finally(() => { row._loading = false })
    },
    // 播放器关闭（点 X 或按关闭）后停流，避免后台一直拉流。
    onPlayerClosed() {
      if (this.playingRow) {
        stopResourceLive(this.playingRow.resourceId).catch(() => {})
        this.playingRow = null
      }
    }
  }
}
</script>

<style scoped>
.unified-channel .sub-text { color: #909399; font-size: 12px; }
</style>
