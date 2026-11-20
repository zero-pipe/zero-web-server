<template>
  <div class="app-container">
    <el-form :inline="true" size="mini">
      <el-form-item label="设备名称">
        <el-input v-model="filters.deviceName" clearable placeholder="设备名称" style="width:180px" @keyup.enter.native="reload" />
      </el-form-item>
      <el-form-item label="事件类型">
        <el-select v-model="filters.eventType" clearable filterable placeholder="全部" style="width:170px">
          <el-option v-for="t in eventTypes" :key="t.code" :label="t.name" :value="t.code" />
        </el-select>
      </el-form-item>
      <el-form-item label="关联状态">
        <el-select v-model="filters.status" clearable placeholder="全部" style="width:120px">
          <el-option label="已关联" value="mapped" />
          <el-option label="未关联" value="unmapped" />
        </el-select>
      </el-form-item>
      <el-form-item label="时间">
        <el-date-picker
          v-model="filters.range"
          type="datetimerange"
          range-separator="至"
          start-placeholder="开始"
          end-placeholder="结束"
          style="width:340px"
        />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" icon="el-icon-search" circle title="查询" aria-label="查询" @click="reload" />
        <el-button icon="el-icon-refresh" circle :loading="loading" @click="load" />
        <el-button type="text" style="color:#EA4335" @click="clearOld">清理 90 天前</el-button>
      </el-form-item>
    </el-form>

    <div class="stats-row">
      <el-tag size="mini" type="info">接收 {{ stats.received || 0 }}</el-tag>
      <el-tag size="mini" type="success">入库 {{ stats.stored || 0 }}</el-tag>
      <el-tag size="mini">重复 {{ stats.duplicate || 0 }}</el-tag>
      <el-tag size="mini" type="warning">未关联 {{ stats.unmapped || 0 }}</el-tag>
      <el-tag size="mini" type="danger">失败 {{ stats.failed || 0 }}</el-tag>
      <el-tag size="mini" type="danger">非法 {{ stats.invalid || 0 }}</el-tag>
    </div>

    <el-table v-loading="loading" :data="list" size="small" border height="calc(100vh - 220px)">
      <el-table-column label="时间" width="170">
        <template slot-scope="{ row }">{{ formatTime(row.occurredAt) }}</template>
      </el-table-column>
      <el-table-column label="设备" min-width="170">
        <template slot-scope="{ row }">
          <div>{{ row.deviceName || '未关联设备' }}</div>
        </template>
      </el-table-column>
      <el-table-column label="事件类型" width="170">
        <template slot-scope="{ row }">
          <div>{{ typeName(row.eventType) }}</div>
          <div class="sub">{{ categoryName(row.category) }}</div>
        </template>
      </el-table-column>
      <el-table-column label="通道" width="150">
        <template slot-scope="{ row }">{{ row.channelName || '—' }}</template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template slot-scope="{ row }">
          <el-tag size="mini" :type="row.status === 'mapped' ? 'success' : 'warning'">
            {{ row.status === 'mapped' ? '已关联' : '未关联' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="图片" width="100">
        <template slot-scope="{ row }">
          <el-image
            v-if="row.imageUrl"
            :src="row.imageUrl"
            :preview-src-list="[row.imageUrl]"
            fit="cover"
            style="width:64px;height:40px;border-radius:4px"
          />
          <span v-else class="sub">—</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="100" fixed="right">
        <template slot-scope="{ row }">
          <el-button type="text" size="mini" icon="el-icon-view" title="详情" aria-label="详情" @click="openDetail(row)" />
          <el-button type="text" size="mini" icon="el-icon-video-play" title="回放" aria-label="回放" :disabled="!row.playback || !row.playback.supported" @click="openPlayback(row)" />
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      style="text-align:right;margin-top:8px"
      layout="total, sizes, prev, pager, next"
      :total="total"
      :page-size="count"
      :current-page.sync="page"
      :page-sizes="[20, 50, 100]"
      @current-change="load"
      @size-change="onSize"
    />

    <el-dialog title="事件详情" :visible.sync="detailVisible" width="760px">
      <template v-if="detail">
        <el-descriptions :column="2" size="small" border>
          <el-descriptions-item label="事件类型">{{ typeName(detail.eventType) }}</el-descriptions-item>
          <el-descriptions-item label="分类">{{ categoryName(detail.category) }}</el-descriptions-item>
          <el-descriptions-item label="设备">{{ detail.deviceName || '未关联设备' }}</el-descriptions-item>
          <el-descriptions-item label="通道">{{ detail.channelName || '—' }}</el-descriptions-item>
          <el-descriptions-item label="时间">{{ formatTime(detail.occurredAt) }}</el-descriptions-item>
          <el-descriptions-item label="报警投影">{{ detail.alarmId ? '已生成报警' : '未生成报警' }}</el-descriptions-item>
          <el-descriptions-item label="回放">
            <el-button v-if="detail.playback && detail.playback.supported" type="text" size="mini" @click="openPlayback(detail)">查看回放</el-button>
            <span v-else class="sub">{{ (detail.playback && detail.playback.reason) || '不可用' }}</span>
          </el-descriptions-item>
        </el-descriptions>
        <div v-if="detail.imageUrl" class="section">
          <el-image :src="detail.imageUrl" :preview-src-list="[detail.imageUrl]" fit="contain" style="max-width:100%;max-height:360px" />
        </div>
        <el-collapse v-if="detail.bboxJson || detail.rawJson" class="section">
          <el-collapse-item title="高级信息（排障用）" name="advanced">
            <div v-if="detail.bboxJson">
              <div class="section-title">目标框数据</div>
              <pre class="json">{{ pretty(detail.bboxJson) }}</pre>
            </div>
            <div v-if="detail.rawJson">
              <div class="section-title">原始事件数据</div>
              <pre class="json">{{ pretty(detail.rawJson) }}</pre>
            </div>
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-dialog>

    <el-dialog :title="playbackTitle" :visible.sync="playbackDialogVisible" width="800px" :before-close="closePlayback" destroy-on-close>
      <div v-if="playbackLoading" style="text-align:center;padding:40px 0;">
        <i class="el-icon-loading" style="font-size:32px" />
        <div style="margin-top:10px;color:#606266">正在加载回放...</div>
      </div>
      <div v-else-if="playbackError" style="text-align:center;padding:40px 0;color:#EA4335">{{ playbackError }}</div>
      <div v-else-if="playbackStreamInfo" style="height:400px;">
        <playerTabs ref="playbackPlayer" :show-button="false" :showTab="true" :has-audio="true" />
      </div>
      <span slot="footer">
        <el-button size="mini" @click="closePlayback">关闭</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import { queryEvents, getEvent, getEventStats, getEventTypes, clearEvents } from '@/api/event'
import { playback as startPlayback, stopPlayback } from '@/api/commonChannel'
import playerTabs from '@/views/common/playerTabs.vue'

function formatDatetime(ts) {
  if (!ts) return null
  const d = new Date(ts)
  const pad = n => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ` +
    `${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

export default {
  name: 'EventCenter',
  components: { playerTabs },
  data() {
    return {
      loading: false,
      list: [],
      total: 0,
      page: 1,
      count: 20,
      filters: { deviceName: '', eventType: '', status: '', range: [] },
      detailVisible: false,
      detail: null,
      stats: {},
      eventTypes: [],
      playbackDialogVisible: false,
      playbackLoading: false,
      playbackError: null,
      playbackStreamInfo: null,
      playbackTitle: '事件回放',
      playbackChannelId: null
    }
  },
  created() { this.load(); this.loadStats(); this.loadTypes() },
  methods: {
    loadTypes() {
      getEventTypes().then(res => {
        this.eventTypes = ((res && res.data && res.data.list) || []).filter(t => t.enabled !== false)
      }).catch(() => {})
    },
    typeName(code) {
      const t = this.eventTypes.find(x => x.code === code)
      return t ? t.name : '其他'
    },
    categoryName(category) {
      return { alarm: '报警', status: '状态', operation: '操作', ai: 'AI', metadata: '数据' }[category] || '—'
    },
    reload() { this.page = 1; this.load() },
    onSize(n) { this.count = n; this.page = 1; this.load() },
    load() {
      this.loading = true
      const params = {
        page: this.page, count: this.count,
        deviceName: this.filters.deviceName,
        eventType: this.filters.eventType,
        status: this.filters.status
      }
      if (this.filters.range && this.filters.range.length === 2) {
        params.beginTime = new Date(this.filters.range[0]).getTime()
        params.endTime = new Date(this.filters.range[1]).getTime()
      }
      queryEvents(params).then(res => {
        const d = (res && res.data) || {}
        this.list = d.list || []
        this.total = d.total || 0
        this.loadStats()
      }).catch(e => this.$message.error(e || '查询失败'))
        .finally(() => { this.loading = false })
    },
    loadStats() {
      getEventStats().then(res => { this.stats = (res && res.data) || {} }).catch(() => {})
    },
    clearOld() {
      const before = Date.now() - 90 * 24 * 3600 * 1000
      this.$confirm('确认清理 90 天前的事件记录？', '提示', { type: 'warning' }).then(() => {
        clearEvents(before).then(res => {
          this.$message.success(`已清理 ${((res && res.data) || {}).deleted || 0} 条`)
          this.load(); this.loadStats()
        })
      }).catch(() => {})
    },
    openDetail(row) {
      getEvent(row.id).then(res => {
        this.detail = (res && res.data) || null
        this.detailVisible = true
      }).catch(e => this.$message.error(e || '加载失败'))
    },
    openPlayback(row) {
      const p = row && row.playback
      if (!p || !p.supported || !p.channelLegacyId) {
        this.$message.warning((p && p.reason) || '该事件暂不支持回放')
        return
      }
      this.playbackTitle = `事件回放 - ${this.typeName(row.eventType)} (${this.formatTime(row.occurredAt)})`
      this.playbackDialogVisible = true
      this.playbackLoading = true
      this.playbackError = null
      this.playbackStreamInfo = null
      this.playbackChannelId = p.channelLegacyId
      startPlayback({
        channelId: p.channelLegacyId,
        startTime: formatDatetime(p.startAt),
        endTime: formatDatetime(p.endAt)
      }).then(data => {
        this.playbackStreamInfo = data
        this.playbackLoading = false
        this.$nextTick(() => {
          if (this.$refs.playbackPlayer) this.$refs.playbackPlayer.setStreamInfo(data)
        })
      }).catch(err => {
        this.playbackLoading = false
        this.playbackError = (err && err.msg) ? err.msg : '回放请求失败，请检查通道是否有该时段录像'
      })
    },
    closePlayback() {
      if (this.$refs.playbackPlayer) this.$refs.playbackPlayer.stop()
      if (this.playbackStreamInfo && this.playbackChannelId) {
        stopPlayback({ channelId: this.playbackChannelId, stream: this.playbackStreamInfo.stream }).catch(() => {})
      }
      this.playbackDialogVisible = false
      this.playbackStreamInfo = null
      this.playbackError = null
      this.playbackChannelId = null
    },
    formatTime(ms) {
      if (!ms) return '-'
      const d = new Date(ms)
      const p = n => (n < 10 ? '0' + n : n)
      return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`
    },
    pretty(raw) {
      try { return JSON.stringify(JSON.parse(raw), null, 2) } catch (e) { return raw }
    }
  }
}
</script>

<style scoped>
.sub { color: #80868B; font-size: 12px; }
.stats-row { margin-bottom: 8px; display: flex; gap: 6px; flex-wrap: wrap; }
.section { margin-top: 14px; }
.section-title { font-weight: 600; margin-bottom: 6px; }
.json { background: #f5f7fa; padding: 10px; border-radius: 6px; max-height: 260px; overflow: auto; font-size: 12px; }
</style>
