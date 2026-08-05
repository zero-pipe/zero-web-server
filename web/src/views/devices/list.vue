<template>
  <div style="height: calc(100vh - 124px);">
    <el-form :inline="true" size="mini">
      <el-form-item label="搜索">
        <el-input
          v-model="query"
          style="width: 180px;"
          placeholder="名称 / 内码 / IP"
          prefix-icon="el-icon-search"
          clearable
          @input="reload"
        />
      </el-form-item>
      <el-form-item label="接入模式">
        <el-select v-model="accessMode" style="width: 110px;" clearable placeholder="全部" @change="reload">
          <el-option label="被动" value="passive" />
          <el-option label="主动" value="active" />
        </el-select>
      </el-form-item>
      <el-form-item label="协议">
        <el-select v-model="protocol" style="width: 130px;" clearable placeholder="全部" @change="reload">
          <el-option label="GB28181" value="gb28181" />
          <el-option label="ONVIF" value="onvif" />
        </el-select>
      </el-form-item>
      <el-form-item label="状态">
        <el-select v-model="status" style="width: 100px;" clearable placeholder="全部" @change="reload">
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
          <el-option label="待上线" value="pending" />
        </el-select>
      </el-form-item>
      <el-form-item>
        <el-button type="primary" icon="el-icon-plus" @click="wizardVisible = true">添加设备</el-button>
        <el-button icon="el-icon-info" @click="showInfo">接入信息</el-button>
        <el-button icon="el-icon-search" @click="handleDiscover">局域网发现</el-button>
      </el-form-item>
      <el-form-item style="float: right;">
        <el-button icon="el-icon-refresh-right" circle :loading="loading" @click="loadList" />
      </el-form-item>
    </el-form>

    <!-- 红框融合列：统一列 + 国标流传输/订阅/统计；无能力协议格子留空 -->
    <el-table v-loading="loading" size="small" :data="list" height="calc(100% - 96px)" header-row-class-name="table-header">
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column label="模式" width="80">
        <template v-slot:default="{ row }">
          <el-tag v-if="row.accessMode === 'passive'" size="mini" type="success">被动</el-tag>
          <el-tag v-else size="mini" type="warning">主动</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="协议" width="90">
        <template v-slot:default="{ row }">
          <el-tag v-if="row.protocol === 'gb28181'" size="mini">GB28181</el-tag>
          <el-tag v-else size="mini" style="color:#7c3aed;border-color:#ddd6fe;background:#f5f3ff">ONVIF</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="addressText" label="地址/注册信息" min-width="150" show-overflow-tooltip />
      <el-table-column prop="vendor" label="厂商" width="90" />
      <el-table-column label="流传输模式" min-width="150">
        <template v-slot:default="{ row }">
          <el-select
            v-if="hasCap(row, 'transport')"
            v-model="row.streamMode"
            size="mini"
            style="width: 120px"
            @change="transportChange(row)"
          >
            <el-option label="UDP" value="UDP" />
            <el-option label="TCP主动模式" value="TCP-ACTIVE" />
            <el-option label="TCP被动模式" value="TCP-PASSIVE" />
          </el-select>
          <span v-else class="cell-empty">—</span>
        </template>
      </el-table-column>
      <el-table-column prop="channelCount" label="通道数" width="80" />
      <el-table-column label="状态" width="90">
        <template v-slot:default="{ row }">
          <el-tag v-if="row.status === 'online'" size="mini" type="success">在线</el-tag>
          <el-tag v-else-if="row.status === 'pending'" size="mini" type="warning">待上线</el-tag>
          <el-tag v-else size="mini" type="info">离线</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="订阅" min-width="220">
        <template v-slot:default="{ row }">
          <template v-if="hasCap(row, 'subscribe')">
            <el-checkbox
              label="目录"
              :checked="row.subscribeCycleForCatalog > 0"
              @change="(e) => subscribeForCatalog(row.dbId, e)"
            />
            <el-checkbox
              label="位置"
              :checked="row.subscribeCycleForMobilePosition > 0"
              @change="(e) => subscribeForMobilePosition(row.dbId, e)"
            />
            <el-checkbox
              label="报警"
              :checked="row.subscribeCycleForAlarm > 0"
              @change="(e) => subscribeForAlarm(row.dbId, e)"
            />
          </template>
          <span v-else class="cell-empty">—</span>
        </template>
      </el-table-column>
      <el-table-column label="统计" min-width="140">
        <template v-slot:default="{ row }">
          <template v-if="hasCap(row, 'stats')">
            <el-button
              type="text"
              size="mini"
              :disabled="row.status !== 'online'"
              icon="iconfont-14 icon-xintiao"
              title="心跳时间统计"
              @click="getKeepaliveTimeStatistics(row.rawId)"
            >心跳</el-button>
            <el-button
              type="text"
              size="mini"
              :disabled="row.status !== 'online'"
              icon="iconfont-14 icon-register"
              title="注册时间统计"
              @click="getRegisterTimeStatistics(row.rawId)"
            >注册</el-button>
          </template>
          <span v-else class="cell-empty">—</span>
        </template>
      </el-table-column>
      <el-table-column label="操作" min-width="300" fixed="right">
        <template v-slot:default="{ row }">
          <!-- 国标操作 -->
          <template v-if="row.protocol === 'gb28181'">
            <el-button
              type="text"
              size="medium"
              icon="el-icon-refresh"
              :disabled="row.status !== 'online'"
              @click="refDevice(row)"
            >刷新</el-button>
            <el-divider direction="vertical" />
            <el-button type="text" size="medium" icon="el-icon-video-camera" @click="$emit('show-channel', row)">通道</el-button>
            <el-divider direction="vertical" />
            <el-button type="text" size="medium" icon="el-icon-edit" @click="openEdit(row)">编辑</el-button>
            <el-divider direction="vertical" />
            <el-dropdown @command="(cmd) => moreClick(cmd, row)">
              <el-button type="text" size="medium">操作<i class="el-icon-arrow-down el-icon--right" /></el-button>
              <el-dropdown-menu slot="dropdown">
                <el-dropdown-item command="delete" style="color:#f56c6c">删除</el-dropdown-item>
                <el-dropdown-item command="setGuard" :disabled="row.status !== 'online'">布防</el-dropdown-item>
                <el-dropdown-item command="resetGuard" :disabled="row.status !== 'online'">撤防</el-dropdown-item>
              </el-dropdown-menu>
            </el-dropdown>
          </template>
          <!-- ONVIF 操作 -->
          <template v-else-if="row.protocol === 'onvif'">
            <el-button type="text" size="medium" icon="el-icon-refresh" @click="handleSync(row)">同步</el-button>
            <el-divider direction="vertical" />
            <el-button type="text" size="medium" icon="el-icon-video-camera" @click="$emit('show-channel', row)">通道</el-button>
            <el-divider direction="vertical" />
            <el-button type="text" size="medium" icon="el-icon-edit" @click="openEdit(row)">编辑</el-button>
            <el-divider direction="vertical" />
            <el-button type="text" size="medium" style="color:#f56c6c" @click="handleDelete(row)">删除</el-button>
          </template>
        </template>
      </el-table-column>
    </el-table>

    <el-pagination
      style="text-align: right; margin-top: 8px;"
      layout="total, sizes, prev, pager, next"
      :total="total"
      :page-size="count"
      :current-page.sync="page"
      :page-sizes="[15, 25, 35, 50]"
      @current-change="loadList"
      @size-change="onSizeChange"
    />

    <add-wizard :visible.sync="wizardVisible" @success="loadList" />
    <edit-dialog :visible.sync="editVisible" :row="editRow" @success="loadList" />
    <config-info ref="configInfo" />
    <sync-channel-progress ref="syncChannelProgress" />
    <time-statistics ref="timeStatistics" />

    <el-dialog title="发现设备" :visible.sync="discoverVisible" width="640px">
      <el-table v-loading="discoverLoading" :data="discovered" size="small" max-height="360">
        <el-table-column prop="name" label="名称" />
        <el-table-column label="地址">
          <template v-slot:default="{ row }">{{ row.ip }}:{{ row.port }}</template>
        </el-table-column>
        <el-table-column prop="location" label="位置" />
        <el-table-column label="操作" width="100">
          <template v-slot:default="{ row }">
            <el-button type="text" @click="fillFromDiscover(row)">添加</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script>
