<template>
  <el-drawer
    :title="'编辑设备 · ' + (info.name || '')"
    :visible.sync="innerVisible"
    size="760px"
    :destroy-on-close="true"
    @close="reset"
  >
    <div v-loading="loading" class="de-wrap">
      <el-tabs v-model="activeTab">
        <!-- 基本信息 -->
        <el-tab-pane label="基本信息" name="basic">
          <el-form :model="form" label-width="110px" size="small">
            <el-form-item label="设备名称" required>
              <el-input v-model="form.name" maxlength="64" show-word-limit />
            </el-form-item>
            <el-form-item label="设备类型">
              <el-select v-model="form.deviceType" clearable placeholder="请选择" style="width: 100%">
                <el-option v-for="it in dicts.device_type" :key="it.value" :label="it.label" :value="it.value" />
              </el-select>
            </el-form-item>
            <el-form-item label="厂商">
              <el-select v-model="form.vendor" clearable placeholder="请选择" style="width: 100%">
                <el-option v-for="it in dicts.manufacturer" :key="it.value" :label="it.label" :value="it.value" />
              </el-select>
            </el-form-item>
            <el-form-item label="型号">
              <el-input v-model="form.model" />
            </el-form-item>
            <el-form-item label="序列号">
              <el-input v-model="form.serialNumber" />
            </el-form-item>
            <el-divider content-position="left">接入信息（只读）</el-divider>
            <el-form-item label="协议">
              <el-tag size="small">{{ protocolLabel }}</el-tag>
              <span class="ro">{{ info.accessMode === 'passive' ? '被动接入' : '主动接入' }}</span>
            </el-form-item>
            <el-form-item label="编号/地址">
              <el-input :value="bindingText" disabled />
            </el-form-item>
          </el-form>
        </el-tab-pane>

        <!-- 扩展信息（位置） -->
        <el-tab-pane label="扩展信息（位置）" name="ext">
          <el-form :model="form" label-width="110px" size="small">
            <el-form-item label="安装位置">
              <el-input v-model="form.address" placeholder="如 安徽省合肥市xx路与xx交口" />
            </el-form-item>
            <el-form-item label="经度">
              <el-input v-model="form.longitude" type="number" placeholder="如 117.227239" />
            </el-form-item>
            <el-form-item label="纬度">
              <el-input v-model="form.latitude" type="number" placeholder="如 31.820586" />
            </el-form-item>
            <el-form-item label="坐标系">
              <el-select v-model="form.geoCoordSys" clearable placeholder="请选择" style="width: 100%">
                <el-option v-for="it in dicts.coordinate_system" :key="it.value" :label="it.label" :value="it.value" />
              </el-select>
            </el-form-item>
            <el-form-item label="地图选点">
              <el-button size="mini" icon="el-icon-location-outline" @click="openMapPick">在地图上选点</el-button>
              <span class="pick-hint">点选后按 WGS84 回填经纬度</span>
            </el-form-item>
            <el-form-item label="备注">
              <el-input v-model="form.remark" />
            </el-form-item>
            <el-form-item label="标签">
              <el-input v-model="form.tags" placeholder="逗号分隔" />
            </el-form-item>
            <el-alert type="info" :closable="false" show-icon
              title="经纬度/安装位置在设备上维护，保存后会同步到该设备的全部通道。" />
          </el-form>
        </el-tab-pane>

        <!-- 通道列表 -->
        <el-tab-pane label="通道列表" name="channels">
          <el-table :data="channels" size="small" border>
            <el-table-column label="通道名称" min-width="160">
              <template v-slot:default="{ row }">
                <el-input v-model="row.name" size="mini" />
              </template>
            </el-table-column>
            <el-table-column label="通道类型" width="150">
              <template v-slot:default="{ row }">
                <el-select v-model="row.channelType" size="mini" clearable placeholder="未设">
                  <el-option v-for="it in dicts.channel_type" :key="it.value" :label="it.label" :value="it.value" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column label="监视方位" width="140">
              <template v-slot:default="{ row }">
                <el-select v-model="row.direction" size="mini" clearable placeholder="未设">
                  <el-option v-for="o in directionOptions" :key="o.value" :label="o.label" :value="o.value" />
                </el-select>
              </template>
            </el-table-column>
            <el-table-column prop="kind" label="类型" width="110" />
            <el-table-column prop="status" label="状态" width="80" />
            <el-table-column label="编码" min-width="150" show-overflow-tooltip>
              <template v-slot:default="{ row }">{{ row.gbCode || row.externalKey || '—' }}</template>
            </el-table-column>
          </el-table>
          <div class="tip">通道只能改名称 / 通道类型 / 监视方位；IP、编号等不在此修改。</div>
        </el-tab-pane>
      </el-tabs>
    </div>
    <div class="de-footer">
      <el-button size="small" @click="innerVisible = false">取消</el-button>
      <el-button type="primary" size="small" :loading="saving" @click="save">保存</el-button>
    </div>
    <MapPickDialog ref="mapPick" />
  </el-drawer>
