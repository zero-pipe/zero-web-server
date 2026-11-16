<template>
  <div class="app-container">
    <el-card shadow="never">
      <div slot="header" class="card-header">
        <span>插件实例</span>
        <el-button type="text" icon="el-icon-refresh" @click="loadAll">刷新</el-button>
      </div>
      <el-table v-loading="loading" :data="plugins" size="small" border>
        <el-table-column prop="id" label="实例 ID" min-width="140" />
        <el-table-column prop="pluginId" label="插件" min-width="120" />
        <el-table-column prop="vendor" label="厂商" width="100" />
        <el-table-column prop="version" label="版本" width="90" />
        <el-table-column label="协议" min-width="140">
          <template slot-scope="{ row }">
            <el-tag v-for="p in row.protocols" :key="p" size="mini" style="margin-right:4px">{{ p }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="能力" min-width="160">
          <template slot-scope="{ row }">
            <el-tag v-for="c in row.capabilities" :key="c" size="mini" type="info" style="margin-right:4px">{{ c }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template slot-scope="{ row }">
            <el-tag :type="row.online ? 'success' : 'info'" size="mini">{{ row.online ? '在线' : '离线' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="endpoint" label="Endpoint" min-width="160" />
        <el-table-column label="最后心跳" width="170">
          <template slot-scope="{ row }">{{ formatTime(row.lastSeen) }}</template>
        </el-table-column>
      </el-table>
      <div class="hint">插件启动后调用 <code>POST /api/v1/plugins/hello</code> 注册；无插件时请先启动 mock 插件。</div>
    </el-card>

    <el-card shadow="never" style="margin-top:16px">
      <div slot="header" class="card-header">
        <span>插件设备</span>
        <el-button type="primary" size="small" icon="el-icon-plus" @click="openAdd" :disabled="!plugins.length">添加设备</el-button>
      </div>
      <el-table v-loading="loading" :data="devices" size="small" border>
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="protocol" label="协议" width="140" />
        <el-table-column prop="deviceUid" label="设备 UID" min-width="140" />
        <el-table-column prop="manufacturer" label="厂商" width="100" />
        <el-table-column prop="model" label="型号" width="120" />
        <el-table-column label="在线" width="80">
          <template slot-scope="{ row }">
            <el-tag :type="row.online ? 'success' : 'info'" size="mini">{{ row.online ? '在线' : '离线' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="channelCount" label="通道数" width="80" />
        <el-table-column label="操作" width="260">
          <template slot-scope="{ row }">
            <el-button type="text" size="mini" @click="openChannels(row)">通道</el-button>
            <el-button type="text" size="mini" @click="syncDevice(row)">同步</el-button>
            <el-button type="text" size="mini" style="color:#f56c6c" @click="removeDevice(row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <!-- 添加设备 -->
    <el-dialog title="添加插件设备" :visible.sync="addVisible" width="520px" @closed="resetAdd">
      <el-form ref="addForm" :model="addForm" label-width="100px" size="small">
        <el-form-item label="插件">
          <el-select v-model="addForm.pluginId" style="width:100%" @change="onPluginChange">
            <el-option v-for="p in onlinePlugins" :key="p.id" :label="p.id + ' (' + p.protocols.join(',') + ')'" :value="p.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="协议">
          <el-select v-model="addForm.protocol" style="width:100%">
            <el-option v-for="p in protocolOptions" :key="p" :label="p" :value="p" />
          </el-select>
        </el-form-item>
        <el-form-item label="设备名称">
          <el-input v-model="addForm.name" placeholder="留空则使用设备 UID" />
        </el-form-item>
        <el-form-item v-for="(f, key) in schemaProps" :key="key" :label="f.title || key">
          <el-input v-if="isNumber(f)" v-model.number="addForm.params[key]" />
          <el-input v-else-if="f.format === 'password'" v-model="addForm.params[key]" type="password" show-password />
          <el-input v-else v-model="addForm.params[key]" />
          <div v-if="f.description" class="field-desc">{{ f.description }}</div>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button size="small" @click="addVisible = false">取消</el-button>
        <el-button size="small" type="primary" :loading="submitting" @click="submitAdd">确定</el-button>
      </div>
    </el-dialog>

    <!-- 通道 -->
    <el-dialog :title="'通道 - ' + (currentDevice && currentDevice.name)" :visible.sync="channelsVisible" width="640px">
      <el-table :data="channels" size="small" border>
        <el-table-column prop="channelId" label="通道 ID" width="120" />
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="status" label="状态" width="80" />
        <el-table-column prop="ptzType" label="PTZ" width="70" />
        <el-table-column label="操作" width="100">
          <template slot-scope="{ row }">
            <el-button type="text" size="mini" @click="playChannel(row)">播放</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>

    <!-- 播放 -->
    <el-dialog :title="'播放 - ' + (playingChannel && playingChannel.name)" :visible.sync="playerVisible" width="820px" @closed="stopPlay">
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
  queryPlugins, queryIpcDevices, addIpcDevice, deleteIpcDevice,
  syncIpcDevice, queryIpcChannels, startIpcLive, stopIpcLive
} from '@/api/plugin'

export default {
  name: 'IpcPlugins',
  components: { jessibuca },
  data() {
    return {
      loading: false,
      submitting: false,
      plugins: [],
      devices: [],
      addVisible: false,
      addForm: { pluginId: '', protocol: '', name: '', params: {} },
      schemaProps: {},
      protocolOptions: [],
      channelsVisible: false,
      currentDevice: null,
      channels: [],
      playerVisible: false,
      playerDevice: null,
      playingChannel: null,
      streamInfo: null,
      videoUrl: ''
    }
  },
  computed: {
    onlinePlugins() {
      return this.plugins.filter(p => p.online)
    }
  },
  created() {
    this.loadAll()
  },
  beforeDestroy() {
    this.stopPlay()
  },
  methods: {
    loadAll() {
      this.loading = true
      Promise.all([queryPlugins(), queryIpcDevices()]).then(([p, d]) => {
        this.plugins = p.data || []
        this.devices = (d.data && d.data.list) || []
      }).finally(() => { this.loading = false })
    },
    formatTime(ms) {
      if (!ms) return '-'
      const d = new Date(ms)
      const pad = n => (n < 10 ? '0' + n : n)
      return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
    },
    openAdd() {
      this.addVisible = true
      if (this.onlinePlugins.length) {
        this.addForm.pluginId = this.onlinePlugins[0].id
        this.onPluginChange(this.addForm.pluginId)
      }
    },
    onPluginChange(id) {
      const plugin = this.plugins.find(p => p.id === id)
      this.protocolOptions = (plugin && plugin.protocols) || []
      this.addForm.protocol = this.protocolOptions[0] || ''
      this.schemaProps = {}
      this.addForm.params = {}
      if (plugin && plugin.configSchema) {
        try {
          const schema = JSON.parse(plugin.configSchema)
          this.schemaProps = schema.properties || {}
          Object.keys(this.schemaProps).forEach(key => {
            const def = this.schemaProps[key].default
            this.$set(this.addForm.params, key, def !== undefined ? def : '')
          })
        } catch (e) {
          console.warn('invalid config schema', e)
        }
      }
    },
    isNumber(f) {
      return f.type === 'integer' || f.type === 'number'
    },
    submitAdd() {
      if (!this.addForm.pluginId) {
        this.$message.warning('请选择插件')
        return
      }
      this.submitting = true
      addIpcDevice({
        pluginId: this.addForm.pluginId,
        protocol: this.addForm.protocol,
        name: this.addForm.name,
        params: this.addForm.params
      }).then(() => {
        this.$message.success('设备已添加')
        this.addVisible = false
        this.loadAll()
      }).finally(() => { this.submitting = false })
    },
    resetAdd() {
      this.addForm = { pluginId: '', protocol: '', name: '', params: {} }
      this.schemaProps = {}
    },
    removeDevice(row) {
      this.$confirm(`删除设备 ${row.name} ?`, '提示', { type: 'warning' }).then(() => {
        deleteIpcDevice(row.id).then(() => {
          this.$message.success('已删除')
          this.loadAll()
        })
      }).catch(() => {})
    },
    syncDevice(row) {
      syncIpcDevice(row.id).then(() => {
        this.$message.success('通道已同步')
        this.loadAll()
      })
    },
    openChannels(row) {
      this.currentDevice = row
      queryIpcChannels(row.id).then(res => {
        this.channels = (res.data && res.data.list) || []
        this.channelsVisible = true
      })
    },
    playChannel(channel) {
      if (!this.currentDevice) return
      this.playingChannel = channel
      startIpcLive(this.currentDevice.id, channel.channelId).then(res => {
        const content = res.data
        if (!content) {
          this.$message.error('未返回播放地址')
          return
        }
        this.streamInfo = content
        this.videoUrl = content.wsFlv || content.flv || ''
        this.playerVisible = true
      })
    },
    stopPlay() {
      if (this.currentDevice && this.playingChannel) {
        stopIpcLive(this.currentDevice.id, this.playingChannel.channelId).catch(() => {})
      }
      this.playerVisible = false
      this.videoUrl = ''
      this.streamInfo = null
      this.playingChannel = null
    }
  }
}
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.hint { margin-top: 8px; color: #909399; font-size: 12px; }
.field-desc { color: #909399; font-size: 12px; line-height: 18px; }
.stream-links { margin-top: 10px; color: #606266; font-size: 12px; word-break: break-all; }
</style>
