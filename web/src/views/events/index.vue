<template>
  <div class="app-container">
    <el-form :inline="true" size="mini">
      <el-form-item label="设备内码">
        <el-input v-model="filters.deviceInternalCode" clearable placeholder="设备内码" style="width:180px" @keyup.enter.native="reload" />
      </el-form-item>
      <el-form-item label="事件类型">
        <el-select v-model="filters.eventType" clearable filterable allow-create placeholder="全部" style="width:150px">
          <el-option label="motion" value="motion" />
          <el-option label="line_crossing" value="line_crossing" />
          <el-option label="intrusion" value="intrusion" />
          <el-option label="video_loss" value="video_loss" />
          <el-option label="tamper" value="tamper" />
          <el-option label="face" value="face" />
          <el-option label="loitering" value="loitering" />
          <el-option label="plate" value="plate" />
          <el-option label="audio_abnormal" value="audio_abnormal" />
          <el-option label="alarm_input" value="alarm_input" />
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
        <el-button type="primary" icon="el-icon-search" @click="reload">查询</el-button>
        <el-button icon="el-icon-refresh" circle :loading="loading" @click="load" />
        <el-button type="text" style="color:#f56c6c" @click="clearOld">清理 90 天前</el-button>
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
          <div class="sub">{{ row.deviceInternalCode }}</div>
        </template>
      </el-table-column>
      <el-table-column prop="eventType" label="事件类型" width="130" />
      <el-table-column prop="channelExternalKey" label="通道" width="90" />
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
      <el-table-column prop="eventId" label="事件 ID" min-width="180" show-overflow-tooltip />
      <el-table-column label="操作" width="90" fixed="right">
        <template slot-scope="{ row }">
          <el-button type="text" size="mini" @click="openDetail(row)">详情</el-button>
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
          <el-descriptions-item label="事件 ID">{{ detail.eventId }}</el-descriptions-item>
          <el-descriptions-item label="事件类型">{{ detail.eventType }}</el-descriptions-item>
          <el-descriptions-item label="设备">{{ detail.deviceName || '未关联' }}</el-descriptions-item>
          <el-descriptions-item label="设备内码">{{ detail.deviceInternalCode }}</el-descriptions-item>
          <el-descriptions-item label="通道">{{ detail.channelExternalKey || '—' }}</el-descriptions-item>
          <el-descriptions-item label="时间">{{ formatTime(detail.occurredAt) }}</el-descriptions-item>
          <el-descriptions-item label="关联状态">{{ detail.status }}</el-descriptions-item>
          <el-descriptions-item label="资源 ID">{{ detail.resourceId || '—' }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="detail.imageUrl" class="section">
          <el-image :src="detail.imageUrl" :preview-src-list="[detail.imageUrl]" fit="contain" style="max-width:100%;max-height:360px" />
        </div>
        <div v-if="detail.bboxJson" class="section">
          <div class="section-title">bbox</div>
          <pre class="json">{{ pretty(detail.bboxJson) }}</pre>
        </div>
        <div v-if="detail.rawJson" class="section">
          <div class="section-title">原始事件</div>
          <pre class="json">{{ pretty(detail.rawJson) }}</pre>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { queryEvents, getEvent, getEventStats, clearEvents } from '@/api/event'

export default {
  name: 'EventCenter',
  data() {
    return {
      loading: false,
      list: [],
      total: 0,
      page: 1,
      count: 20,
      filters: { deviceInternalCode: '', eventType: '', status: '', range: [] },
      detailVisible: false,
      detail: null,
      stats: {}
    }
  },
  created() { this.load(); this.loadStats() },
  methods: {
    reload() { this.page = 1; this.load() },
    onSize(n) { this.count = n; this.page = 1; this.load() },
    load() {
      this.loading = true
      const params = {
        page: this.page, count: this.count,
        deviceInternalCode: this.filters.deviceInternalCode,
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
.sub { color: #909399; font-size: 12px; }
.stats-row { margin-bottom: 8px; display: flex; gap: 6px; flex-wrap: wrap; }
.section { margin-top: 14px; }
.section-title { font-weight: 600; margin-bottom: 6px; }
.json { background: #f5f7fa; padding: 10px; border-radius: 6px; max-height: 260px; overflow: auto; font-size: 12px; }
</style>
