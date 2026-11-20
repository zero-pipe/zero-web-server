<template>
  <div class="app-container">
    <el-card shadow="never">
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
          <el-button type="text" icon="el-icon-refresh" @click="loadAll">刷新</el-button>
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
        <el-table-column label="操作" width="220" fixed="right">
          <template slot-scope="{ row }">
            <el-button type="text" size="mini" @click="openInstall(row)">安装</el-button>
            <el-button type="text" size="mini" @click="showDetail(row)">详情</el-button>
            <el-button type="text" size="mini" style="color:#f56c6c" @click="removePackage(row)">删除</el-button>
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
        <el-table-column label="操作" width="110" fixed="right">
          <template slot-scope="{ row }">
            <el-button
              v-if="row.status === 'installed'"
              type="text"
              size="mini"
              style="color:#f56c6c"
              @click="doUninstall(row)"
            >卸载</el-button>
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
      <div v-if="!onlineAgents.length" class="hint" style="color:#e6a23c">暂无在线 Agent，请先启动 zero-agent 并完成注册。</div>
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
import { queryAgents } from '@/api/agent'

export default {
  name: 'PluginPackages',
  data() {
    return {
      loading: false,
      uploading: false,
      installLoading: false,
      installing: false,
      packages: [],
      installs: [],
      agents: [],
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
      this.loadAgents()
      this.loadInstalls()
    },
    loadPackages() {
      this.loading = true
      queryPluginPackages().then(res => {
        this.packages = (res.data && res.data.list) || []
      }).finally(() => { this.loading = false })
    },
    loadAgents() {
      queryAgents().then(res => {
        this.agents = (res.data && res.data.list) || []
      }).catch(() => { this.agents = [] })
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
.hint { font-size: 12px; color: #909399; margin-bottom: 10px; line-height: 1.6; }
.sub-text { color: #909399; font-size: 12px; }
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
