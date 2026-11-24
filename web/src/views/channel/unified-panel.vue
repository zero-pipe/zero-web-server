<template>
  <div class="unified-channel">
    <!-- 编辑与国标通道完全复用整页表单：国标码默认为空、需手工填写 -->
    <channel-edit
      v-if="editCtx"
      :internal-code="editCtx.internalCode"
      :protocol="editCtx.protocol"
      :close-edit="closeEdit"
    />
    <template v-else>
    <el-form :inline="true" size="mini">
      <el-form-item v-if="embedded" style="margin-right: 2rem">
        <el-page-header content="通道列表" @back="$emit('back')" />
      </el-form-item>
      <el-form-item>
        <el-input
          v-model="query"
          clearable
          style="width: 15rem"
          placeholder="通道名称 / 设备名称"
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
        <el-tag size="mini" closable @close="clearDevice">设备：{{ deviceName || '已选择' }}</el-tag>
      </el-form-item>
      <el-form-item>
        <el-button size="mini" icon="el-icon-refresh-right" circle title="刷新" aria-label="刷新" @click="load" />
      </el-form-item>
    </el-form>

    <el-table v-loading="loading" :data="list" size="small" style="width: 100%; font-size: 12px;" header-row-class-name="table-header">
      <el-table-column prop="_ordinal" label="通道号" width="90" align="center" />
      <el-table-column prop="name" label="名称" min-width="180" />
      <el-table-column v-if="showDeviceColumn" label="所属设备" min-width="180">
        <template slot-scope="{ row }">
          <div class="device-link" @click="openDevice(row)">{{ row.deviceName }}</div>
          <el-tag size="mini" effect="plain">{{ protocolLabel(row.protocol) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="能力集" min-width="260">
        <template slot-scope="{ row }">
          <span
            v-for="cap in capTemplate"
            :key="cap.op"
            class="cap-item"
            :class="{ 'cap-on': (row.caps || []).includes(cap.op) }"
            :title="(row.caps || []).includes(cap.op) ? ('支持' + cap.label) : ('不支持' + cap.label)"
          >
            <i :class="cap.icon" /> {{ cap.label }}
          </span>
        </template>
      </el-table-column>
      <el-table-column label="操作" min-width="180" fixed="right">
        <template slot-scope="{ row }">
          <el-button
            type="text"
            size="medium"
            icon="el-icon-video-play"
            :disabled="!row.online"
            :loading="row._loading"
            @click="play(row)"
          >播放</el-button>
          <el-divider direction="vertical" />
          <el-button
            type="text"
            size="medium"
            icon="el-icon-edit"
            @click="openEdit(row)"
          >编辑</el-button>
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
    </template>
  </div>
</template>

<script>
import DevicePlayer from '@/views/device/dialog/devicePlayer.vue'
import ChannelEdit from '@/views/device/channel/edit.vue'
import { queryChannels, startResourceLive, stopResourceLive, getResourceDetail } from '@/api/asset'

// 能力集展示模板：决定「展示哪些能力图标」，顺序即展示顺序。
// 点亮与否由后端每行的 caps（驱动真实实现 ∩ 通道事实）决定——扩展一个能力，
// 这里加一行即可；是否真能用，取决于对应驱动 Capabilities() 里有没有声明。
const CAP_TEMPLATE = [
  { op: 'play', icon: 'el-icon-video-play', label: '预览' },
  { op: 'playback', icon: 'el-icon-video-camera-solid', label: '回放' },
  { op: 'record', icon: 'el-icon-folder-opened', label: '录像检索' },
  { op: 'ptz', icon: 'el-icon-aim', label: '云台' },
  { op: 'snapshot', icon: 'el-icon-camera', label: '抓图' },
  { op: 'event', icon: 'el-icon-bell', label: '事件订阅' }
]

export default {
  name: 'UnifiedChannelPanel',
  components: { DevicePlayer, ChannelEdit },
  props: {
    // 固定设备内码（设备详情下钻时传入）
    deviceInternalCode: { type: String, default: '' },
    // 设备名称（展示用）
    deviceName: { type: String, default: '' },
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
      editCtx: null,
      playingRow: null,
      capTemplate: CAP_TEMPLATE
    }
  },
  computed: {
    lockedProtocol() {
      return !!this.protocol
    },
    // 锁定单设备时（设备下钻/插件接入）不重复显示所属设备
    showDeviceColumn() {
      return !this.deviceCode
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
        this.list = (data.list || []).map(item => ({ ...item, _loading: false, _profiles: [], _profilesLoading: true }))
        this.total = data.total || 0
        this.assignOrdinals()
        this.list.forEach(row => this.loadProfiles(row))
      }).finally(() => { this.loading = false })
    },
    // 通道号 = 顺位号：按设备分组、组内顺序 1,2,3…（与协议原生通道号无关）。
    assignOrdinals() {
      const counter = {}
      this.list.forEach(row => {
        const key = row.deviceInternalCode || row.deviceId || ''
        counter[key] = (counter[key] || 0) + 1
        row._ordinal = counter[key]
      })
    },
    // 码流类型下拉：取资源下已投影的码流（插件当前只有主码流）
    loadProfiles(row) {
      getResourceDetail(row.resourceId).then(res => {
        const detail = (res && res.data) || {}
        const profiles = detail.profiles || []
        if (profiles.length) {
          row._profiles = profiles.map(p => ({
            id: p.id,
            name: p.name || (p.role === 'sub' ? '子码流' : '主码流')
          }))
          const def = profiles.find(p => p.isDefault) || profiles[0]
          row._profileId = def.id
        } else {
          row._profiles = [{ id: '', name: '主码流' }]
          row._profileId = ''
        }
      }).catch(() => {
        row._profiles = [{ id: '', name: '主码流' }]
        row._profileId = ''
      }).finally(() => { row._profilesLoading = false })
    },
    openEdit(row) {
      this.editCtx = { internalCode: row.internalCode, protocol: row.protocol }
    },
    closeEdit() {
      this.editCtx = null
      this.load()
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
      player.openDialog('media', '', '', { pending: true, name: row.name, hasAudio: !!row.hasAudio, ptzType: 0, showPtz: false })
      row._loading = true
      this.playingRow = row
      startResourceLive(row.resourceId, row._profileId).then(res => {
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
.unified-channel .sub-text { color: #80868B; font-size: 12px; }
.unified-channel .device-link { cursor: pointer; }
.unified-channel .device-link:hover { color: #1A73E8; }
.unified-channel .cap-item {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  margin-right: 10px;
  color: #C0C4CC;
  white-space: nowrap;
}
.unified-channel .cap-item.cap-on { color: #1A73E8; }
</style>
