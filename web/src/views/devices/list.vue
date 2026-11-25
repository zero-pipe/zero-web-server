<template>
  <div style="height: calc(100vh - 124px);">
    <el-form :inline="true" size="mini">
      <el-form-item label="搜索">
        <el-input
          v-model="query"
          style="width: 180px;"
          placeholder="名称 / IP"
          prefix-icon="el-icon-search"
          clearable
          @input="reload"
        />
      </el-form-item>
      <el-form-item label="协议">
        <el-select v-model="protocol" style="width: 150px;" clearable placeholder="全部" @change="reload">
          <el-option-group label="标准协议">
            <el-option label="GB28181" value="gb28181" />
            <el-option label="ONVIF" value="onvif" />
          </el-option-group>
          <el-option-group label="SDK 协议">
            <el-option label="SDK" value="plugin" />
          </el-option-group>
        </el-select>
      </el-form-item>
      <el-form-item label="状态">
        <el-select v-model="status" style="width: 100px;" clearable placeholder="全部" @change="reload">
          <el-option label="在线" value="online" />
          <el-option label="离线" value="offline" />
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

    <!-- 统一 10 列：三类接入完全对齐，新增协议只需在此渲染 -->
    <el-table v-loading="loading" size="small" :data="list" height="calc(100% - 96px)" header-row-class-name="table-header">
      <el-table-column prop="name" label="名称" min-width="140" />
      <el-table-column prop="vendor" label="厂商" width="100" />
      <el-table-column prop="model" label="型号" width="130" show-overflow-tooltip />
      <el-table-column label="协议" width="100">
        <template v-slot:default="{ row }">
          <span class="proto-text">
            <i class="proto-dot" :class="protoDotClass(row.protocol)" />{{ protoLabel(row.protocol) }}
          </span>
        </template>
      </el-table-column>
      <el-table-column prop="addressText" label="地址/注册信息" min-width="150" show-overflow-tooltip />
      <el-table-column label="状态" width="90">
        <template v-slot:default="{ row }">
          <el-tag v-if="row.status === 'online'" size="mini" type="success">在线</el-tag>
          <el-tag v-else size="mini" type="info">离线</el-tag>
        </template>
      </el-table-column>
      <el-table-column prop="channelCount" label="通道数" width="80" align="center" />
      <el-table-column label="订阅" min-width="230">
        <template v-slot:default="{ row }">
          <!-- 三类协议统一表现：勾选框 + 名称；勾选成功文字变蓝，取消成功恢复原色 -->
          <!-- 国标：目录/位置/报警 -->
          <template v-if="row.protocol === 'gb28181'">
            <el-checkbox
              class="sub-chk"
              :class="{ 'sub-on': row.subscribeCycleForCatalog > 0 }"
              label="目录"
              :checked="row.subscribeCycleForCatalog > 0"
              @change="(e) => subscribeForCatalog(row.dbId, e)"
            />
            <el-checkbox
              class="sub-chk"
              :class="{ 'sub-on': row.subscribeCycleForMobilePosition > 0 }"
              label="位置"
              :checked="row.subscribeCycleForMobilePosition > 0"
              @change="(e) => subscribeForMobilePosition(row.dbId, e)"
            />
            <el-checkbox
              class="sub-chk"
              :class="{ 'sub-on': row.subscribeCycleForAlarm > 0 }"
              label="报警"
              :checked="row.subscribeCycleForAlarm > 0"
              @change="(e) => subscribeForAlarm(row.dbId, e)"
            />
          </template>
          <!-- ONVIF：事件订阅（真开关，控制 PullPoint） -->
          <template v-else-if="row.protocol === 'onvif'">
            <el-checkbox
              class="sub-chk"
              :class="{ 'sub-on': row.eventSubscribe }"
              label="事件"
              :checked="row.eventSubscribe"
              :disabled="row.status !== 'online'"
              @change="(e) => toggleOnvifEvent(row, e)"
            />
          </template>
          <!-- 插件：事件（布防=订阅，撤防=取消） -->
          <template v-else-if="row.protocol === 'plugin'">
            <el-checkbox
              class="sub-chk"
              :class="{ 'sub-on': row._armed }"
              label="事件"
              :checked="row._armed"
              :disabled="row.status !== 'online'"
              @change="(e) => togglePluginEvent(row, e)"
            />
          </template>
          <span v-else class="cell-empty">—</span>
        </template>
      </el-table-column>
      <el-table-column label="统计" min-width="150">
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
              v-if="row.protocol === 'gb28181'"
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
            <el-button type="text" size="medium" style="color:#EA4335" @click="handleDelete(row)">删除</el-button>
          </template>
          <!-- ONVIF 操作 -->
          <template v-else-if="row.protocol === 'onvif'">
            <el-button type="text" size="medium" icon="el-icon-refresh" @click="handleSync(row)">同步</el-button>
            <el-divider direction="vertical" />
            <el-button type="text" size="medium" icon="el-icon-video-camera" @click="$emit('show-channel', row)">通道</el-button>
            <el-divider direction="vertical" />
            <el-button type="text" size="medium" icon="el-icon-edit" @click="openEdit(row)">编辑</el-button>
            <el-divider direction="vertical" />
            <el-button type="text" size="medium" style="color:#EA4335" @click="handleDelete(row)">删除</el-button>
          </template>
          <!-- 插件操作 -->
          <template v-else-if="row.protocol === 'plugin'">
            <el-button type="text" size="medium" icon="el-icon-refresh" @click="handleSync(row)">同步</el-button>
            <el-divider direction="vertical" />
            <el-button type="text" size="medium" icon="el-icon-video-camera" @click="$emit('show-channel', row)">通道</el-button>
            <el-divider direction="vertical" />
            <el-button type="text" size="medium" icon="el-icon-edit" @click="openEdit(row)">编辑</el-button>
            <el-divider direction="vertical" />
            <el-button type="text" size="medium" style="color:#EA4335" @click="handleDelete(row)">删除</el-button>
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
    <device-edit-drawer :visible.sync="editVisible" :row="editRow" @success="loadList" />
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
import { listDevices, syncDevice, deleteDevice as deleteUnifiedDevice } from '@/api/devices'
import { discoverDevices, setEventSubscribe } from '@/api/onvif'
import { armIpcDevice, disarmIpcDevice, getIpcArmStatus, syncIpcDevice } from '@/api/plugin'
import addWizard from './addWizard.vue'
import DeviceEditDrawer from './DeviceEditDrawer.vue'
import configInfo from '@/views/dialog/configInfo.vue'
import syncChannelProgress from '@/views/device/dialog/SyncChannelProgress.vue'
import timeStatistics from '@/views/device/dialog/timeStatistics.vue'

