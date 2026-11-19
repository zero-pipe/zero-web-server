<template>
  <div class="app-container">
    <!-- 设备列表 / 通道列表：与标准接入一致的页内下钻 -->
    <channel-panel
      v-if="channelCtx"
      embedded
      device-locked
      protocol="plugin"
      :device-internal-code="channelCtx.internalCode"
      :device-name="channelCtx.name"
      @back="channelCtx = null"
    />
    <div v-show="!channelCtx">
    <el-card shadow="never">
      <div slot="header" class="card-header">
        <span>Agent 与插件</span>
        <el-button type="text" icon="el-icon-refresh" @click="loadAll">刷新</el-button>
      </div>
      <el-table v-loading="loading" :data="agentPlugins" size="small" border>
        <el-table-column label="Agent" min-width="150">
          <template slot-scope="{ row }">
            <div>{{ row.agent.hostname || row.agent.id }}</div>
            <el-tag :type="row.agent.status === 'online' ? 'success' : 'info'" size="mini">
              {{ row.agent.status === 'online' ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="插件" min-width="160">
          <template slot-scope="{ row }">
            <div>{{ row.name || row.pluginId }}</div>
            <span class="sub-text">v{{ row.version }}</span>
          </template>
        </el-table-column>
        <el-table-column label="协议" min-width="130">
          <template slot-scope="{ row }">
            <el-tag v-for="p in row.protocols" :key="p" size="mini" style="margin-right:4px">{{ p }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="能力" min-width="160">
          <template slot-scope="{ row }">
            <el-tag v-for="c in row.capabilities" :key="c" size="mini" type="info" style="margin-right:4px">{{ c }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template slot-scope="{ row }">
            <el-tag :type="stateType(row.actualState)" size="mini">{{ stateLabel(row.actualState) }}</el-tag>
            <el-tag v-if="row.desiredState === 'disabled'" type="warning" size="mini" style="margin-left:4px">已禁用</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最后上报" width="170">
          <template slot-scope="{ row }">{{ formatTime(row.lastSeen) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="190" fixed="right">
          <template slot-scope="{ row }">
            <el-button v-if="row.desiredState !== 'enabled'" type="text" size="mini" @click="setState(row, 'enable')">启用</el-button>
            <el-button v-else type="text" size="mini" @click="setState(row, 'disable')">禁用</el-button>
            <el-button type="text" size="mini" @click="setState(row, 'restart')">重启</el-button>
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!agentPlugins.length" class="hint">未发现 Agent。请先在接入主机启动 zero-agent，插件目录放入 plugin.yaml 与可执行文件。</div>
      <div v-else class="hint">插件由 zero-agent 托管；禁用/重启后平台会自动同步插件与设备在线状态。</div>
    </el-card>

    <el-card shadow="never" style="margin-top:16px">
      <div slot="header" class="card-header">
        <span>插件设备</span>
        <el-button type="primary" size="small" icon="el-icon-plus" @click="openAdd" :disabled="!plugins.length">添加设备</el-button>
      </div>
      <el-table v-loading="loading" :data="devices" size="small" border>
        <el-table-column prop="name" label="名称" min-width="140" />
        <el-table-column prop="protocol" label="协议" width="140" />
        <el-table-column label="地址/注册信息" min-width="160">
          <template slot-scope="{ row }">{{ deviceAddress(row) }}</template>
        </el-table-column>
        <el-table-column prop="manufacturer" label="厂商" width="100" />
        <el-table-column prop="model" label="型号" width="120" />
        <el-table-column label="在线" width="80">
          <template slot-scope="{ row }">
            <el-tag :type="row.online ? 'success' : 'info'" size="mini">{{ row.online ? '在线' : '离线' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="channelCount" label="通道数" width="80" />
        <el-table-column label="布防" width="90">
          <template slot-scope="{ row }">
            <el-tag v-if="row._armed" size="mini" type="success">已布防</el-tag>
            <el-tag v-else size="mini" type="info">未布防</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="330">
          <template slot-scope="{ row }">
            <el-button type="text" size="mini" @click="goChannels(row)">通道</el-button>
            <el-button type="text" size="mini" @click="syncDevice(row)">同步</el-button>
            <el-button type="text" size="mini" :disabled="!row.online" @click="toggleArm(row)">
              {{ row._armed ? '撤防' : '布防' }}
            </el-button>
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
          <el-input v-model="addForm.name" placeholder="留空则使用 IP:端口" />
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
    </div>
  </div>
</template>

<script>
import {
  queryPlugins, queryIpcDevices, addIpcDevice, deleteIpcDevice,
  syncIpcDevice, armIpcDevice, disarmIpcDevice, getIpcArmStatus
} from '@/api/plugin'
import { queryAgents, queryAgentPlugins, setAgentPluginState } from '@/api/agent'
import { getPipelineStatus } from '@/api/infra'
import ChannelPanel from '@/views/channel/unified-panel'

export default {
  name: 'IpcPlugins',
  components: { ChannelPanel },
  data() {
    return {
      loading: false,
      submitting: false,
      plugins: [],
      agentPlugins: [],
      devices: [],
      addVisible: false,
      addForm: { pluginId: '', protocol: '', name: '', params: {} },
      schemaProps: {},
      protocolOptions: [],
      pipelineReady: false,
      pipelineMsg: '',
      armStatuses: {},
      channelCtx: null
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
      this.loadAgents()
      Promise.all([queryPlugins(), queryIpcDevices(), getPipelineStatus()]).then(([p, d, ps]) => {
        this.plugins = p.data || []
        this.devices = (d.data && d.data.list) || []
        this.applyPipeline((ps && ps.data) || {})
        this.loadArmStatuses()
      }).finally(() => { this.loading = false })
    },
    loadAgents() {
      queryAgents().then(res => {
        const agents = (res.data && res.data.list) || []
        return Promise.all(agents.map(a => queryAgentPlugins(a.id).then(r => {
          const list = (r.data && r.data.list) || []
          return list.map(p => ({ ...p, agent: a }))
        })))
      }).then(rows => {
        this.agentPlugins = rows.reduce((all, cur) => all.concat(cur), [])
      }).catch(() => { this.agentPlugins = [] })
    },
    setState(row, action) {
      const label = action === 'enable' ? '启用' : action === 'disable' ? '禁用' : '重启'
      setAgentPluginState(row.agentId, row.pluginId, action).then(() => {
        this.$message.success(label + '命令已下发')
        this.loadAll()
      }).catch(e => this.$message.error(e || (label + '失败')))
    },
    stateLabel(state) {
      return { running: '运行中', starting: '启动中', stopped: '已停止', error: '异常', discovered: '已发现', offline: '离线' }[state] || state || '-'
    },
    stateType(state) {
      return state === 'running' ? 'success' : (state === 'error' ? 'danger' : 'info')
    },
    applyPipeline(p) {
      this.pipelineReady = !!p.ready
      const missing = []
      if (!p.objectStore || !p.objectStore.enabled || !p.objectStore.healthy) missing.push('对象存储')
      if (!p.mqtt || !p.mqtt.enabled || !p.mqtt.connected) missing.push('MQTT')
      this.pipelineMsg = missing.length ? ('事件管道未就绪：请先配置 ' + missing.join(' / ')) : ''
    },
    loadArmStatuses() {
      this.devices.forEach(row => {
        this.$set(row, '_armed', false)
        getIpcArmStatus(row.id).then(res => {
          const d = (res && res.data) || {}
          const sub = d.subscription
          this.$set(row, '_armed', !!(sub && sub.armed))
          this.$set(row, '_armStatus', sub || null)
        }).catch(() => {})
      })
    },
    toggleArm(row) {
      if (row._armed) {
        this.$confirm(`确认对设备 ${row.name} 撤防？`, '提示', { type: 'warning' }).then(() => {
          disarmIpcDevice(row.id).then(() => {
            this.$message.success('已撤防')
            this.loadArmStatuses()
          })
        }).catch(() => {})
        return
      }
      if (!this.pipelineReady) {
        this.$message.warning(this.pipelineMsg || '事件管道未就绪，请先配置对象存储与 MQTT')
        return
      }
      armIpcDevice(row.id, ['motion', 'line_crossing', 'intrusion']).then(res => {
        const d = (res && res.data) || {}
        if (d.armed) this.$message.success('布防成功')
        this.loadArmStatuses()
      }).catch(e => this.$message.error(e || '布防失败'))
    },
    deviceAddress(row) {
      let params = {}
      try { params = JSON.parse(row.driverParams || '{}') } catch (e) { params = {} }
      if (params.ip) return params.port ? `${params.ip}:${params.port}` : params.ip
      const uid = row.deviceUid || ''
      const i = uid.lastIndexOf('-')
      if (i > 0 && /^\d+$/.test(uid.slice(i + 1))) return uid.slice(0, i) + ':' + uid.slice(i + 1)
      return uid || '—'
    },
    formatTime(ms) {      if (!ms) return '-'
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
    // 与标准接入一致的通道下钻：页内切换通道列表，返回时回到设备列表。
    goChannels(row) {
      this.channelCtx = { internalCode: row.internalCode, name: row.name }
    }
  }
}
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.hint { margin-top: 8px; color: #909399; font-size: 12px; }
.sub-text { color: #909399; font-size: 12px; }
.field-desc { color: #909399; font-size: 12px; line-height: 18px; }
.stream-links { margin-top: 10px; color: #606266; font-size: 12px; word-break: break-all; }
</style>