import { listDevices, syncDevice } from '@/api/devices'
import { deleteDevice as deleteGBDevice } from '@/api/device'
import { deleteDevice as deleteOnvifDevice, discoverDevices } from '@/api/onvif'
import addWizard from './addWizard.vue'
import editDialog from './editDialog.vue'
import configInfo from '@/views/dialog/configInfo.vue'
import syncChannelProgress from '@/views/device/dialog/SyncChannelProgress.vue'
import timeStatistics from '@/views/device/dialog/timeStatistics.vue'

export default {
  name: 'DevicesList',
  components: { addWizard, editDialog, configInfo, syncChannelProgress, timeStatistics },
  data() {
    return {
      loading: false,
      list: [],
      total: 0,
      page: 1,
      count: 15,
      query: '',
      accessMode: '',
      protocol: '',
      status: '',
      wizardVisible: false,
      editVisible: false,
      editRow: null,
      discoverVisible: false,
      discoverLoading: false,
      discovered: [],
      looper: 0
    }
  },
  mounted() {
    this.loadList()
    this.looper = setInterval(this.loadList, 15000)
  },
  destroyed() {
    clearInterval(this.looper)
  },
  methods: {
    hasCap(row, cap) {
      const caps = (row && row.capabilities) || []
      return caps.indexOf(cap) >= 0
    },
    openEdit(row) {
      this.editRow = row
      this.editVisible = true
    },
    reload() {
      this.page = 1
      this.loadList()
    },
    onSizeChange(val) {
      this.count = val
      this.reload()
    },
    loadList() {
      this.loading = true
      listDevices({
        page: this.page,
        count: this.count,
        query: this.query,
        accessMode: this.accessMode || undefined,
        protocol: this.protocol || undefined,
        status: this.status || undefined
      }).then(res => {
        const data = (res && res.data) || {}
        this.list = data.list || []
        this.total = data.total || 0
      }).catch(err => {
        this.$message.error((err && err.message) || '加载失败')
      }).finally(() => {
        this.loading = false
      })
    },
    showInfo() {
      this.$store.dispatch('server/getSystemConfig').then(data => {
        this.$refs.configInfo.openDialog(data)
      })
    },
    handleDiscover() {
      this.discoverVisible = true
      this.discoverLoading = true
      discoverDevices(5).then(res => {
        this.discovered = (res && res.data) || []
      }).catch(err => {
        this.$message.error((err && err.message) || '发现失败')
      }).finally(() => {
        this.discoverLoading = false
      })
    },
    fillFromDiscover(row) {
      this.discoverVisible = false
      this.wizardVisible = true
      this.$nextTick(() => {
        this.$root.$emit('devices-prefill-onvif', {
          ip: row.ip,
          port: row.port || 80,
          name: row.name
        })
      })
    },
    transportChange(row) {
      this.$store.dispatch('device/updateDeviceTransport', [row.rawId, row.streamMode])
        .then(() => this.$message.success('流传输模式已更新'))
        .catch(err => this.$message.error(err.message || err || '更新失败'))
    },
    subscribeForCatalog(dbId, value) {
      this.$store.dispatch('device/subscribeCatalog', { id: dbId, cycle: value ? 60 : 0 })
        .then(() => this.$message.success(value ? '订阅成功' : '取消订阅成功'))
        .catch(err => this.$message.error(err.message || err))
    },
    subscribeForMobilePosition(dbId, value) {
      this.$store.dispatch('device/subscribeMobilePosition', {
        id: dbId,
        cycle: value ? 60 : 0,
        interval: value ? 5 : 0
      }).then(() => this.$message.success(value ? '订阅成功' : '取消订阅成功'))
        .catch(err => this.$message.error(err.message || err))
    },
    subscribeForAlarm(dbId, value) {
      this.$store.dispatch('device/subscribeForAlarm', { id: dbId, cycle: value ? 60 : 0 })
        .then(() => this.$message.success(value ? '订阅成功' : '取消订阅成功'))
        .catch(err => this.$message.error(err.message || err))
    },
    getKeepaliveTimeStatistics(deviceId) {
      this.$refs.timeStatistics.openDialog('心跳时间统计', 'device/getKeepaliveTimeStatistics', deviceId, 60)
    },
    getRegisterTimeStatistics(deviceId) {
      this.$refs.timeStatistics.openDialog('注册时间统计', 'device/getRegisterTimeStatistics', deviceId, 10)
    },
    refDevice(row) {
      this.$store.dispatch('device/sync', row.rawId).then(data => {
        if (data && data.errorMsg) {
          this.$message.error(data.errorMsg)
          return
        }
        this.$refs.syncChannelProgress.openDialog(row.rawId, () => this.loadList())
      }).catch(err => {
        this.$message.error(err.message || err || '刷新失败')
      }).finally(() => {
        this.loadList()
      })
    },
    handleSync(row) {
      syncDevice(row.id).then(() => {
        this.$message.success('已触发同步')
        this.loadList()
      }).catch(err => {
        this.$message.error(err.message || err || '同步失败')
      })
    },
    moreClick(command, row) {
      if (command === 'delete') this.handleDelete(row)
      else if (command === 'setGuard') this.setGuard(row)
      else if (command === 'resetGuard') this.resetGuard(row)
    },
    setGuard(row) {
      this.$store.dispatch('device/setGuard', row.rawId)
        .then(() => this.$message.success('布防成功'))
        .catch(err => this.$message.error(err.message || err))
    },
    resetGuard(row) {
      this.$store.dispatch('device/resetGuard', row.rawId)
        .then(() => this.$message.success('撤防成功'))
        .catch(err => this.$message.error(err.message || err))
    },
    handleDelete(row) {
      this.$confirm(`确定删除设备「${row.name}」？`, '提示', { type: 'warning' }).then(() => {
        // 走各协议已有删除接口，避免统一 DELETE + id 冒号的兼容问题
        if (row.protocol === 'gb28181') {
          return deleteGBDevice(row.rawId)
        }
        if (row.protocol === 'onvif') {
          return deleteOnvifDevice(row.rawId)
        }
        return Promise.reject(new Error('不支持的协议'))
      }).then(() => {
        this.$message.success('已删除')
        this.loadList()
      }).catch(err => {
        if (err === 'cancel' || err === 'close') return
        const msg = (err && err.message) || (typeof err === 'string' ? err : '') || '删除失败'
        // 请求拦截器已弹过业务错误时，避免再弹一次空/未知提示
        if (msg && msg !== 'Error' && msg !== '未知错误') {
          this.$message.error(msg)
        }
      })
    }
  }
}
</script>

<style scoped>
.cell-empty {
  color: #c0c4cc;
}
</style>
