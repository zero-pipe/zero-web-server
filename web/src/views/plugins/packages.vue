<template>
  <div class="app-container">
    <el-card shadow="never">
      <div slot="header" class="card-header">
        <span>内置插件</span>
        <el-button type="text" icon="el-icon-refresh" title="刷新" aria-label="刷新" @click="loadBuiltins" />
      </div>
      <div class="hint">
        随 zero-agent 分发的内置插件，发现即默认启用。停用生命周期：<b>删除设备 → 禁用插件（停进程）→ 删除插件包</b>。
      </div>
      <el-table v-loading="builtinLoading" :data="builtins" size="small" border>
        <el-table-column label="Agent" min-width="140">
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
            <span class="sub-text">{{ row.pluginId }} · v{{ row.version }}</span>
          </template>
        </el-table-column>
        <el-table-column label="协议" min-width="120">
          <template slot-scope="{ row }">
            <el-tag v-for="p in row.protocols" :key="p" size="mini" style="margin-right:4px">{{ p }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="120">
          <template slot-scope="{ row }">
            <el-tag :type="stateType(row.actualState)" size="mini">{{ stateLabel(row.actualState) }}</el-tag>
            <el-tag v-if="row.desiredState === 'disabled'" type="warning" size="mini" style="margin-left:4px">已禁用</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="200" fixed="right">
          <template slot-scope="{ row }">
            <el-button
              v-if="row.desiredState !== 'enabled'"
              type="text" size="mini" icon="el-icon-check" title="启用" aria-label="启用"
              @click="setBuiltinState(row, 'enable')"
            />
            <el-button
              v-else
              type="text" size="mini" icon="el-icon-close" title="禁用（停止进程）" aria-label="禁用"
              @click="setBuiltinState(row, 'disable')"
            />
            <el-button
              type="text" size="mini" icon="el-icon-delete" style="color:#EA4335"
              :disabled="row.desiredState === 'enabled' || row.actualState === 'running'"
              :title="row.desiredState === 'enabled' || row.actualState === 'running' ? '请先禁用插件再删除' : '删除插件包'"
              aria-label="删除" @click="removeBuiltin(row)"
            />
          </template>
        </el-table-column>
      </el-table>
      <div v-if="!builtins.length" class="hint">未发现内置插件（随 zero-agent 分发到插件目录的插件会在此显示）。</div>
    </el-card>

    <el-card shadow="never" style="margin-top:16px">
      <div slot="header" class="card-header">
        <span>插件包</span>
        <div>
          <el-upload
            :show-file-list="false"
            :before-upload="beforeUpload"
            :http-request="doUpload"
            accept=".zwplugin,.zip"
            style="display:inline-block;margin-right:8px"
          >
            <el-button type="primary" size="small" icon="el-icon-upload" :loading="uploading">上传插件包</el-button>
          </el-upload>
          <el-button type="text" icon="el-icon-refresh" title="刷新" aria-label="刷新" @click="loadAll" />
        </div>
      </div>
      <div class="hint">
        插件包为 <code>.zwplugin</code>（zip），根目录需包含 <code>plugin.yaml</code>。上传后选择目标 Agent 安装，安装完成后在「插件接入」中启用插件、添加设备。
      </div>
      <el-table v-loading="loading" :data="packages" size="small" border>
        <el-table-column label="插件" min-width="180">
          <template slot-scope="{ row }">
            <div>{{ row.name || row.pluginId }}</div>
            <span class="sub-text">{{ row.pluginId }} · v{{ row.version }}</span>
          </template>
        </el-table-column>
        <el-table-column label="厂商" width="120">
          <template slot-scope="{ row }">{{ row.vendor || '-' }}</template>
        </el-table-column>
        <el-table-column label="协议" min-width="120">
          <template slot-scope="{ row }">
            <el-tag v-for="p in row.protocols" :key="p" size="mini" style="margin-right:4px">{{ p }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="能力" min-width="180">
          <template slot-scope="{ row }">
            <el-tag v-for="c in row.capabilities" :key="c" size="mini" type="info" style="margin-right:4px">{{ c }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="架构" width="150">
          <template slot-scope="{ row }">
            <el-tag v-for="a in row.architectures" :key="a" size="mini" type="info" style="margin-right:4px">{{ a }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="大小" width="100">
          <template slot-scope="{ row }">{{ formatSize(row.sizeBytes) }}</template>
        </el-table-column>
        <el-table-column label="状态" width="90">
          <template slot-scope="{ row }">
            <el-tag :type="row.status === 'valid' ? 'success' : 'danger'" size="mini">{{ row.status === 'valid' ? '有效' : '无效' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120" fixed="right">
          <template slot-scope="{ row }">
            <el-button type="text" size="mini" icon="el-icon-download" title="安装" aria-label="安装" @click="openInstall(row)" />
            <el-button type="text" size="mini" icon="el-icon-view" title="详情" aria-label="详情" @click="showDetail(row)" />
            <el-button type="text" size="mini" icon="el-icon-delete" style="color:#EA4335" title="删除" aria-label="删除" @click="removePackage(row)" />
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-card shadow="never" style="margin-top:16px">
      <div slot="header" class="card-header">
        <span>安装记录</span>
        <el-select v-model="filterAgent" size="small" clearable placeholder="全部 Agent" style="width:200px" @change="loadInstalls">
          <el-option v-for="a in agents" :key="a.id" :label="a.hostname || a.id" :value="a.id" />
        </el-select>
      </div>
      <el-table v-loading="installLoading" :data="installs" size="small" border>
        <el-table-column label="Agent" min-width="150">
          <template slot-scope="{ row }">{{ agentName(row.agentId) }}</template>
        </el-table-column>
        <el-table-column label="插件" min-width="160">
          <template slot-scope="{ row }">{{ row.pluginId }} · v{{ row.version }}</template>
        </el-table-column>
        <el-table-column label="状态" width="110">
          <template slot-scope="{ row }">
            <el-tag :type="installType(row.status)" size="mini">{{ installLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="错误" min-width="200">
          <template slot-scope="{ row }">
            <span class="sub-text">{{ row.lastError || '-' }}</span>
          </template>
        </el-table-column>
        <el-table-column label="更新时间" width="170">
          <template slot-scope="{ row }">{{ row.updateTime || '-' }}</template>
        </el-table-column>
        <el-table-column label="操作" width="80" fixed="right">
          <template slot-scope="{ row }">
            <el-button
              v-if="row.status === 'installed'"
              type="text"
              size="mini"
              icon="el-icon-delete"
              style="color:#EA4335"
              title="卸载"
              aria-label="卸载"
              @click="doUninstall(row)"
            />
          </template>
        </el-table-column>
      </el-table>
    </el-card>

    <el-dialog title="安装到 Agent" :visible.sync="installVisible" width="420px">
      <div class="install-line">
        <span>插件：</span>{{ installTarget.name || installTarget.pluginId }} · v{{ installTarget.version }}
      </div>
      <el-select v-model="installAgent" placeholder="选择在线 Agent" style="width:100%;margin-top:12px">
        <el-option
          v-for="a in onlineAgents"
          :key="a.id"
          :label="(a.hostname || a.id) + ' (' + a.ip + ')'"
          :value="a.id"
        />
      </el-select>
      <div v-if="!onlineAgents.length" class="hint" style="color:#FBBC05">暂无在线 Agent，请先启动 zero-agent 并完成注册。</div>
      <div slot="footer">
        <el-button @click="installVisible = false">取消</el-button>
        <el-button type="primary" :loading="installing" :disabled="!installAgent" @click="doInstall">安装</el-button>
      </div>
    </el-dialog>

    <el-dialog title="插件包详情" :visible.sync="detailVisible" width="720px">
      <pre class="detail-pre">{{ detailJson }}</pre>
    </el-dialog>
  </div>
</template>

<script>
import {
  queryPluginPackages, uploadPluginPackage, deletePluginPackage,
  installPluginPackage, queryPluginInstalls, uninstallPlugin
} from '@/api/pluginPackage'
import { queryAgents, queryAgentPlugins, setAgentPluginState, deleteAgentPlugin } from '@/api/agent'

export default {
  name: 'PluginPackages',
  data() {
    return {
      loading: false,
      uploading: false,
      installLoading: false,
      installing: false,
      builtinLoading: false,
      packages: [],
      installs: [],
      agents: [],
      builtins: [],
      filterAgent: '',
      installVisible: false,
      detailVisible: false,
      installTarget: {},
      installAgent: '',
      detailJson: ''
    }
  },
  computed: {
    onlineAgents() {
      return this.agents.filter(a => a.status === 'online')
    }
  },
  created() {
    this.loadAll()
  },
  methods: {
    loadAll() {
      this.loadPackages()
      this.loadBuiltins()
      this.loadInstalls()
    },
    loadPackages() {
      this.loading = true
      queryPluginPackages().then(res => {
        this.packages = (res.data && res.data.list) || []
      }).finally(() => { this.loading = false })
    },
    loadBuiltins() {
      this.builtinLoading = true
      queryAgents().then(res => {
        const agents = (res.data && res.data.list) || []
        this.agents = agents
        return Promise.all(agents.map(a => queryAgentPlugins(a.id).then(r => {
          const list = (r.data && r.data.list) || []
          return list
            .filter(p => p.source !== 'platform')
            .map(p => ({ ...p, agent: a, agentId: a.id }))
        }).catch(() => [])))
      }).then(rows => {
        this.builtins = rows.reduce((all, cur) => all.concat(cur), [])
      }).catch(() => { this.builtins = [] })
        .finally(() => { this.builtinLoading = false })
    },
    setBuiltinState(row, action) {
      const label = action === 'enable' ? '启用' : '禁用'
      setAgentPluginState(row.agentId, row.pluginId, action).then(() => {
        this.$message.success(label + '命令已下发')
        this.loadBuiltins()
      }).catch(e => this.$message.error(e || (label + '失败')))
    },
    removeBuiltin(row) {
      this.$confirm(`删除内置插件 ${row.pluginId}？将停止进程并移除插件包（须已禁用且无设备）。`, '提示', { type: 'warning' }).then(() => {
        deleteAgentPlugin(row.agentId, row.pluginId).then(() => {
          this.$message.success('已删除插件包')
          this.loadBuiltins()
        }).catch(e => this.$message.error(e || '删除失败'))
      }).catch(() => {})
    },
    stateLabel(state) {
      return { running: '运行中', starting: '启动中', stopped: '已停止', error: '异常', discovered: '已发现', offline: '离线' }[state] || state || '-'
    },
    stateType(state) {
      return state === 'running' ? 'success' : (state === 'error' ? 'danger' : 'info')
    },
    loadInstalls() {
      this.installLoading = true
      queryPluginInstalls(this.filterAgent || undefined).then(res => {
        this.installs = (res.data && res.data.list) || []
      }).finally(() => { this.installLoading = false })
    },
    beforeUpload(file) {
      const ok = /\.(zwplugin|zip)$/i.test(file.name)
      if (!ok) this.$message.warning('请选择 .zwplugin 或 .zip 文件')
      if (file.size > 500 * 1024 * 1024) {
        this.$message.warning('文件不能超过 500MB')
        return false
      }
      return ok
    },
    doUpload({ file }) {
      this.uploading = true
      uploadPluginPackage(file).then(res => {
        this.$message.success('上传成功：' + (res.data.pluginId || '') + ' v' + (res.data.version || ''))
        this.loadPackages()
      }).catch(e => this.$message.error(e || '上传失败'))
        .finally(() => { this.uploading = false })
    },
    openInstall(row) {
      this.installTarget = row
      this.installAgent = this.onlineAgents.length ? this.onlineAgents[0].id : ''
      this.installVisible = true
    },
    doInstall() {
      this.installing = true
      installPluginPackage(this.installTarget.id, this.installAgent).then(() => {
        this.$message.success('安装成功，可在「插件接入」启用插件')
        this.installVisible = false
        this.loadInstalls()
      }).catch(e => this.$message.error(e || '安装失败'))
        .finally(() => { this.installing = false })
    },
    removePackage(row) {
      this.$confirm(`删除插件包 ${row.pluginId} v${row.version}？`, '提示', { type: 'warning' }).then(() => {
        deletePluginPackage(row.id).then(() => {
          this.$message.success('已删除')
          this.loadPackages()
        }).catch(e => this.$message.error(e || '删除失败'))
      }).catch(() => {})
    },
    doUninstall(row) {
      this.$confirm(`卸载 ${row.pluginId} v${row.version}？已接入设备时需先删除设备。`, '提示', { type: 'warning' }).then(() => {
        uninstallPlugin(row.id).then(() => {
          this.$message.success('已卸载')
          this.loadInstalls()
        }).catch(e => this.$message.error(e || '卸载失败'))
      }).catch(() => {})
    },
    showDetail(row) {
      this.detailJson = row.manifestJson || JSON.stringify(row, null, 2)
      this.detailVisible = true
    },
    agentName(id) {
      const a = this.agents.find(x => x.id === id)
      return a ? (a.hostname || a.id) : id
    },
    installLabel(s) {
      return { pending: '待安装', downloading: '下载中', installed: '已安装', failed: '失败', uninstalled: '已卸载' }[s] || s || '-'
    },
    installType(s) {
      return s === 'installed' ? 'success' : (s === 'failed' ? 'danger' : 'info')
    },
    formatSize(bytes) {
      if (!bytes) return '-'
      if (bytes < 1024) return bytes + ' B'
      if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
      return (bytes / 1024 / 1024).toFixed(1) + ' MB'
    }
  }
}
</script>

<style scoped>
.card-header { display: flex; justify-content: space-between; align-items: center; }
.hint { font-size: 12px; color: #80868B; margin-bottom: 10px; line-height: 1.6; }
.sub-text { color: #80868B; font-size: 12px; }
.install-line { font-size: 14px; color: #303133; }
.detail-pre {
  max-height: 420px;
  overflow: auto;
  background: #f5f7fa;
  padding: 12px;
  font-size: 12px;
  line-height: 1.6;
  border-radius: 4px;
  white-space: pre-wrap;
  word-break: break-all;
}
</style>
