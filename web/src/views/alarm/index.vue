<template>
  <div id="alarmManage" class="app-container">
    <div style="height: calc(100vh - 124px);">
      <el-form :inline="true" size="mini">
        <el-form-item label="开始时间">
          <el-date-picker
            v-model="beginTime"
            type="datetime"
            placeholder="开始时间"
            value-format="yyyy-MM-dd HH:mm:ss"
            style="width: 180px;"
            clearable
          />
        </el-form-item>
        <el-form-item label="结束时间">
          <el-date-picker
            v-model="endTime"
            type="datetime"
            placeholder="结束时间"
            value-format="yyyy-MM-dd HH:mm:ss"
            style="width: 180px;"
            clearable
          />
        </el-form-item>
        <el-form-item label="报警类型">
          <el-select
            v-model="selectedAlarmTypes"
            multiple
            collapse-tags
            placeholder="全部类型"
            style="width: 200px;"
            clearable
          >
            <el-option
              v-for="item in alarmTypeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item>
          <el-button size="mini" type="primary" icon="el-icon-search" @click="search">查询</el-button>
          <el-button size="mini" icon="el-icon-refresh-right" @click="getAlarmList()">刷新</el-button>
          <el-button size="mini" type="danger" icon="el-icon-delete" :disabled="selectedRows.length === 0" @click="deleteSelected">删除选中</el-button>
          <el-button size="mini" type="text" style="color:#EA4335" icon="el-icon-delete" @click="openDelete">删除记录</el-button>
        </el-form-item>
      </el-form>
      <el-table
        ref="alarmTable"
        size="small"
        :data="alarmList"
        height="calc(100% - 64px)"
        style="width: 100%"
        header-row-class-name="table-header"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="alarmType" label="报警类型" width="160">
          <template v-slot:default="scope">
            <el-tag size="mini" :type="getAlarmTypeTagType(scope.row.alarmType)">
              {{ scope.row.alarmCode ? eventTypeLabel(scope.row.alarmCode) : getAlarmTypeLabel(scope.row.alarmType) }}
            </el-tag>
          </template>
        </el-table-column>
        <el-table-column label="级别" width="80" align="center">
          <template v-slot:default="scope">
            <el-tag size="mini" :type="severityType(scope.row.severity)">{{ severityLabel(scope.row.severity) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="处置" width="90" align="center">
          <template v-slot:default="scope">
            <el-tag size="mini" :type="statusType(scope.row.status)">{{ statusLabel(scope.row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="快照" width="90" align="center">
          <template v-slot:default="scope">
            <el-image
              v-if="scope.row.snapPath"
              :src="getSnapUrl(scope.row.id)"
              :preview-src-list="[getSnapUrl(scope.row.id)]"
              fit="cover"
              style="width: 64px; height: 48px; cursor: pointer;"
              lazy
            >
              <div slot="error" style="width: 64px; height: 48px; line-height: 48px; text-align: center; color: #c0c4cc; font-size: 12px;">
                <i class="el-icon-picture-outline" />
              </div>
            </el-image>
            <span v-else style="color: #c0c4cc; font-size: 12px;">无</span>
          </template>
        </el-table-column>
        <el-table-column prop="description" label="报警描述" min-width="200" show-overflow-tooltip />
        <el-table-column prop="deviceName" label="设备" width="140" show-overflow-tooltip />
        <el-table-column prop="channelName" label="通道名称" width="140" show-overflow-tooltip />
        <el-table-column prop="longitude" label="经度" width="100" />
        <el-table-column prop="latitude" label="纬度" width="100" />
        <el-table-column label="报警时间" width="165">
          <template v-slot:default="scope">
            {{ formatTime(scope.row.alarmTime) }}
          </template>
        </el-table-column>
        <el-table-column label="操作" width="400" fixed="right" align="center">
          <template v-slot:default="scope">
            <el-button size="medium" icon="el-icon-view" type="text" @click="openDetail(scope.row)">详情</el-button>
            <el-divider direction="vertical" />
            <el-button size="medium" icon="el-icon-video-play" type="text" @click="openPlayback(scope.row)">回放</el-button>
            <template v-if="scope.row.status !== 'handled'">
              <el-divider direction="vertical" />
              <el-button size="medium" icon="el-icon-check" type="text" @click="setStatus(scope.row, 'handled')">确认</el-button>
            </template>
            <template v-if="scope.row.status !== 'ignored'">
              <el-divider direction="vertical" />
              <el-button size="medium" icon="el-icon-close" type="text" @click="setStatus(scope.row, 'ignored')">忽略</el-button>
            </template>
            <el-divider direction="vertical" />
            <el-button size="medium" icon="el-icon-delete" style="color: #EA4335" type="text" @click="deleteSingle(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-pagination
        style="text-align: right"
        :current-page="currentPage"
        :page-size="count"
        :page-sizes="[15, 25, 35, 50]"
        layout="total, sizes, prev, pager, next"
        :total="total"
        @size-change="handleSizeChange"
        @current-change="currentChange"
      />
    </div>

    <!-- 录像回放对话框 -->
    <el-dialog
      :title="playbackTitle"
      :visible.sync="playbackDialogVisible"
      width="800px"
      :before-close="closePlayback"
      destroy-on-close
    >
      <div v-if="playbackLoading" style="text-align: center; padding: 40px 0;">
        <i class="el-icon-loading" style="font-size: 32px;" />
        <div style="margin-top: 10px; color: #606266;">正在加载回放...</div>
      </div>
      <div v-else-if="playbackError" style="text-align: center; padding: 40px 0; color: #EA4335;">
        <i class="el-icon-warning-outline" style="font-size: 32px;" />
        <div style="margin-top: 10px;">{{ playbackError }}</div>
      </div>
      <div v-else-if="playbackStreamInfo" style="height: 400px;">
        <playerTabs
          ref="playbackPlayer"
          :show-button="false"
          :showTab="true"
          :has-audio="true"
        />
      </div>
      <span slot="footer" class="dialog-footer">
        <el-button size="mini" @click="closePlayback">关闭</el-button>
      </span>
    </el-dialog>

    <el-dialog title="报警详情" :visible.sync="detailVisible" width="720px">
      <template v-if="detail">
        <el-descriptions :column="2" size="small" border>
          <el-descriptions-item label="报警类型">{{ detail.alarmCode ? eventTypeLabel(detail.alarmCode) : getAlarmTypeLabel(detail.alarmType) }}</el-descriptions-item>
          <el-descriptions-item label="时间">{{ formatTime(detail.alarmTime) }}</el-descriptions-item>
          <el-descriptions-item label="设备">{{ detail.deviceName || '—' }}</el-descriptions-item>
          <el-descriptions-item label="通道">{{ detail.channelName || '—' }}</el-descriptions-item>
          <el-descriptions-item label="级别">{{ severityLabel(detail.severity) }}</el-descriptions-item>
          <el-descriptions-item label="处置">{{ statusLabel(detail.status) }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="detail.snapPath" style="margin-top:14px;text-align:center">
          <el-image :src="getSnapUrl(detail.id)" :preview-src-list="[getSnapUrl(detail.id)]" fit="contain" style="max-width:100%;max-height:340px" />
        </div>
        <el-collapse v-if="detail.bboxJson" style="margin-top:14px">
          <el-collapse-item title="高级信息（排障用）" name="advanced">
            <div class="section-title">目标框数据</div>
            <pre class="json">{{ pretty(detail.bboxJson) }}</pre>
          </el-collapse-item>
        </el-collapse>
        <div style="margin-top:14px">
          <el-button size="small" type="primary" :disabled="!detail.channelId" @click="openPlayback(detail)">查看回放</el-button>
        </div>
      </template>
    </el-dialog>

    <el-dialog title="删除报警记录" :visible.sync="deleteVisible" width="460px">
      <div class="delete-tip">删除后不可恢复，请谨慎操作。</div>
      <el-form label-width="90px" size="small">
        <el-form-item label="删除范围">
          <el-radio-group v-model="deleteAll" @change="onDeleteAllChange">
            <el-radio :label="false">按时间范围</el-radio>
            <el-radio :label="true">全部删除</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="!deleteAll" label="时间范围">
          <el-date-picker
            v-model="deleteRange"
            type="datetimerange"
            range-separator="至"
            start-placeholder="开始时间"
            end-placeholder="结束时间"
            value-format="timestamp"
            style="width:100%"
          />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button size="small" @click="deleteVisible = false">取消</el-button>
        <el-button size="small" type="danger" :loading="deleting" @click="confirmDelete">删除</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import playerTabs from '../common/playerTabs.vue'
import { setAlarmStatus } from '@/api/alarm'

const ALARM_TYPE_OPTIONS = [
  { value: 'VideoLoss', label: '视频丢失报警' },
  { value: 'DeviceTamper', label: '设备防拆报警' },
  { value: 'StorageFull', label: '存储设备磁盘满报警' },
  { value: 'DeviceHighTemperature', label: '设备高温报警' },
  { value: 'DeviceLowTemperature', label: '设备低温报警' },
  { value: 'ManualVideo', label: '人工视频报警' },
  { value: 'MotionDetection', label: '运动目标检测报警' },
  { value: 'LeftObjectDetection', label: '遗留物检测报警' },
  { value: 'ObjectRemovalDetection', label: '物体移除检测报警' },
  { value: 'TripwireDetection', label: '绊线检测报警' },
  { value: 'IntrusionDetection', label: '入侵检测报警' },
  { value: 'MobileDetection', label: '移动侦测报警' },
  { value: 'VideoOcclusion', label: '视频遮挡报警' },
  { value: 'ReverseDetection', label: '逆行检测报警' },
  { value: 'LoiteringDetection', label: '徘徊检测报警' },
  { value: 'FlowStatistics', label: '流量统计报警' },
  { value: 'DensityDetection', label: '密度检测报警' },
  { value: 'VideoAbnormal', label: '视频异常检测报警' },
  { value: 'RapidMovement', label: '快速移动报警' },
  { value: 'StorageFault', label: '存储设备磁盘故障报警' },
  { value: 'StorageFanFault', label: '存储设备风扇故障报警' },
  { value: 'SoundAbnormal', label: '声音异常报警' },
  { value: 'SignalAbnormal', label: '信号量异常报警' },
  { value: 'IllegalAccess', label: '非法访问报警' },
  { value: 'Defocus', label: '虚焦报警' },
  { value: 'SceneChange', label: '场景变更报警' },
  { value: 'CrowdGathering', label: '人员聚集报警' },
  { value: 'ParkingDetection', label: '停车侦测报警' },
  { value: 'Other', label: '其他报警' }
]

const EVENT_ALARM_TYPES = {
  1: '视频丢失报警', 2: '设备防拆报警', 3: '存储设备磁盘满报警',
  4: '设备高温报警', 5: '设备低温报警', 6: '人工视频报警',
  7: '运动目标检测报警', 8: '遗留物检测报警', 9: '物体移除检测报警',
  10: '绊线检测报警', 11: '入侵检测报警', 12: '移动侦测报警',
  13: '视频遮挡报警', 14: '逆行检测报警', 15: '徘徊检测报警',
  16: '流量统计报警', 17: '密度检测报警', 18: '视频异常检测报警',
  19: '快速移动报警', 20: '存储设备磁盘故障报警', 21: '存储设备风扇故障报警',
  22: '声音异常报警', 23: '信号量异常报警', 24: '非法访问报警',
  25: '虚焦报警', 26: '场景变更报警', 27: '人员聚集报警',
  28: '停车侦测报警', 99: '其他报警'
}

function formatDatetime(ts) {
  if (!ts) return null
  const date = new Date(ts)
  const pad = n => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ` +
         `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

export default {
  name: 'AlarmManage',
  components: { playerTabs },
  data() {
    return {
      alarmList: [],
      beginTime: null,
      endTime: null,
      selectedAlarmTypes: [],
      alarmTypeOptions: ALARM_TYPE_OPTIONS,
      currentPage: 1,
      count: 15,
      total: 0,
      selectedRows: [],
      // 回放相关
      playbackDialogVisible: false,
      playbackLoading: false,
      playbackError: null,
      playbackStreamInfo: null,
      playbackTitle: '录像回放',
      currentPlaybackChannelId: null,
      detailVisible: false,
      detail: null,
      deleteVisible: false,
      deleteAll: false,
      deleteRange: [],
      deleting: false
    }
  },
  created() {
    this.getAlarmList()
  },
  methods: {
    currentChange(val) {
      this.currentPage = val
      this.getAlarmList()
    },
    handleSizeChange(val) {
      this.count = val
      this.getAlarmList()
    },
    handleSelectionChange(rows) {
      this.selectedRows = rows
    },
    search() {
      this.currentPage = 1
      this.total = 0
      this.getAlarmList()
    },
    getAlarmList() {
      this.$store.dispatch('alarm/getAlarmList', {
        page: this.currentPage,
        count: this.count,
        alarmType: this.selectedAlarmTypes.length > 0 ? this.selectedAlarmTypes : undefined,
        beginTime: this.beginTime || undefined,
        endTime: this.endTime || undefined
      }).then(data => {
        this.total = data.total
        this.alarmList = data.list
        this.$nextTick(() => {
          this.$refs.alarmTable.doLayout()
        })
      }).catch(error => {
        console.log(error)
      })
    },
    getSnapUrl(id) {
      const baseUrl = window.baseUrl ? window.baseUrl : ''
      return ((process.env.NODE_ENV === 'development') ? process.env.VUE_APP_BASE_API : baseUrl) + `/api/v1/alarm/snap/${id}`
    },
    openDetail(row) {
      this.detail = row
      this.detailVisible = true
    },
    setStatus(row, status) {
      const label = status === 'handled' ? '确认' : '忽略'
      this.$confirm(`确认将该报警标记为${label}？`, '提示', { type: 'warning' }).then(() => {
        setAlarmStatus(row.id, { status, handledBy: 'admin' }).then(() => {
          this.$message.success('已更新')
          this.getAlarmList()
        }).catch(e => this.$message.error(e || '更新失败'))
      }).catch(() => {})
    },
    eventTypeLabel(code) {
      return {
        motion: '移动侦测', line_crossing: '越界侦测', intrusion: '区域入侵', loitering: '徘徊',
        video_loss: '视频丢失', tamper: '遮挡/破坏', audio_abnormal: '音频异常', scene_change: '场景变更',
        face: '人脸', plate: '车牌', alarm_input: '报警输入', other: '其他'
      }[code] || '其他'
    },
    statusLabel(status) {
      return { unhandled: '未处理', handled: '已确认', ignored: '已忽略' }[status] || (status || '未处理')
    },
    statusType(status) {
      return status === 'handled' ? 'success' : (status === 'ignored' ? 'info' : 'warning')
    },
    severityType(severity) {
      return severity === 'critical' ? 'danger' : (severity === 'major' ? 'warning' : 'info')
    },
    severityLabel(severity) {
      return { critical: '严重', major: '重要', warning: '一般', info: '提示' }[severity] || '提示'
    },
    pretty(raw) {
      try { return JSON.stringify(JSON.parse(raw), null, 2) } catch (e) { return raw }
    },
    openPlayback(row) {
      if (!row.channelId) {
        this.$message({ showClose: true, message: '该报警无关联通道，无法回放', type: 'warning' })
        return
      }
      if (row.playback && row.playback.supported === false) {
        this.$message({ showClose: true, message: row.playback.reason || '该通道暂不支持回放', type: 'warning' })
        return
      }
      this.playbackTitle = `录像回放 - ${row.channelName || row.channelDeviceId} (${this.formatTime(row.alarmTime)})`
      this.playbackDialogVisible = true
      this.playbackLoading = true
      this.playbackError = null
      this.playbackStreamInfo = null
      this.playbackVideoUrl = null
      this.currentPlaybackChannelId = (row.playback && row.playback.channelLegacyId) || row.channelId

      // 优先使用后端推导的回放窗口，否则报警时间前后 10 秒。
      const startTime = row.playback && row.playback.startAt
        ? formatDatetime(row.playback.startAt)
        : formatDatetime(row.alarmTime - 10 * 1000)
      const endTime = row.playback && row.playback.endAt
        ? formatDatetime(row.playback.endAt)
        : formatDatetime(row.alarmTime + 10 * 1000)

      this.$store.dispatch('commonChanel/playback', {
        channelId: this.currentPlaybackChannelId,
        startTime: startTime,
        endTime: endTime
      }).then(data => {
        this.playbackStreamInfo = data
        this.playbackLoading = false
        this.$nextTick(() => {
          if (this.$refs.playbackPlayer) {
            this.$refs.playbackPlayer.setStreamInfo(data)
          }
        })
      }).catch(err => {
        this.playbackLoading = false
        this.playbackError = (err && err.msg) ? err.msg : '回放请求失败，请检查通道是否有该时段录像'
        console.log(err)
      })
    },
    closePlayback() {
      if (this.$refs.playbackPlayer) {
        this.$refs.playbackPlayer.stop()
      }
      if (this.playbackStreamInfo && this.currentPlaybackChannelId) {
        this.$store.dispatch('commonChanel/stopPlayback', {
          channelId: this.currentPlaybackChannelId,
          stream: this.playbackStreamInfo.stream
        }).catch(err => {
          console.log(err)
        })
      }
      this.playbackDialogVisible = false
      this.playbackStreamInfo = null
      this.playbackError = null
      this.currentPlaybackChannelId = null
    },
    deleteSingle(row) {
      this.$confirm('确定删除该报警记录?', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        this.$store.dispatch('alarm/deleteAlarms', [row.id])
          .then(() => {
            this.$message({ showClose: true, message: '删除成功', type: 'success' })
            this.getAlarmList()
          })
          .catch(error => {
            this.$message({ showClose: true, message: error, type: 'error' })
          })
      }).catch(() => {})
    },
    deleteSelected() {
      this.$confirm(`确定删除选中的 ${this.selectedRows.length} 条报警记录?`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        const ids = this.selectedRows.map(r => r.id)
        this.$store.dispatch('alarm/deleteAlarms', ids)
          .then(() => {
            this.$message({ showClose: true, message: '删除成功', type: 'success' })
            this.getAlarmList()
          })
          .catch(error => {
            this.$message({ showClose: true, message: error, type: 'error' })
          })
      }).catch(() => {})
    },
    openDelete() {
      this.deleteAll = false
      this.deleteRange = []
      this.deleteVisible = true
    },
    onDeleteAllChange() {
      if (this.deleteAll) this.deleteRange = []
    },
    confirmDelete() {
      const all = this.deleteAll
      if (!all && (!this.deleteRange || this.deleteRange.length !== 2)) {
        this.$message.warning('请选择时间范围，或选择全部删除')
        return
      }
      const beginTime = all ? undefined : this.deleteRange[0]
      const endTime = all ? undefined : this.deleteRange[1]
      const desc = all ? '全部报警记录' : `${this.formatTime(beginTime)} 至 ${this.formatTime(endTime)}`
      this.$confirm(`确认删除${desc}？此操作不可恢复。`, '删除报警', {
        confirmButtonText: '确定删除',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        this.deleting = true
        this.$store.dispatch('alarm/clearAlarms', { beginTime, endTime }).then(count => {
          this.$message({ showClose: true, message: (count === undefined || count === null) ? '删除成功' : `已删除 ${count} 条报警记录`, type: 'success' })
          this.deleteVisible = false
          this.currentPage = 1
          this.getAlarmList()
        }).catch(error => {
          this.$message({ showClose: true, message: error || '删除失败', type: 'error' })
        }).finally(() => { this.deleting = false })
      }).catch(() => {})
    },
    getAlarmTypeLabel(value) {
      if (typeof value === 'number') {
        return EVENT_ALARM_TYPES[value] || `类型 ${value}`
      }
      const option = ALARM_TYPE_OPTIONS.find(o => o.value === value)
      return option ? option.label : value
    },
    getAlarmTypeTagType(value) {
      if (typeof value === 'number') {
        if ([1, 2, 10, 11, 24].includes(value)) return 'danger'
        if ([7, 12, 14, 27, 19].includes(value)) return 'warning'
        return 'info'
      }
      const dangerTypes = ['VideoLoss', 'IntrusionDetection', 'IllegalAccess', 'ManualVideo', 'TripwireDetection']
      const warningTypes = ['MotionDetection', 'MobileDetection', 'ReverseDetection', 'CrowdGathering', 'RapidMovement']
      if (dangerTypes.includes(value)) return 'danger'
      if (warningTypes.includes(value)) return 'warning'
      return 'info'
    },
    formatTime(timestamp) {
      if (!timestamp) return '-'
      return formatDatetime(timestamp)
    }
  }
}
</script>

<style scoped>
.sub-text { color: #80868B; font-size: 11px; line-height: 14px; }
.delete-tip { color: #606266; font-size: 13px; margin-bottom: 12px; }
.section-title { font-weight: 600; margin-bottom: 6px; color: #303133; }
.json { background: #f5f7fa; padding: 10px; border-radius: 6px; max-height: 240px; overflow: auto; font-size: 12px; }
</style>
