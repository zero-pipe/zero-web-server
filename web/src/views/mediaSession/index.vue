<template>
  <div class="app-container media-session">
    <div class="ms-toolbar">
      <el-input
        v-model="keyword"
        size="small"
        class="ms-search"
        placeholder="设备/通道/流ID 关键字"
        prefix-icon="el-icon-search"
        clearable
      />
      <el-select v-model="nodeFilter" size="small" class="ms-node" placeholder="全部节点" clearable>
        <el-option label="全部节点" value="" />
        <el-option v-for="n in mediaServerList" :key="n.id" :label="nodeLabel(n)" :value="n.id" />
      </el-select>
      <el-select v-model="sourceFilter" size="small" class="ms-source" placeholder="全部来源" clearable>
        <el-option label="全部来源" value="" />
        <el-option label="GB28181 推流" value="gb28181" />
        <el-option label="ONVIF 拉流" value="onvif" />
        <el-option label="插件" value="plugin" />
        <el-option label="手动代理/其他" value="manual" />
      </el-select>
      <div class="ms-spacer" />
      <el-button size="small" icon="el-icon-plus" type="primary" @click="openProxy">新建临时代理</el-button>
      <el-button size="small" icon="el-icon-refresh-right" :loading="loading" @click="load">刷新</el-button>
    </div>

    <el-table
      v-loading="loading"
      :data="filteredList"
      size="small"
      height="calc(100vh - 200px)"
      :row-key="(row) => row.mediaServerId + row.app + row.stream"
      :empty-text="emptyText"
    >
      <el-table-column label="设备 / 通道" min-width="220">
        <template slot-scope="scope">
          <div class="ms-name">{{ scope.row.deviceName || '—' }}</div>
          <div class="ms-sub">
            <span>{{ scope.row.channelName || scope.row.channelInternalCode || '—' }}</span>
            <span v-if="scope.row.deviceInternalCode" class="ms-code">{{ scope.row.deviceInternalCode }}</span>
          </div>
        </template>
      </el-table-column>
      <el-table-column label="来源" min-width="170">
        <template slot-scope="scope">
          <el-tag size="mini" :type="sourceTagType(scope.row.sourceType)" effect="plain">
            {{ scope.row.sourceLabel }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="流" min-width="230">
        <template slot-scope="scope">
          <div class="ms-stream">{{ scope.row.app }}/{{ scope.row.stream }}</div>
        </template>
      </el-table-column>
      <el-table-column label="编码" min-width="170">
        <template slot-scope="scope">
          <span v-if="scope.row.videoCodec" class="ms-mono">{{ scope.row.videoCodec }}</span>
          <span v-if="scope.row.width && scope.row.height" class="ms-sub-inline">
            {{ scope.row.width }}×{{ scope.row.height }}
          </span>
          <span v-if="scope.row.videoFps" class="ms-sub-inline">{{ scope.row.videoFps }}fps</span>
        </template>
      </el-table-column>
      <el-table-column label="码率" width="110">
        <template slot-scope="scope">{{ formatBitrate(scope.row.bytesSpeed) }}</template>
      </el-table-column>
      <el-table-column label="观看" width="80">
        <template slot-scope="scope">{{ scope.row.readerCount || 0 }}</template>
      </el-table-column>
      <el-table-column label="时长" width="110">
        <template slot-scope="scope">{{ formatDuration(scope.row.aliveSecond) }}</template>
      </el-table-column>
      <el-table-column label="节点" min-width="150">
        <template slot-scope="scope">
          <span class="ms-sub">{{ nodeName(scope.row.mediaServerId) }}</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="180" fixed="right">
        <template slot-scope="scope">
          <el-button type="text" size="mini" icon="el-icon-link" @click="openAddress(scope.row)">地址</el-button>
          <el-tooltip content="在媒体节点上关闭该路流；若仍有客户端观看或设备仍在推流，可能被自动重新拉起。" placement="top">
            <el-button
              type="text"
              size="mini"
              icon="el-icon-switch-button"
              class="ms-danger"
              @click="stop(scope.row)"
            >关闭</el-button>
          </el-tooltip>
        </template>
      </el-table-column>
    </el-table>

    <!-- 地址 / 直链（仅协议直链，不暴露分享页） -->
    <el-dialog :visible.sync="addrVisible" width="720px" append-to-body custom-class="ms-addr-dialog">
      <div slot="title" class="ms-dlg-title">
        <i class="el-icon-link" />
        <span>流地址</span>
      </div>
      <div v-if="addrRow" class="ms-addr-body">
        <div class="ms-addr-meta">
          <span class="ms-addr-name">{{ addrRow.deviceName || addrRow.stream }}</span>
          <span class="ms-addr-stream">{{ addrRow.app }}/{{ addrRow.stream }}</span>
        </div>
        <div v-if="!addrRow.addresses || !addrRow.addresses.length" class="ms-addr-empty">
          <i class="el-icon-warning-outline" /> 暂无可用地址（节点离线或流未就绪）
        </div>
        <div v-else class="ms-addr-list">
          <div v-for="a in addrRow.addresses" :key="a.protocol" class="ms-addr-item">
            <span class="ms-proto" :class="'proto-' + a.protocol">{{ a.label }}</span>
            <span class="ms-url" :title="a.url">{{ a.url }}</span>
            <el-button class="ms-copy" type="text" size="mini" icon="el-icon-document-copy" @click="copy(a.url)">复制</el-button>
          </div>
        </div>
        <div class="ms-addr-tip">直链可在播放器或第三方工具中直接拉流；地址含内网信息，请勿随意外泄。</div>
      </div>
    </el-dialog>

    <!-- 新建临时拉流代理 -->
    <el-dialog :visible.sync="proxyVisible" title="新建临时代理" width="520px" append-to-body>
      <el-form :model="proxyForm" label-width="90px" size="small">
        <el-form-item label="媒体节点">
          <el-select v-model="proxyForm.nodeId" placeholder="默认节点" clearable style="width: 100%">
            <el-option v-for="n in mediaServerList" :key="n.id" :label="nodeLabel(n)" :value="n.id" />
          </el-select>
        </el-form-item>
        <el-form-item label="应用名">
          <el-input v-model="proxyForm.app" placeholder="proxy（默认）" />
        </el-form-item>
        <el-form-item label="流ID">
          <el-input v-model="proxyForm.stream" placeholder="留空自动生成" />
        </el-form-item>
        <el-form-item label="拉流地址" required>
          <el-input v-model="proxyForm.url" placeholder="rtsp:// / rtmp:// / http-flv 源地址" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button size="small" @click="proxyVisible = false">取消</el-button>
        <el-button size="small" type="primary" :loading="proxySubmitting" @click="submitProxy">创建</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { queryMediaSessions, stopMediaSession, addMediaProxy } from '@/api/mediaSession'

export default {
  name: 'MediaSession',
  data() {
    return {
      loading: false,
      list: [],
      keyword: '',
      nodeFilter: '',
      sourceFilter: '',
      mediaServerList: [],
      updateLooper: 0,
      addrVisible: false,
      addrRow: null,
      proxyVisible: false,
      proxySubmitting: false,
      proxyForm: { nodeId: '', app: '', stream: '', url: '' }
    }
  },
  computed: {
    filteredList() {
      const kw = (this.keyword || '').trim().toLowerCase()
      return this.list.filter((r) => {
        if (this.nodeFilter && r.mediaServerId !== this.nodeFilter) return false
        if (this.sourceFilter && r.sourceType !== this.sourceFilter) return false
        if (!kw) return true
        return [r.deviceName, r.channelName, r.deviceInternalCode, r.channelInternalCode, r.stream]
          .filter(Boolean)
          .some((s) => String(s).toLowerCase().indexOf(kw) >= 0)
      })
    },
    emptyText() {
      return this.list.length ? '无匹配会话' : '暂无媒体会话（播放/推流后在此出现）'
    }
  },
  mounted() {
    this.getMediaServers()
    this.load()
    this.updateLooper = setInterval(this.load, 3000)
  },
  destroyed() {
    clearInterval(this.updateLooper)
  },
  methods: {
    getMediaServers() {
      this.$store.dispatch('server/getMediaServerList')
        .then((data) => { this.mediaServerList = data || [] })
        .catch(() => {})
    },
    load() {
      this.loading = true
      queryMediaSessions()
        .then((res) => {
          this.list = (res && res.data && res.data.list) || []
        })
        .catch(() => {})
        .finally(() => { this.loading = false })
    },
    openAddress(row) {
      this.addrRow = row
      this.addrVisible = true
    },
    copy(text) {
      if (!text) {
        this.$message.warning('暂无地址可复制')
        return
      }
      this.$copyText(text).then(
        () => this.$message.success('已复制到剪贴板'),
        () => {}
      )
    },
    stop(row) {
      this.$confirm(
        `在媒体节点上关闭流 ${row.app}/${row.stream}？若仍有客户端观看或设备仍在推流，可能被自动重新拉起。`,
        '关闭流',
        {
          confirmButtonText: '关闭',
          cancelButtonText: '取消',
          type: 'warning'
        }
      ).then(() => {
        stopMediaSession({ nodeId: row.mediaServerId, app: row.app, stream: row.stream })
          .then(() => {
            this.$message.success('已关闭')
            this.load()
          })
          .catch(() => {})
      }).catch(() => {})
    },
    openProxy() {
      this.proxyForm = { nodeId: '', app: '', stream: '', url: '' }
      this.proxyVisible = true
    },
    submitProxy() {
      if (!this.proxyForm.url) {
        this.$message.warning('请填写拉流地址')
        return
      }
      this.proxySubmitting = true
      addMediaProxy(this.proxyForm)
        .then(() => {
          this.$message.success('代理已创建')
          this.proxyVisible = false
          this.load()
        })
        .catch(() => {})
        .finally(() => { this.proxySubmitting = false })
    },
    nodeLabel(n) {
      return n.name ? `${n.name} (${n.id})` : n.id
    },
    nodeName(id) {
      const n = this.mediaServerList.find((x) => x.id === id)
      return n ? (n.name || n.id) : (id || '—')
    },
    sourceTagType(type) {
      switch (type) {
        case 'gb28181': return 'success'
        case 'onvif': return 'warning'
        case 'plugin': return ''
        default: return 'info'
      }
    },
    formatBitrate(bytes) {
      const b = Number(bytes) || 0
      if (b <= 0) return '—'
      const kbps = (b * 8) / 1024
      if (kbps >= 1024) return (kbps / 1024).toFixed(2) + ' Mbps'
      return kbps.toFixed(0) + ' kbps'
    },
    formatDuration(sec) {
      const s = Number(sec) || 0
      if (s <= 0) return '—'
      const h = Math.floor(s / 3600)
      const m = Math.floor((s % 3600) / 60)
      const ss = s % 60
      const pad = (v) => String(v).padStart(2, '0')
      return (h > 0 ? h + ':' : '') + pad(m) + ':' + pad(ss)
    }
  }
}
</script>

<style lang="scss" scoped>
.media-session {
  .ms-toolbar {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 12px;
  }
  .ms-search { width: 240px; }
  .ms-node { width: 200px; }
  .ms-source { width: 160px; }
  .ms-spacer { flex: 1; }

  .ms-name {
    font-size: 13px;
    color: #202124;
    font-weight: 500;
  }
  .ms-sub {
    font-size: 12px;
    color: #80868b;
    margin-top: 2px;
  }
  .ms-sub-inline {
    font-size: 12px;
    color: #80868b;
    margin-left: 8px;
  }
  .ms-code {
    font-family: Consolas, 'Courier New', monospace;
    font-size: 11px;
    color: #5f6368;
    margin-left: 8px;
  }
  .ms-stream,
  .ms-mono {
    font-family: Consolas, 'Courier New', monospace;
    font-size: 12px;
    color: #3c4043;
  }
  .ms-danger { color: #ea4335; }
}

/* 地址弹窗：append-to-body 后 DOM 挂在 body 下，
   因此这些样式必须是顶层选择器（不能嵌在 .media-session 内），否则 scoped 后代选择器失效。 */
.ms-dlg-title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 600;
  color: #202124;

  i { color: #1a73e8; font-size: 16px; }
}

.ms-addr-body { padding: 2px 2px 4px; }

.ms-addr-meta {
  display: flex;
  align-items: baseline;
  gap: 12px;
  padding: 0 2px 12px;
  border-bottom: 1px solid #f1f3f4;
  margin-bottom: 8px;
}
.ms-addr-name { font-size: 14px; font-weight: 600; color: #202124; }
.ms-addr-stream {
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
  color: #80868b;
}

.ms-addr-empty {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 16px 4px;
  color: #80868b;
  font-size: 13px;
}

.ms-addr-list {
  max-height: 52vh;
  overflow-y: auto;
  padding: 2px 0;
}

.ms-addr-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 8px 6px;
  border-radius: 6px;
  transition: background 0.15s ease;

  &:hover { background: #f8f9fa; }
}
.ms-addr-item:hover .ms-url { color: #174ea6; }

.ms-proto {
  flex: none;
  min-width: 76px;
  height: 24px;
  padding: 0 8px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
  background: #e8f0fe;
  color: #1a73e8;

  &.proto-flv, &.proto-ws_flv { background: #e6f4ea; color: #137333; }
  &.proto-hls { background: #fef7e0; color: #b06000; }
  &.proto-rtmp { background: #fce8e6; color: #c5221f; }
  &.proto-rtsp { background: #f3e8fd; color: #8430ce; }
  &.proto-rtc, &.proto-rtcs { background: #e8f0fe; color: #1a73e8; }
}

.ms-url {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
  color: #3c4043;
}

.ms-copy {
  flex: none;
  padding: 0 6px !important;
}

.ms-addr-tip {
  margin-top: 10px;
  padding: 8px 10px;
  border-radius: 6px;
  background: #f8f9fa;
  color: #80868b;
  font-size: 12px;
  line-height: 1.5;
}
</style>