</template>

<script>
import { getAssetDetail, updateAsset, updateChannel } from '@/api/asset'
import { listDictItems } from '@/api/dict'
import MapPickDialog from './MapPickDialog.vue'

const DIRECTION_OPTIONS = [
  { value: 0, label: '未设置' },
  { value: 1, label: '东(西→东)' }, { value: 2, label: '西(东→西)' },
  { value: 3, label: '南(北→南)' }, { value: 4, label: '北(南→北)' },
  { value: 5, label: '东南' }, { value: 6, label: '东北' },
  { value: 7, label: '西南' }, { value: 8, label: '西北' },
  { value: 91, label: '左' }, { value: 92, label: '后' }, { value: 93, label: '前' }, { value: 94, label: '右' },
  { value: 95, label: '左前' }, { value: 96, label: '右前' }, { value: 97, label: '左后' }, { value: 98, label: '右后' }
]

const dictTypes = ['device_type', 'manufacturer', 'coordinate_system', 'channel_type']

export default {
  name: 'DeviceEditDrawer',
  components: { MapPickDialog },
  props: {
    visible: { type: Boolean, default: false },
    row: { type: Object, default: null }
  },
  data() {
    return {
      activeTab: 'basic',
      loading: false,
      saving: false,
      info: {},
      info2: {},
      bindingText: '',
      form: this.emptyForm(),
      channels: [],
      dicts: { device_type: [], manufacturer: [], coordinate_system: [], channel_type: [] },
      directionOptions: DIRECTION_OPTIONS
    }
  },
  computed: {
    innerVisible: {
      get() { return this.visible },
      set(v) { this.$emit('update:visible', v) }
    },
    protocolLabel() {
      return { gb28181: 'GB28181', onvif: 'ONVIF', plugin: '插件' }[this.info.protocol] || '—'
    }
  },
  watch: {
    visible(v) {
      if (v && this.row) this.open(this.row)
    }
  },
  methods: {
    openMapPick() {
      const lon = parseFloat(this.form.longitude)
      const lat = parseFloat(this.form.latitude)
      const center = (!isNaN(lon) && !isNaN(lat) && lon && lat) ? [lon, lat] : null
      this.$refs.mapPick.openDialog(center, ([plon, plat]) => {
        this.form.longitude = plon.toFixed(6)
        this.form.latitude = plat.toFixed(6)
        this.form.geoCoordSys = 'WGS84'
      })
    },
    emptyForm() {
      return {
        name: '', deviceType: '', vendor: '', model: '', serialNumber: '',
        address: '', longitude: '', latitude: '', geoCoordSys: '', remark: '', tags: ''
      }
    },
    loadDicts() {
      dictTypes.forEach(t => {
        listDictItems(t, true).then(res => {
          this.$set(this.dicts, t, (res.data && res.data.list) || [])
        }).catch(() => {})
      })
    },
    open(row) {
      this.activeTab = 'basic'
      this.channels = []
      this.loadDicts()
      this.loading = true
      getAssetDetail(row.dbId).then(res => {
        const d = res.data || {}
        const dev = d.device || {}
        const ext = d.ext || {}
        const b = d.binding || {}
        this.info = { protocol: b.protocol || row.protocol, accessMode: b.mode || row.accessMode, name: dev.name }
        this.info2 = dev
        this.bindingText = b.externalId || b.endpoint || dev.internalCode || ''
        this.form = {
          name: dev.name || '',
          deviceType: dev.deviceType || '',
          vendor: dev.vendor || '',
          model: dev.model || '',
          serialNumber: dev.serialNumber || '',
          address: ext.address || '',
          longitude: ext.longitude != null ? String(ext.longitude) : '',
          latitude: ext.latitude != null ? String(ext.latitude) : '',
          geoCoordSys: ext.geoCoordSys || '',
          remark: ext.remark || '',
          tags: ext.tags || ''
        }
        this.channels = (d.resources || []).map(r => ({
          internalCode: r.internalCode,
          name: r.name,
          channelType: r.channelType || '',
          direction: r.direction || 0,
          kind: this.kindLabel(r.kind),
          status: r.status,
          gbCode: r.gbCode,
          externalKey: r.externalKey
        }))
      }).catch(e => this.$message.error((e && e.message) || '加载失败')).finally(() => { this.loading = false })
    },
    kindLabel(k) {
      return { video_sensor: '视频', audio_input: '音频输入', alarm_input: '报警输入', metadata_source: '元数据' }[k] || k
    },
    num(v) {
      if (v === '' || v === null || v === undefined) return null
      const n = Number(v)
      return isNaN(n) ? null : n
    },
    save() {
      if (!(this.form.name || '').trim()) {
        this.$message.warning('设备名称不能为空')
        return
      }
      this.saving = true
      const f = this.form
      const patch = {
        name: f.name.trim(),
        deviceType: f.deviceType || '',
        vendor: f.vendor || '',
        model: f.model || '',
        serialNumber: f.serialNumber || ''
      }
      // 位置字段：仅在有值时提交（避免把空经纬度写入）
      if (f.address !== '') patch.address = f.address
      if (f.geoCoordSys !== '') patch.geoCoordSys = f.geoCoordSys
      if (f.remark !== '') patch.remark = f.remark
      if (f.tags !== '') patch.tags = f.tags
      const lon = this.num(f.longitude)
      const lat = this.num(f.latitude)
      if (lon !== null) patch.longitude = lon
      if (lat !== null) patch.latitude = lat

      const devReq = updateAsset(this.row.dbId, patch)
      const chReqs = this.channels.map(ch => updateChannel(ch.internalCode, {
        name: (ch.name || '').trim(),
        channelType: ch.channelType || '',
        direction: ch.direction || 0
      }))
      Promise.all([devReq, ...chReqs]).then(() => {
        this.$message.success('已保存')
        this.innerVisible = false
        this.$emit('success')
      }).catch(e => this.$message.error((e && e.message) || '保存失败')).finally(() => { this.saving = false })
    },
    reset() {
      this.form = this.emptyForm()
      this.channels = []
      this.info = {}
    }
  }
}
</script>

<style scoped>
.de-wrap {
  padding: 0 20px 60px;
}
.de-footer {
  position: absolute;
  bottom: 0;
  left: 0;
  right: 0;
  padding: 12px 20px;
  border-top: 1px solid #ebeef5;
  text-align: right;
  background: #fff;
}
.ro {
  margin-left: 8px;
  color: #909399;
  font-size: 12px;
}
.pick-hint {
  margin-left: 10px;
  color: #909399;
  font-size: 12px;
}
.tip {
  margin-top: 8px;
  color: #909399;
  font-size: 12px;
}
</style>