export default {
  name: 'DevicesList',
  components: { addWizard, DeviceEditDrawer, configInfo, syncChannelProgress, timeStatistics },
  data() {
    return {
      loading: false,
      list: [],
      total: 0,
      page: 1,
      count: 15,
      query: '',
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
    // 支持从「插件接入」页带 protocol=plugin 跳转过来直接看插件设备。
    if (this.$route && this.$route.query && this.$route.query.protocol) {
      this.protocol = this.$route.query.protocol
    }
    this.loadList()
    this.looper = setInterval(this.loadList, 15000)
  },
  destroyed() {
    clearInterval(this.looper)
  },
  methods: {
    protoLabel(p) {
      return { gb28181: 'GB28181', onvif: 'ONVIF', plugin: 'SDK' }[p] || (p || '—')
    },
    protoDotClass(p) {
      return p === 'gb28181' ? 'gb' : (p === 'onvif' ? 'onvif' : 'sdk')
    },
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
        protocol: this.protocol || undefined,
        status: this.status || undefined
      }).then(res => {
        const data = (res && res.data) || {}
        this.list = (data.list || []).map(row => ({ ...row, _armed: false, _armLoading: true }))
        this.total = data.total || 0
        this.loadArmStatuses()
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
      const call = row.protocol === 'plugin' ? syncIpcDevice(row.rawId) : syncDevice(row.id)
      call.then(() => {
        this.$message.success('已触发同步')
        this.loadList()
      }).catch(err => {
        this.$message.error(err.message || err || '同步失败')
      })
    },
    // 插件设备布防/撤防：与事件订阅等价（布防=订阅事件，撤防=取消）。
    loadArmStatuses() {
      this.list.filter(r => r.protocol === 'plugin').forEach(row => {
        getIpcArmStatus(row.rawId).then(res => {
          const sub = (res && res.data && res.data.subscription) || null
          this.$set(row, '_armed', !!(sub && sub.armed))
        }).catch(() => {}).finally(() => { this.$set(row, '_armLoading', false) })
      })
    },
    // ONVIF 事件订阅开关：真开关，立即启停 PullPoint。勾选成功字变蓝，取消成功恢复原色。
    toggleOnvifEvent(row, checked) {
      setEventSubscribe(row.rawId, checked).then(() => {
        this.$set(row, 'eventSubscribe', checked)
        this.$message.success(checked ? '已开启事件订阅' : '已关闭事件订阅')
      }).catch(e => {
        // 失败不改动勾选态（:checked 由 row.eventSubscribe 决定，无需回滚）
        this.$message.error((e && e.message) || (checked ? '开启失败' : '关闭失败'))
      })
    },
    // 插件设备事件：勾选=布防(订阅)，取消=撤防(取消订阅)；与国标/ONVIF 表现一致。
    togglePluginEvent(row, checked) {
      if (!checked) {
        // 取消勾选 → 撤防
        this.$confirm(`确认关闭设备「${row.name}」的事件订阅？`, '提示', { type: 'warning' }).then(() => {
          disarmIpcDevice(row.rawId).then(() => {
            this.$set(row, '_armed', false)
            this.$message.success('已关闭事件订阅')
          }).catch(e => {
            this.$message.error((e && e.message) || '取消失败')
          })
        }).catch(() => {})
        return
      }
      // 勾选 → 布防
      armIpcDevice(row.rawId, ['motion', 'line_crossing', 'intrusion']).then(res => {
        const d = (res && res.data) || {}
        if (d.armed === false) {
          this.$message.error('开启失败')
          return
        }
        this.$set(row, '_armed', true)
        this.$message.success('已开启事件订阅')
      }).catch(e => {
        this.$message.error((e && e.message) || '开启失败')
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
        // 统一删除：级联清理旧表通道 + 统一资产（设备/资源/能力/映射）
        return deleteUnifiedDevice(row.id)
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
/* 协议列：彩色圆点 + 文字 */
.proto-text {
  display: inline-flex;
  align-items: center;
  font-size: 13px;
  color: #303133;
}
.proto-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  margin-right: 7px;
  flex: 0 0 auto;
  background: #c0c4cc;
}
.proto-dot.gb { background: #1A73E8; }
.proto-dot.onvif { background: #7c3aed; }
.proto-dot.sdk { background: #e6a23c; }
/* 订阅勾选框：默认灰色文字，勾选成功后文字变蓝；取消成功后恢复默认色 */
.sub-chk {
  margin-right: 16px;
}
.sub-chk ::v-deep .el-checkbox__label {
  color: #606266;
  transition: color .2s;
}
.sub-on ::v-deep .el-checkbox__label {
  color: #1A73E8;
  font-weight: 600;
}
</style>
