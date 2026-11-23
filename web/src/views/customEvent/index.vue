<template>
  <div class="app-container">
    <el-form :inline="true" size="mini">
      <el-form-item label="设备内码">
        <el-input v-model="filters.deviceCode" clearable placeholder="D..." style="width:180px" @keyup.enter.native="reload" />
      </el-form-item>
      <el-form-item label="通道内码">
        <el-input v-model="filters.channelCode" clearable placeholder="C..." style="width:180px" @keyup.enter.native="reload" />
      </el-form-item>
      <el-form-item label="大类">
        <el-select v-model="filters.majorType" clearable filterable placeholder="全部" style="width:150px" @change="onMajorChange">
          <el-option v-for="m in majorTypes" :key="m" :label="m" :value="m" />
        </el-select>
      </el-form-item>
      <el-form-item label="小类">
        <el-select v-model="filters.minorType" clearable filterable placeholder="全部" style="width:170px">
          <el-option v-for="n in minorTypes" :key="n" :label="n" :value="n" />
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
      </el-form-item>
    </el-form>

    <el-alert
      type="info"
      :closable="false"
      show-icon
      style="margin-bottom:8px"
      title="自定义事件由设备驱动上报，平台不解析其语义；原始数据（raw）在详情中查看，供集成方自建业务。"
    />

    <el-table v-loading="loading" :data="list" size="small" border height="calc(100vh - 260px)">
      <el-table-column label="时间" width="170">
        <template slot-scope="{ row }">{{ formatTime(row.occurAt) }}</template>
      </el-table-column>
      <el-table-column label="大类" width="140">
        <template slot-scope="{ row }">{{ row.majorType || '—' }}</template>
      </el-table-column>
      <el-table-column label="小类" width="180">
        <template slot-scope="{ row }">{{ row.minorType || '—' }}</template>
      </el-table-column>
      <el-table-column label="设备内码" min-width="180">
        <template slot-scope="{ row }">{{ row.deviceInternalCode || '—' }}</template>
      </el-table-column>
      <el-table-column label="通道内码" width="180">
        <template slot-scope="{ row }">{{ row.channelInternalCode || '—' }}</template>
      </el-table-column>
      <el-table-column label="级别" width="90">
        <template slot-scope="{ row }">
          <el-tag size="mini" :type="levelTag(row.level)">{{ row.level || 'info' }}</el-tag>
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
      <el-table-column label="操作" width="90" fixed="right">
        <template slot-scope="{ row }">
          <el-button type="text" size="mini" icon="el-icon-view" title="详情" aria-label="详情" @click="openDetail(row)" />
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

    <el-dialog title="自定义事件详情" :visible.sync="detailVisible" width="760px">
      <template v-if="detail">
        <el-descriptions :column="2" size="small" border>
          <el-descriptions-item label="大类">{{ detail.majorType || '—' }}</el-descriptions-item>
          <el-descriptions-item label="小类">{{ detail.minorType || '—' }}</el-descriptions-item>
          <el-descriptions-item label="设备内码">{{ detail.deviceInternalCode || '—' }}</el-descriptions-item>
          <el-descriptions-item label="通道内码">{{ detail.channelInternalCode || '—' }}</el-descriptions-item>
          <el-descriptions-item label="时间">{{ formatTime(detail.occurAt) }}</el-descriptions-item>
          <el-descriptions-item label="级别">{{ detail.level || 'info' }}</el-descriptions-item>
          <el-descriptions-item label="事件ID" :span="2">{{ detail.eventId }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="detail.imageUrl" class="section">
          <el-image :src="detail.imageUrl" :preview-src-list="[detail.imageUrl]" fit="contain" style="max-width:100%;max-height:360px" />
        </div>
        <el-collapse v-if="detail.rawJson" class="section">
          <el-collapse-item title="原始数据 raw（驱动语义，平台不解析）" name="raw">
            <pre class="json">{{ pretty(detail.rawJson) }}</pre>
          </el-collapse-item>
        </el-collapse>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { queryCustomEvents, getCustomEvent, getCustomEventTypes } from '@/api/customEvent'

export default {
  name: 'CustomEventCenter',
  data() {
    return {
      loading: false,
      list: [],
      total: 0,
      page: 1,
      count: 20,
      filters: { deviceCode: '', channelCode: '', majorType: '', minorType: '', range: [] },
      types: [],
      detailVisible: false,
      detail: null
    }
  },
  computed: {
    majorTypes() {
      return Array.from(new Set(this.types.map(t => t.majorType).filter(Boolean)))
    },
    minorTypes() {
      const maj = this.filters.majorType
      return Array.from(new Set(
        this.types.filter(t => !maj || t.majorType === maj).map(t => t.minorType).filter(Boolean)
      ))
    }
  },
  created() { this.load(); this.loadTypes() },
  methods: {
    loadTypes() {
      getCustomEventTypes(this.filters.deviceCode).then(res => {
        this.types = (res && res.data) || []
      }).catch(() => {})
    },
    onMajorChange() { this.filters.minorType = '' },
    reload() { this.page = 1; this.load(); this.loadTypes() },
    onSize(n) { this.count = n; this.page = 1; this.load() },
    load() {
      this.loading = true
      const params = {
        page: this.page, count: this.count,
        deviceCode: this.filters.deviceCode,
        channelCode: this.filters.channelCode,
        majorType: this.filters.majorType,
        minorType: this.filters.minorType
      }
      if (this.filters.range && this.filters.range.length === 2) {
        params.begin = new Date(this.filters.range[0]).getTime()
        params.end = new Date(this.filters.range[1]).getTime()
      }
      queryCustomEvents(params).then(res => {
        const d = (res && res.data) || {}
        this.list = d.list || []
        this.total = d.total || 0
      }).catch(e => this.$message.error(e || '查询失败'))
        .finally(() => { this.loading = false })
    },
    openDetail(row) {
      getCustomEvent(row.id).then(res => {
        this.detail = (res && res.data) || null
        this.detailVisible = true
      }).catch(e => this.$message.error(e || '加载失败'))
    },
    levelTag(level) {
      return { info: '', success: 'success', warning: 'warning', major: 'danger', critical: 'danger' }[level] || ''
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
.section { margin-top: 14px; }
.json { background: #f5f7fa; padding: 10px; border-radius: 6px; max-height: 320px; overflow: auto; font-size: 12px; }
</style>
