<template>
  <el-dialog
    v-el-drag-dialog
    title="添加设备"
    width="60%"
    top="2rem"
    :close-on-click-modal="false"
    :visible.sync="showDialog"
    :destroy-on-close="true"
    append-to-body
    @close="close"
  >
    <el-form :inline="true" size="mini">
      <el-form-item>
        <el-input
          v-model="query"
          clearable
          style="width: 14rem"
          placeholder="设备名称 / 编号"
          prefix-icon="el-icon-search"
          @keyup.enter.native="search"
          @clear="search"
        />
      </el-form-item>
      <el-form-item>
        <el-select v-model="protocol" clearable placeholder="全部协议" style="width: 9rem" @change="search">
          <el-option label="全部协议" value="" />
          <el-option label="GB28181" value="gb28181" />
          <el-option label="ONVIF" value="onvif" />
          <el-option label="插件" value="plugin" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button size="mini" icon="el-icon-refresh-right" circle title="刷新" @click="load" />
      </el-form-item>
    </el-form>

    <el-table
      ref="table"
      v-loading="loading"
      :data="list"
      size="small"
      height="360"
      style="width: 100%"
      header-row-class-name="table-header"
      @selection-change="onSelectionChange"
    >
      <el-table-column type="selection" width="50" />
      <el-table-column prop="name" label="设备名称" min-width="180" />
      <el-table-column label="协议" width="110">
        <template slot-scope="{ row }">
          <el-tag size="mini" effect="plain">{{ protocolLabel(row.protocol) }}</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="状态" width="90">
        <template slot-scope="{ row }">
          <el-tag size="mini" :type="row.status === 'online' ? 'success' : 'info'">
            {{ row.status === 'online' ? '在线' : '离线' }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="channelCount" label="通道数" width="90" align="center" />
    </el-table>

    <el-pagination
      style="margin-top: 12px; text-align: right"
      background
      :current-page.sync="page"
      :page-size="count"
      :total="total"
      layout="total, prev, pager, next"
      @current-change="load"
    />

    <div slot="footer">
      <el-button size="small" @click="showDialog = false">取消</el-button>
      <el-button size="small" type="primary" :disabled="selection.length === 0" @click="confirm">
        确定（{{ selection.length }}）
      </el-button>
    </div>
  </el-dialog>
</template>

<script>
import { listDevices } from '@/api/devices'

export default {
  name: 'DeviceSelectDialog',
  data() {
    return {
      showDialog: false,
      loading: false,
      list: [],
      total: 0,
      page: 1,
      count: 15,
      query: '',
      protocol: '',
      selection: [],
      callback: null
    }
  },
  methods: {
    openDialog(callback) {
      this.callback = typeof callback === 'function' ? callback : null
      this.query = ''
      this.protocol = ''
      this.page = 1
      this.selection = []
      this.showDialog = true
      this.load()
    },
    protocolLabel(p) {
      return { gb28181: 'GB28181', onvif: 'ONVIF', plugin: '插件' }[p] || (p || '-')
    },
    search() {
      this.page = 1
      this.load()
    },
    load() {
      this.loading = true
      listDevices({
        page: this.page,
        count: this.count,
        query: this.query,
        protocol: this.protocol || undefined
      }).then(res => {
        const data = (res && res.data) || {}
        this.list = data.list || []
        this.total = data.total || 0
      }).finally(() => { this.loading = false })
    },
    onSelectionChange(val) {
      this.selection = val
    },
    confirm() {
      if (this.callback) this.callback(this.selection.slice())
      this.showDialog = false
    },
    close() {
      this.selection = []
      this.callback = null
    }
  }
}
</script>
