<template>
  <div class="app-container">
    <div>
    <el-card shadow="never">
      <div slot="header" class="card-header">
        <span>Agent 与插件</span>
        <el-button type="text" icon="el-icon-refresh" title="刷新" aria-label="刷新" @click="loadAll" />
      </div>
      <el-table v-loading="loading" :data="agentPlugins" size="small" border>
        <el-table-column label="Agent" min-width="200">
          <template slot-scope="{ row }">
            <div>{{ row.agent.hostname || row.agent.id }}</div>
            <el-tag :type="row.agent.status === 'online' ? 'success' : 'info'" size="mini">
              {{ row.agent.status === 'online' ? '在线' : '离线' }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="插件" min-width="240">
          <template slot-scope="{ row }">
            <div>{{ row.name || row.pluginId }}</div>
            <span class="sub-text">v{{ row.version }}</span>
          </template>
        </el-table-column>
        <el-table-column label="协议" min-width="160" align="center">
          <template slot-scope="{ row }">
            <el-tag v-for="p in row.protocols" :key="p" size="mini" style="margin-right:4px">{{ p }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="能力" min-width="340">
          <template slot-scope="{ row }">
            <el-tag v-for="c in row.capabilities" :key="c" size="mini" type="info" style="margin-right:4px">{{ c }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="120" align="center">
          <template slot-scope="{ row }">
            <el-tag :type="stateType(row.actualState)" size="mini">{{ stateLabel(row.actualState) }}</el-tag>
            <el-tag v-if="row.desiredState === 'disabled'" type="warning" size="mini" style="margin-left:4px">已禁用</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="最后上报" min-width="180">
          <template slot-scope="{ row }">{{ formatTime(row.lastSeen) }}</template>
        </el-table-column>
        <el-table-column label="操作" width="190" fixed="right" align="center">
          <template slot-scope="{ row }">
            <el-switch
              v-model="row._enabled"
              :disabled="row.agent.status !== 'online'"
              active-color="#1A73E8"
              @change="(val) => onPluginToggle(row, val)"
            />
            <el-divider direction="vertical" />
            <el-button
              type="text"
              size="medium"
              icon="el-icon-refresh-left"
              :disabled="row.agent.status !== 'online'"
              @click="setState(row, 'restart')"
            >重启</el-button>
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
      <div class="hint">
        插件设备已合并到「标准接入」统一展示与管理（名称/厂商/型号/协议/地址/状态/通道数/订阅(布防撤防)/统计(心跳)/操作）。
        <el-button type="text" size="small" icon="el-icon-right" @click="$router.push({ path: '/devices', query: { protocol: 'plugin' } })">前往标准接入查看</el-button>
      </div>
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
import { queryPlugins, addIpcDevice } from '@/api/plugin'
import { queryAgents, queryAgentPlugins, setAgentPluginState } from '@/api/agent'
import { getPipelineStatus } from '@/api/infra'

export default {
  name: 'IpcPlugins',
  data() {
    return {
      loading: false,
      submitting: false,
      plugins: [],
      agentPlugins: [],
      addVisible: false,
      addForm: { pluginId: '', protocol: '', name: '', params: {} },
      schemaProps: {},
      protocolOptions: [],
      pipelineReady: false,
      pipelineMsg: ''
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
  methods: {
    loadAll() {
      this.loading = true
      this.loadAgents()
      Promise.all([queryPlugins(), getPipelineStatus()]).then(([p, ps]) => {
        this.plugins = p.data || []
        this.applyPipeline((ps && ps.data) || {})
      }).finally(() => { this.loading = false })
    },
    loadAgents() {
      queryAgents().then(res => {
        const agents = (res.data && res.data.list) || []
        return Promise.all(agents.map(a => queryAgentPlugins(a.id).then(r => {
          const list = (r.data && r.data.list) || []
          return list.map(p => ({ ...p, agent: a, agentId: a.id, _enabled: p.desiredState === 'enabled' }))
        })))
      }).then(rows => {
        this.agentPlugins = rows.reduce((all, cur) => all.concat(cur), [])
      }).catch(() => { this.agentPlugins = [] })
    },
    onPluginToggle(row, val) {
      const action = val ? 'enable' : 'disable'
      setAgentPluginState(row.agentId, row.pluginId, action).then(() => {
        this.$message.success(val ? '已启用' : '已禁用')
        this.loadAll()
      }).catch(e => {
        row._enabled = !val
        this.$message.error((e && e.message) || (val ? '启用失败' : '禁用失败'))
      })
    },
    setState(row, action) {
      const label = action === 'restart' ? '重启' : ''
      setAgentPluginState(row.agentId, row.pluginId, action).then(() => {
        this.$message.success(label + '命令已下发')
        this.loadAll()
      }).catch(e => this.$message.error((e && e.message) || (label + '失败')))
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
    }
  }
}
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.hint { margin-top: 8px; color: #80868B; font-size: 12px; }
.sub-text { color: #80868B; font-size: 12px; }
.field-desc { color: #80868B; font-size: 12px; line-height: 18px; }
.stream-links { margin-top: 10px; color: #606266; font-size: 12px; word-break: break-all; }
</style>
