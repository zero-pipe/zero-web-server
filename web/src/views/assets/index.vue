<template>
  <div class="app-container asset-page">
    <!-- 顶部统计卡 -->
    <div class="stat-row">
      <div class="stat-card">
        <div class="stat-head"><span>接入设备总数</span><i class="el-icon-office-building" /></div>
        <div class="stat-num">{{ deviceTotal }} <span class="unit">台</span></div>
        <div class="stat-sub"><span class="dot on" /> 在线 {{ deviceOnline }} <span class="dot off" /> 离线 {{ deviceTotal - deviceOnline }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-head"><span>关联通道总数</span><i class="el-icon-video-camera" /></div>
        <div class="stat-num">{{ channelTotal }} <span class="unit">路</span></div>
        <div class="stat-sub"><span class="dot on" /> 在线 {{ channelOnline }} <span class="dot off" /> 离线 {{ channelTotal - channelOnline }}</div>
      </div>
      <div class="stat-card">
        <div class="stat-head"><span>协议分布</span></div>
        <div class="proto-tags">
          <span class="proto-chip gb">GB28181 · {{ protoCount('gb28181') }}</span>
          <span class="proto-chip onvif">ONVIF · {{ protoCount('onvif') }}</span>
          <span class="proto-chip plugin">插件 · {{ protoCount('plugin') }}</span>
        </div>
        <div class="proto-bar">
          <span class="seg gb" :style="{ width: protoPct('gb28181') }" />
          <span class="seg onvif" :style="{ width: protoPct('onvif') }" />
          <span class="seg plugin" :style="{ width: protoPct('plugin') }" />
        </div>
      </div>
    </div>

    <!-- 过滤栏 -->
    <el-form :inline="true" size="mini" class="asset-filter">
      <el-form-item>
        <el-input v-model="query" clearable placeholder="设备/通道名称、IP、编号" prefix-icon="el-icon-search" style="width:230px" @input="reload" />
      </el-form-item>
      <el-form-item label="协议">
        <el-select v-model="protocol" clearable placeholder="全部" style="width:120px" @change="reload">
          <el-option label="GB28181" value="gb28181" />
          <el-option label="ONVIF" value="onvif" />
          <el-option label="插件" value="plugin" />
        </el-select>
      </el-form-item>
      <el-form-item label="状态">
        <el-select v-model="status" clearable placeholder="全部" style="width:100px" @change="reload">
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button-group>
          <el-button :type="view === 'tree' ? 'primary' : ''" icon="el-icon-s-grid" @click="setView('tree')">设备聚合(树形)</el-button>
          <el-button :type="view === 'flat' ? 'primary' : ''" icon="el-icon-s-unfold" @click="setView('flat')">通道视角(扁平)</el-button>
        </el-button-group>
      </el-form-item>
      <el-form-item v-if="view === 'tree'">
        <el-button icon="el-icon-sort" @click="toggleExpandAll">{{ allExpanded ? '全部折叠' : '全部展开' }}</el-button>
      </el-form-item>
      <el-form-item style="float:right">
        <el-button icon="el-icon-refresh" circle :loading="loading" @click="loadData" />
      </el-form-item>
    </el-form>

    <!-- 树表：设备父行 + 通道子行 -->
    <el-table
      ref="table"
      v-loading="loading"
      :data="rows"
      :row-key="rowKey"
      :tree-props="{ children: 'children' }"
      style="width:100%"
      size="small"
      header-row-class-name="table-header"
      :row-class-name="rowClass"
      max-height="calc(100vh - 300px)"
    >
      <el-table-column label="设备 / 通道名称" min-width="220">
        <template slot-scope="{ row }">
          <template v-if="row._kind === 'device'">
            <i class="el-icon-office-building node-ico" />
            <span class="node-name">{{ row.name }}</span>
            <span class="node-count">({{ row.channelCount != null ? row.channelCount : (row.children || []).length }} 个通道)</span>
          </template>
          <template v-else>
            <i class="el-icon-video-camera node-ico ch" />
            <span class="node-name">{{ row.name }}</span>
            <el-tag v-if="row.kindLabel" size="mini" effect="plain" class="kind-tag">{{ row.kindLabel }}</el-tag>
          </template>
        </template>
      </el-table-column>

      <el-table-column label="接入协议 / 厂商型号" min-width="170">
        <template slot-scope="{ row }">
          <template v-if="row._kind === 'device'">
            <el-tag size="mini" :type="protoTagType(row.protocol)">{{ protocolLabel(row.protocol) }}</el-tag>
            <span class="sub-text">{{ row.vendor }}</span>
            <span class="sub-text">{{ row.model }}</span>
          </template>
          <span v-else class="sub-text">继承设备 [{{ protocolLabel(row.protocol) }}]</span>
        </template>
      </el-table-column>

      <el-table-column label="IP / 编号" min-width="180">
        <template slot-scope="{ row }">
          <template v-if="row._kind === 'device'">
            <div class="mono">{{ row.addressText || '—' }}</div>
            <div class="sub-text mono">{{ deviceCode(row) }}</div>
          </template>
          <span v-else class="mono sub-text">{{ row.externalKey || row.gbCode || '—' }}</span>
        </template>
      </el-table-column>

      <el-table-column label="音频 / 通道" width="100" align="center">
        <template slot-scope="{ row }">
          <template v-if="row._kind === 'device'"><span class="muted">—</span></template>
          <i v-else :class="row.hasAudio ? 'el-icon-microphone audio-on' : 'el-icon-microphone audio-off'" :title="row.hasAudio ? '有音频' : '无音频'" />
        </template>
      </el-table-column>

      <el-table-column label="设备能力集" min-width="260">
        <template slot-scope="{ row }">
          <template v-if="row._kind === 'device'">
            <el-tag v-for="c in (row.capabilities || [])" :key="c" size="mini" type="info" class="cap-chip">{{ capLabel(c) }}</el-tag>
          </template>
          <template v-else>
            <span v-for="c in row.caps || []" :key="c" class="cap-item cap-on">
              <i :class="capIcon(c)" /> {{ capLabel(c) }}
            </span>
            <span v-if="!(row.caps || []).length" class="muted">—</span>
          </template>
        </template>
      </el-table-column>

      <el-table-column label="状态" width="90" align="center">
        <template slot-scope="{ row }">
          <el-tag :type="row.online ? 'success' : 'info'" size="mini">{{ row.online ? '在线' : '离线' }}</el-tag>
        </template>
      </el-table-column>

      <el-table-column label="操作" width="200" fixed="right">
        <template slot-scope="{ row }">
          <template v-if="row._kind === 'device'">
            <el-button type="text" size="mini" icon="el-icon-refresh" @click="syncDevice(row)">同步</el-button>
            <el-divider direction="vertical" />
            <el-button type="text" size="mini" icon="el-icon-edit" @click="editDevice(row)">配置与详情</el-button>
          </template>
          <template v-else>
            <el-button type="text" size="mini" icon="el-icon-edit" @click="editChannel(row)">编辑</el-button>
          </template>
        </template>
      </el-table-column>
    </el-table>

    <!-- 设备编辑弹窗（复用统一设备编辑） -->
    <edit-dialog :visible.sync="deviceEditVisible" :row="deviceEditRow" @success="loadData" />

    <!-- 通道编辑弹窗（复用统一通道编辑表单） -->
    <el-dialog title="编辑通道" :visible.sync="channelEditVisible" width="680px" append-to-body @closed="onChannelEditClosed">
      <channel-edit v-if="channelEditVisible" :internal-code="channelEditCtx.internalCode" :protocol="channelEditCtx.protocol" :close-edit="closeChannelEdit" />
    </el-dialog>
  </div>
</template>

<script>
import { listDevices, syncDevice as syncUnifiedDevice } from '@/api/devices'
import { queryChannels } from '@/api/asset'
import editDialog from '@/views/devices/editDialog.vue'
import ChannelEdit from '@/views/device/channel/edit.vue'

const CAP_LABELS = {
  play: '预览', playback: '回放', record: '录像', ptz: '云台', snapshot: '抓图',
  event: '事件', alarm: '报警', subscribe: '订阅', catalog: '目录', sync: '同步',
  stats: '统计', edit: '编辑', delete: '删除'
}
const CAP_ICONS = {
  play: 'el-icon-video-play', playback: 'el-icon-video-camera-solid', record: 'el-icon-folder-opened',
  ptz: 'el-icon-aim', snapshot: 'el-icon-camera', event: 'el-icon-bell'
}
const KIND_LABELS = {
  video_sensor: '视频通道', audio_input: '音频输入', audio_output: '音频输出',
  alarm_input: '报警输入', relay_output: '继电器', metadata_source: '数据通道'
}

export default {
  name: 'DeviceAssets',
  components: { editDialog, ChannelEdit },
  data() {
    return {
      loading: false,
      query: '',
      protocol: '',
      status: '',
      view: 'tree',
      devices: [],
      channels: [],
      allExpanded: true,
      deviceEditVisible: false,
      deviceEditRow: null,
      channelEditVisible: false,
      channelEditCtx: { internalCode: '', protocol: '' }
    }
  },
  computed: {
    deviceTotal() { return this.devices.length },
    deviceOnline() { return this.devices.filter(d => d.online).length },
    channelTotal() { return this.channels.length },
    channelOnline() { return this.channels.filter(c => c.online).length },
    // 扁平视图：所有通道作为平铺行（设备信息内联在行内）
    rows() {
      if (this.view === 'flat') {
        return this.channels.map(c => ({ ...c, _kind: 'channel' }))
      }
      return this.buildTree()
    }
  },
  created() {
    const q = (this.$route && this.$route.query) || {}
    if (q.query) this.query = q.query
    this.loadData()
  },
  methods: {
    protocolLabel(p) { return { gb28181: 'GB28181', onvif: 'ONVIF', plugin: '插件' }[p] || (p || '—') },
    // 设备级编号：国标展示 GB 20 位码，其它展示内码
    deviceCode(row) {
      if (row.protocol === 'gb28181' && row.gb && row.gb.deviceId) return 'GB ' + row.gb.deviceId
      return row.internalCode || '—'
    },
    protoTagType(p) { return p === 'onvif' ? '' : (p === 'plugin' ? 'warning' : 'success') },
    capLabel(c) { return CAP_LABELS[c] || c },
    capIcon(c) { return CAP_ICONS[c] || 'el-icon-check' },
    // row-key 用后端保证唯一的 id（gb:xxx / onvif:n / plugin:n），避免内码为空时树表丢行
    rowKey(row) { return row._kind === 'device' ? 'd:' + row.id : 'r:' + row.resourceId },
    rowClass({ row }) { return row._kind === 'device' ? 'row-device' : 'row-channel' },
    protoCount(p) { return this.devices.filter(d => d.protocol === p).length },
    protoPct(p) {
      const total = this.deviceTotal || 1
      return (this.protoCount(p) / total * 100) + '%'
    },
    setView(v) {
      this.view = v
      // 切回树形且处于「全部展开」时重新展开（扁平视图下无展开概念）
      if (v === 'tree' && this.allExpanded) {
        this.$nextTick(() => {
          const table = this.$refs.table
          if (!table) return
          this.rows.forEach(row => {
            if (row._kind === 'device' && (row.children || []).length) {
              table.toggleRowExpansion(row, true)
            }
          })
        })
      }
    },
    reload() { this.loadData() },
    loadData() {
      this.loading = true
      Promise.all([
        listDevices({ page: 1, count: 999, query: this.query, protocol: this.protocol || undefined, status: this.status || undefined }),
        queryChannels({ page: 1, count: 2000, query: this.query, protocol: this.protocol || undefined })
      ]).then(([devRes, chRes]) => {
        const dData = (devRes && devRes.data) || {}
        const cData = (chRes && chRes.data) || {}
        // DeviceView 的在线态是 status 字符串，统一归一化为 online 布尔供模板使用
        this.devices = (dData.list || []).map(d => ({ ...d, online: d.status === 'online', _kind: 'device' }))
        let channels = (cData.list || []).map(c => ({
          ...c,
          _kind: 'channel',
          online: c.online !== undefined ? c.online : c.status === 'online',
          kindLabel: KIND_LABELS[c.kind] || (c.kind === 'video_sensor' ? '视频通道' : '')
        }))
        // 设备已按协议/状态过滤，通道跟随设备可见性
        if (this.status) {
          channels = channels.filter(c => (this.status === 'online') === !!c.online)
        }
        this.channels = channels
        // 树形视图且处于「全部展开」时，加载后主动展开所有设备行
        if (this.view === 'tree' && this.allExpanded) {
          this.$nextTick(() => {
            const table = this.$refs.table
            if (!table) return
            this.rows.forEach(row => {
              if (row._kind === 'device' && (row.children || []).length) {
                table.toggleRowExpansion(row, true)
              }
            })
          })
        }
      }).catch(err => {
        this.$message.error((err && err.message) || '加载失败')
      }).finally(() => { this.loading = false })
    },
    buildTree() {
      const byDevice = {}
      this.channels.forEach(c => {
        const k = c.deviceInternalCode || c.deviceId || ''
        ;(byDevice[k] = byDevice[k] || []).push({ ...c, _kind: 'channel' })
      })
      return this.devices.map(d => {
        const kids = byDevice[d.internalCode] || []
        const node = { ...d, _kind: 'device' }
        // 仅有子通道时才挂 children，避免 el-table 给空设备画展开箭头
        if (kids.length) node.children = kids
        return node
      })
    },
    toggleExpandAll() {
      this.allExpanded = !this.allExpanded
      const expand = this.allExpanded
      // 用 table 实例逐行展开/收起，比纯 expand-row-keys 更可靠（折叠时后者不生效）
      this.$nextTick(() => {
        const table = this.$refs.table
        if (!table) return
        this.rows.forEach(row => {
          if (row._kind === 'device' && (row.children || []).length) {
            table.toggleRowExpansion(row, expand)
          }
        })
      })
      this.allExpanded = expand
    },
    syncDevice(row) {
      syncUnifiedDevice(row.id).then(() => {
        this.$message.success('已触发同步')
        this.loadData()
      }).catch(err => {
        this.$message.error((err && err.message) || '同步失败')
      })
    },
    editDevice(row) {
      this.deviceEditRow = row
      this.deviceEditVisible = true
    },
    editChannel(row) {
      this.channelEditCtx = { internalCode: row.internalCode, protocol: row.protocol }
      this.channelEditVisible = true
    },
    closeChannelEdit() {
      this.channelEditVisible = false
    },
    onChannelEditClosed() {
      this.loadData()
    }
  }
}
</script>

<style scoped>
.asset-page { display: flex; flex-direction: column; }
.stat-row { display: flex; gap: 14px; margin-bottom: 14px; }
.stat-card {
  flex: 1;
  background: #fff;
  border: 1px solid #ebeef5;
  border-radius: 10px;
  padding: 14px 18px;
  box-shadow: 0 1px 3px rgba(0, 0, 0, .04);
}
.stat-head { display: flex; justify-content: space-between; align-items: center; color: #606266; font-size: 13px; }
.stat-head i { color: #c0c4cc; font-size: 16px; }
.stat-num { font-size: 26px; font-weight: 700; color: #303133; margin: 6px 0 4px; }
.stat-num .unit { font-size: 13px; font-weight: 400; color: #909399; }
.stat-sub { font-size: 12px; color: #909399; display: flex; align-items: center; gap: 5px; }
.dot { width: 7px; height: 7px; border-radius: 50%; display: inline-block; margin-left: 6px; }
.dot.on { background: #34a853; }
.dot.off { background: #c0c4cc; }
.proto-tags { display: flex; gap: 8px; margin: 8px 0; flex-wrap: wrap; }
.proto-chip { font-size: 12px; padding: 2px 8px; border-radius: 12px; }
.proto-chip.gb { color: #1A73E8; background: #e8f0fe; }
.proto-chip.onvif { color: #7c3aed; background: #f5f3ff; }
.proto-chip.plugin { color: #e6a23c; background: #fdf6ec; }
.proto-bar { display: flex; height: 6px; border-radius: 3px; overflow: hidden; background: #f0f2f5; }
.proto-bar .seg { height: 100%; }
.proto-bar .seg.gb { background: #1A73E8; }
.proto-bar .seg.onvif { background: #7c3aed; }
.proto-bar .seg.plugin { background: #e6a23c; }
.asset-filter { margin-bottom: 6px; }
.node-ico { color: #1A73E8; margin-right: 6px; }
.node-ico.ch { color: #909399; }
.node-name { font-weight: 600; color: #303133; }
.node-count { color: #a8abb2; font-size: 12px; margin-left: 6px; }
.kind-tag { margin-left: 6px; }
.sub-text { color: #909399; font-size: 12px; margin-left: 6px; }
.mono { font-family: Menlo, Consolas, monospace; font-size: 12px; color: #606266; }
.muted { color: #c0c4cc; }
.cap-chip { margin: 0 4px 3px 0; }
.cap-item { display: inline-flex; align-items: center; gap: 2px; margin-right: 10px; color: #C0C4CC; white-space: nowrap; font-size: 12px; }
.cap-item.cap-on { color: #1A73E8; }
.audio-on { color: #34a853; }
.audio-off { color: #c0c4cc; }
::v-deep .row-device { background: #fafbfc; }
</style>
