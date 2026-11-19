<template>
  <div class="app-container asset-page">
    <el-form :inline="true" size="mini" class="asset-filter">
      <el-form-item label="搜索">
        <el-input v-model="query" clearable placeholder="名称 / 序列号 / IP" style="width:220px" @input="reload" />
      </el-form-item>
      <el-form-item label="接入">
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
        <el-button icon="el-icon-refresh" circle :loading="loading" @click="loadTree" />
      </el-form-item>
    </el-form>

    <div class="asset-body">
      <el-card shadow="never" class="tree-card">
        <div slot="header" class="card-header">
          <span>设备与通道</span>
          <span class="muted">{{ treeData.length }} 台设备</span>
        </div>
        <el-input v-model="treeFilter" size="mini" clearable placeholder="过滤设备/通道" prefix-icon="el-icon-search" style="margin-bottom:8px" />
        <el-tree
          ref="tree"
          v-loading="loading"
          :data="treeData"
          node-key="id"
          :props="treeProps"
          :filter-node-method="filterNode"
          highlight-current
          @node-click="onNodeClick"
        >
          <span slot-scope="{ data }" class="tree-node">
            <i :class="nodeIcon(data)" />
            <span class="tree-label">{{ data.label }}</span>
            <el-tag v-if="data.online !== undefined" :type="data.online ? 'success' : 'info'" size="mini" class="tree-tag">
              {{ data.online ? '在线' : '离线' }}
            </el-tag>
            <el-tag v-if="data.kindLabel" size="mini" type="info" class="tree-tag">{{ data.kindLabel }}</el-tag>
          </span>
        </el-tree>
      </el-card>

      <el-card shadow="never" class="detail-card">
        <!-- 设备详情 -->
        <div v-if="selectedDevice">
          <div class="detail-head">
            <div>
              <span class="detail-title">{{ selectedDevice.device.name }}</span>
              <el-tag size="mini" style="margin-left:8px">{{ selectedDevice.binding ? selectedDevice.binding.protocol : '未绑定' }}</el-tag>
              <el-tag size="mini" :type="selectedDevice.device.online ? 'success' : 'info'" style="margin-left:4px">
                {{ selectedDevice.device.online ? '在线' : '离线' }}
              </el-tag>
            </div>
            <div>
              <el-button size="mini" @click="syncDevice">同步资源</el-button>
            </div>
          </div>

          <el-form :model="deviceForm" label-width="90px" size="small" class="edit-form">
            <el-row :gutter="16">
              <el-col :span="12"><el-form-item label="设备名称"><el-input v-model="deviceForm.name" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="厂商"><el-input v-model="deviceForm.vendor" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="型号"><el-input v-model="deviceForm.model" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="序列号"><el-input v-model="deviceForm.serialNumber" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="启用"><el-switch v-model="deviceForm.enabled" /></el-form-item></el-col>
            </el-row>
            <el-button size="small" type="primary" :loading="saving" @click="saveDevice">保存设备</el-button>
          </el-form>

          <div class="section-title">接入信息</div>
          <el-descriptions :column="2" size="small" border>
            <el-descriptions-item label="接入协议">{{ selectedDevice.binding ? selectedDevice.binding.protocol : '—' }}</el-descriptions-item>
            <el-descriptions-item label="接入方式">{{ selectedDevice.binding ? selectedDevice.binding.mode : '—' }}</el-descriptions-item>
            <el-descriptions-item label="接入地址">{{ accessAddress }}</el-descriptions-item>
            <el-descriptions-item label="国标编号">{{ gbDeviceCode }}</el-descriptions-item>
          </el-descriptions>

          <div class="section-title">能力</div>
          <el-tag v-for="c in selectedDevice.capabilities" :key="c.id" size="mini" style="margin:0 4px 4px 0"
                  :type="c.available ? 'success' : 'info'">{{ c.operation }}</el-tag>

          <div class="section-title">通道 / 资源（{{ selectedDevice.resources.length }}）</div>
          <el-table :data="selectedDevice.resources" size="small" border>
            <el-table-column label="名称" min-width="140">
              <template slot-scope="{ row }">
                <el-button type="text" size="mini" @click="selectResource(row)">{{ row.resource.name || row.resource.internalCode }}</el-button>
              </template>
            </el-table-column>
            <el-table-column label="类型" width="110">
              <template slot-scope="{ row }">{{ kindLabel(row.resource.kind) }}</template>
            </el-table-column>
            <el-table-column label="经纬度" min-width="150">
              <template slot-scope="{ row }">
                <span v-if="row.resource.longitude || row.resource.latitude">{{ row.resource.longitude }}, {{ row.resource.latitude }}</span>
                <span v-else class="muted">—</span>
              </template>
            </el-table-column>
            <el-table-column label="能力" min-width="160">
              <template slot-scope="{ row }">
                <el-tag v-for="c in row.capabilities" :key="c.id" size="mini" style="margin-right:3px">{{ c.operation }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column label="操作" width="110">
              <template slot-scope="{ row }">
                <el-button type="text" size="mini" @click="selectResource(row)">编辑</el-button>
                <el-button v-if="row.resource.kind === 'video_sensor'" type="text" size="mini" @click="playResource(row)">播放</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <!-- 通道详情 -->
        <div v-else-if="selectedResource">
          <div class="detail-head">
            <div>
              <el-button type="text" icon="el-icon-back" @click="backToDevice">返回设备</el-button>
              <span class="detail-title">{{ resourceForm.name || '未命名通道' }}</span>
              <el-tag size="mini" type="info" style="margin-left:8px">{{ kindLabel(selectedResource.resource.kind) }}</el-tag>
            </div>
            <div>
              <el-button v-if="selectedResource.resource.kind === 'video_sensor'" size="mini" type="primary" @click="playResource(selectedResource)">播放</el-button>
            </div>
          </div>

          <el-form :model="resourceForm" label-width="100px" size="small" class="edit-form">
            <el-row :gutter="16">
              <el-col :span="12"><el-form-item label="通道名称"><el-input v-model="resourceForm.name" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="资源类型"><el-input :value="kindLabel(selectedResource.resource.kind)" disabled /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="经度"><el-input-number v-model="resourceForm.longitude" :precision="6" :step="0.000001" style="width:100%" controls-position="right" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="纬度"><el-input-number v-model="resourceForm.latitude" :precision="6" :step="0.000001" style="width:100%" controls-position="right" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="地图层级"><el-input-number v-model="resourceForm.mapLevel" :min="0" :max="30" style="width:100%" controls-position="right" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="PTZ 类型"><el-input-number v-model="resourceForm.ptzType" :min="0" style="width:100%" controls-position="right" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="音频"><el-switch v-model="resourceForm.hasAudio" /></el-form-item></el-col>
              <el-col :span="12"><el-form-item label="启用"><el-switch v-model="resourceForm.enabled" /></el-form-item></el-col>
              <el-col :span="24"><el-form-item label="标签"><el-input v-model="resourceForm.tags" placeholder="逗号分隔" /></el-form-item></el-col>
            </el-row>
            <el-button size="small" type="primary" :loading="saving" @click="saveChannel">保存通道</el-button>
          </el-form>

          <div class="section-title">码流</div>
          <el-table :data="selectedResource.profiles" size="small" border>
            <el-table-column prop="name" label="名称" min-width="120" />
            <el-table-column prop="role" label="角色" width="90" />
            <el-table-column prop="resolution" label="分辨率" width="120" />
            <el-table-column prop="codec" label="编码" width="90" />
            <el-table-column label="默认" width="70">
              <template slot-scope="{ row }"><el-tag v-if="row.isDefault" size="mini" type="success">默认</el-tag></template>
            </el-table-column>
          </el-table>

          <div class="section-title">能力</div>
          <el-tag v-for="c in selectedResource.capabilities" :key="c.id" size="mini" style="margin:0 4px 4px 0"
                  :type="c.available ? 'success' : 'info'">{{ c.operation }}</el-tag>
        </div>

        <div v-else class="empty-detail">
          <i class="el-icon-connection" />
          <div>从左侧选择设备或通道</div>
        </div>
      </el-card>
    </div>

    <el-dialog :title="playing ? ('播放 - ' + (playing.resource.name || '未命名通道')) : '播放'" :visible.sync="playerVisible" width="820px" @closed="stopLive">
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
import {
  queryAssetTree, getAssetDetail, updateAsset, updateChannel,
  syncAsset, startResourceLive, stopResourceLive
} from '@/api/asset'

const KIND_LABELS = {
  video_sensor: '视频通道',
  audio_input: '音频输入',
  audio_output: '音频输出',
  alarm_input: '报警输入',
  relay_output: '继电器',
  metadata_source: '数据通道'
}

export default {
  name: 'DeviceAssets',
  components: { jessibuca },
  data() {
    return {
      loading: false,
      saving: false,
      query: '',
      protocol: '',
      status: '',
      treeFilter: '',
      nodes: [],
      treeData: [],
      treeProps: { label: 'label', children: 'children' },
      selectedDevice: null,
      selectedResource: null,
      deviceForm: {},
      resourceForm: {},
      playerVisible: false,
      playing: null,
      streamInfo: null,
      videoUrl: ''
    }
  },
  watch: {
    treeFilter(v) { this.$refs.tree && this.$refs.tree.filter(v) }
  },
  computed: {
    accessAddress() {
      const b = this.selectedDevice && this.selectedDevice.binding
      if (!b) return '—'
      if (b.protocol === 'gb28181') return b.endpoint || '—'
      return b.endpoint || b.externalId || '—'
    },
    gbDeviceCode() {
      const b = this.selectedDevice && this.selectedDevice.binding
      return b && b.protocol === 'gb28181' ? (b.externalId || '—') : '—'
    }
  },
  created() {
    const q = (this.$route && this.$route.query) || {}
    if (q.query) this.query = q.query
    this.loadTree()
  },
  beforeDestroy() { this.stopLive() },
  methods: {
    reload() { this.loadTree() },
    loadTree() {
      this.loading = true
      queryAssetTree({ query: this.query, protocol: this.protocol, status: this.status })
        .then(res => {
          this.nodes = (res.data && res.data.list) || []
          this.buildTree()
        }).finally(() => { this.loading = false })
    },
    buildTree() {
      this.treeData = this.nodes.map(n => {
        const device = n.device
        const children = (n.resources || []).map(r => ({
          id: 'r:' + r.resource.id,
          label: r.resource.name || '未命名通道',
          kindLabel: KIND_LABELS[r.resource.kind] || r.resource.kind,
          type: 'resource',
          node: r,
          children: (r.profiles || []).map(p => ({
            id: 'p:' + p.id,
            label: (p.name || p.role) + (p.resolution ? ' · ' + p.resolution : ''),
            type: 'profile',
            node: p
          }))
        }))
        return {
          id: 'd:' + device.id,
          label: device.name,
          online: device.online,
          kindLabel: n.binding ? n.binding.protocol : '',
          type: 'device',
          node: n,
          children
        }
      })
    },
    filterNode(value, data) {
      if (!value) return true
      const node = data.node || {}
      const hay = [data.label, node.name, node.device && node.device.name, node.device && node.device.serialNumber].filter(Boolean).join(' ')
      return hay.toLowerCase().includes(value.toLowerCase())
    },
    nodeIcon(data) {
      if (data.type === 'device') return 'el-icon-office-building'
      if (data.type === 'profile') return 'el-icon-video-camera'
      return data.node && data.node.resource && data.node.resource.kind === 'video_sensor' ? 'el-icon-video-camera-solid' : 'el-icon-connection'
    },
    onNodeClick(data) {
      if (data.type === 'device') {
        this.openDevice(data.node.device.id)
      } else if (data.type === 'resource') {
        if (!this.selectedDevice || this.selectedDevice.device.id !== data.node.resource.deviceId) {
          this.openDevice(data.node.resource.deviceId, () => this.selectResource(data.node))
        } else {
          this.selectResource(data.node)
        }
      }
    },
    openDevice(deviceId, done) {
      getAssetDetail(deviceId).then(res => {
        const detail = res.data || {}
        const resources = (detail.resources || []).map(r => ({ resource: r, profiles: [], capabilities: [] }))
        // 详情接口不返回资源级 profile/能力，从树节点补齐。
        const treeNode = this.nodes.find(n => n.device.id === deviceId)
        if (treeNode) {
          resources.forEach(r => {
            const tr = (treeNode.resources || []).find(x => x.resource.id === r.resource.id)
            if (tr) {
              r.profiles = tr.profiles || []
              r.capabilities = tr.capabilities || []
            }
          })
        }
        this.selectedDevice = {
          device: detail.device || {},
          binding: detail.binding || null,
          capabilities: detail.capabilities || [],
          resources
        }
        this.deviceForm = {
          name: this.selectedDevice.device.name,
          vendor: this.selectedDevice.device.vendor,
          model: this.selectedDevice.device.model,
          serialNumber: this.selectedDevice.device.serialNumber,
          enabled: this.selectedDevice.device.enabled
        }
        this.selectedResource = null
        if (done) done()
      })
    },
    selectResource(node) {
      this.selectedResource = node
      const r = node.resource || {}
      this.resourceForm = {
        name: r.name,
        longitude: Number(r.longitude) || 0,
        latitude: Number(r.latitude) || 0,
        mapLevel: Number(r.mapLevel) || 0,
        ptzType: Number(r.ptzType) || 0,
        hasAudio: !!r.hasAudio,
        enabled: r.enabled !== false,
        tags: r.tags || ''
      }
    },
    backToDevice() {
      if (this.selectedDevice) this.openDevice(this.selectedDevice.device.id)
    },
    saveDevice() {
      if (!this.selectedDevice) return
      this.saving = true
      updateAsset(this.selectedDevice.device.id, this.deviceForm).then(() => {
        this.$message.success('设备已保存')
        this.loadTree()
        this.openDevice(this.selectedDevice.device.id)
      }).finally(() => { this.saving = false })
    },
    saveChannel() {
      if (!this.selectedResource) return
      this.saving = true
      updateChannel(this.selectedResource.resource.internalCode, this.resourceForm).then(() => {
        this.$message.success('通道已保存')
        const deviceId = this.selectedDevice && this.selectedDevice.device.id
        this.loadTree()
        if (deviceId) this.openDevice(deviceId)
      }).finally(() => { this.saving = false })
    },
    syncDevice() {
      if (!this.selectedDevice) return
      syncAsset(this.selectedDevice.device.id).then(() => {
        this.$message.success('资源已同步')
        this.loadTree()
        this.openDevice(this.selectedDevice.device.id)
      })
    },
    playResource(node) {
      const profile = (node.profiles || []).find(p => p.isDefault) || (node.profiles || [])[0] || {}
      startResourceLive(node.resource.id, profile.id || '').then(res => {
        const content = res.data
        if (!content) { this.$message.error('未返回播放地址'); return }
        this.playing = node
        this.streamInfo = content
        this.videoUrl = content.wsFlv || content.flv || ''
        this.playerVisible = true
      })
    },
    stopLive() {
      if (this.playing) stopResourceLive(this.playing.resource.id).catch(() => {})
      this.playerVisible = false
      this.videoUrl = ''
      this.streamInfo = null
      this.playing = null
    },
    kindLabel(kind) { return KIND_LABELS[kind] || kind || '—' }
  }
}
</script>

<style scoped>
.asset-filter { margin-bottom: 8px; }
.asset-body { display: flex; gap: 12px; height: calc(100vh - 190px); }
.tree-card { width: 320px; flex: 0 0 320px; overflow: auto; }
.detail-card { flex: 1; overflow: auto; }
.card-header { display: flex; justify-content: space-between; align-items: center; }
.muted { color: #909399; font-size: 12px; }
.tree-node { display: flex; align-items: center; gap: 5px; min-width: 0; }
.tree-label { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tree-tag { transform: scale(0.9); }
.detail-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px; }
.detail-title { font-size: 17px; font-weight: 600; color: #303133; }
.section-title { margin: 16px 0 8px; font-weight: 600; color: #303133; }
.edit-form { max-width: 900px; }
.empty-detail { height: 60vh; display: flex; flex-direction: column; align-items: center; justify-content: center; color: #c0c4cc; gap: 10px; font-size: 14px; }
.empty-detail i { font-size: 42px; }
.stream-links { margin-top: 10px; color: #606266; font-size: 12px; word-break: break-all; }
</style>
