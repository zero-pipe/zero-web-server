<template>
  <div id="region" class="app-container" style="height: calc(100vh - 118px); display: grid; grid-template-columns: 400px auto; gap: 12px;">
    <RegionTree
      ref="regionTree"
      :show-header="true"
      :edit="true"
      @clickEvent="treeNodeClickEvent"
      @onChannelChange="onChannelChange"
      :enable-add-channel="true"
      :add-channel-to-civil-code="addChannelToCivilCode"
    />
    <div style="padding: 0 12px; min-width: 0;">
      <el-form :inline="true" size="mini">
        <el-form-item>
          <el-breadcrumb v-if="regionParents.length > 0" separator="/" style="display: ruby">
            <el-breadcrumb-item v-for="key in regionParents" :key="key">{{ key }}</el-breadcrumb-item>
          </el-breadcrumb>
          <div v-else style="color: #1A73E8">未选择行政区划</div>
        </el-form-item>
        <div style="float: right;">
          <el-form-item>
            <el-input
              v-model="searchStr"
              style="width: 10rem; margin-right: 1rem;"
              placeholder="关键字"
              prefix-icon="el-icon-search"
              clearable
              @input="search"
            />
          </el-form-item>
          <el-form-item label="在线状态">
            <el-select
              v-model="online"
              style="width: 8rem; margin-right: 1rem;"
              placeholder="请选择"
              default-first-option
              @change="search"
            >
              <el-option label="全部" value="" />
              <el-option label="在线" value="true" />
              <el-option label="离线" value="false" />
            </el-select>
          </el-form-item>
          <el-form-item label="协议" >
            <el-select
              v-model="protocol"
              style="width: 8rem; margin-right: 1rem;"
              placeholder="请选择"
              default-first-option
              @change="search"
            >
              <el-option label="全部" value="" />
              <el-option label="GB28181" value="gb28181" />
              <el-option label="ONVIF" value="onvif" />
              <el-option label="SDK" value="plugin" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="add()">
              添加设备
            </el-button>
            <el-button :disabled="multipleSelection.length === 0" type="danger" @click="remove()">
              移除设备
            </el-button>
            <el-button plain type="warning" @click="showUnusualChanel()">
              异常挂载通道
            </el-button>
            <el-button plain @click="openBuild">按区划建树</el-button>
            <el-button plain @click="openMapping">编码映射</el-button>
          </el-form-item>
          <el-form-item >
            <el-button icon="el-icon-refresh-right" circle @click="loadDevices()" />
          </el-form-item>
        </div>
      </el-form>
      <el-table
        ref="deviceTable"
        size="medium"
        :data="deviceList"
        :height="tableHeight"
        style="width: 100%"
        header-row-class-name="table-header"
        @selection-change="handleSelectionChange"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="name" label="名称" min-width="180" show-overflow-tooltip />
        <el-table-column label="厂家" min-width="120" show-overflow-tooltip>
          <template v-slot:default="scope">{{ scope.row.vendor || '—' }}</template>
        </el-table-column>
        <el-table-column label="协议" min-width="120">
          <template v-slot:default="scope">
            <protocol-tag :protocol="scope.row.protocol" />
          </template>
        </el-table-column>
        <el-table-column label="位置信息" min-width="170">
          <template v-slot:default="scope">
            <template v-if="scope.row.longitude && scope.row.latitude">
              <div>经度：{{ scope.row.longitude }}</div>
              <div>纬度：{{ scope.row.latitude }}</div>
            </template>
            <span v-else>无</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="100">
          <template v-slot:default="scope">
            <el-tag v-if="scope.row.online" size="medium">在线</el-tag>
            <el-tag v-else size="medium" type="info">离线</el-tag>
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
    <DeviceSelectDialog ref="deviceSelect" />
    <UnusualRegionChannelSelect ref="unusualRegionChannelSelect" />
    <el-dialog title="按行政区划建树" :visible.sync="buildDialog" width="460px" @open="loadProvinces">
      <el-alert type="info" :closable="false" style="margin-bottom:12px;"
        title="选择到需要的层级后确认，系统会自动补齐上级区划节点（已存在的跳过）。" />
      <el-form label-width="90px" size="small">
        <el-form-item label="省/直辖市">
          <el-select v-model="selProv" filterable placeholder="请选择" style="width:100%;" @change="onProvChange">
            <el-option v-for="p in provList" :key="p.deviceId" :label="p.name" :value="p.deviceId" />
          </el-select>
        </el-form-item>
        <el-form-item label="市/地区">
          <el-select v-model="selCity" filterable placeholder="可选" clearable style="width:100%;" @change="onCityChange">
            <el-option v-for="c in cityList" :key="c.deviceId" :label="c.name" :value="c.deviceId" />
          </el-select>
        </el-form-item>
        <el-form-item label="区/县">
          <el-select v-model="selDist" filterable placeholder="可选" clearable style="width:100%;">
            <el-option v-for="d in distList" :key="d.deviceId" :label="d.name" :value="d.deviceId" />
          </el-select>
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button size="small" @click="buildDialog = false">取消</el-button>
        <el-button size="small" type="primary" :loading="buildLoading" @click="confirmBuild">确定建树</el-button>
      </div>
    </el-dialog>

    <el-dialog title="区划节点对外编码映射" :visible.sync="mappingDialog" width="640px" @open="loadMappings">
      <el-alert type="info" :closable="false" style="margin-bottom:12px;"
        title="上级平台（如省厅）指定区划节点国标码时在此配置；未配置则按 区划码补0+00+216+0+000000 推导。" />
      <el-form :inline="true" size="mini">
        <el-form-item label="区划编号"><el-input v-model="mappingForm.regionDeviceId" placeholder="如 530102" style="width:140px;" /></el-form-item>
        <el-form-item label="对外国标码"><el-input v-model="mappingForm.gbCode" placeholder="20位" maxlength="20" style="width:220px;" /></el-form-item>
        <el-form-item><el-button type="primary" size="mini" @click="saveMapping">保存</el-button></el-form-item>
      </el-form>
      <el-table size="small" :data="mappingList" height="320">
        <el-table-column prop="regionDeviceId" label="区划编号" width="160" />
        <el-table-column prop="gbCode" label="对外国标码" />
        <el-table-column label="操作" width="80">
          <template v-slot:default="scope">
            <el-button type="text" icon="el-icon-delete" style="color:#EA4335" title="删除" aria-label="删除" @click="deleteMapping(scope.row)" />
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script>
import RegionTree from '../..//common/RegionTree.vue'
import DeviceSelectDialog from '@/views/devices/DeviceSelectDialog.vue'
import ProtocolTag from '@/components/ProtocolTag.vue'
import UnusualRegionChannelSelect from './UnusualRegionChannelSelect.vue'
import { addByCivilCode, queryChildListInBase, codeMappingList, saveCodeMapping, deleteCodeMapping } from '@/api/region'
import { queryChannels, queryAssets } from '@/api/asset'

export default {
  name: 'Region',
  components: {
    DeviceSelectDialog,
    ProtocolTag,
    RegionTree,
    UnusualRegionChannelSelect
  },
  data() {
    return {
      deviceList: [],
      allDevices: [],
      tableHeight: 'calc(100vh - 190px)',
      searchStr: '',
      protocol: '',
      online: '',
      currentPage: 1,
      count: 15,
      total: 0,
      loading: false,
      loadSnap: {},
      regionId: '',
      regionDeviceId: '',
      regionParents: [],
      multipleSelection: [],
      buildDialog: false,
      buildLoading: false,
      provList: [],
      cityList: [],
      distList: [],
      selProv: '',
      selCity: '',
      selDist: '',
      mappingDialog: false,
      mappingList: [],
      mappingForm: { regionDeviceId: '', gbCode: '' }
    }
  },

  created() {
    this.initData()
  },
  destroyed() {
  },
  methods: {
    openBuild() {
      this.selProv = ''
      this.selCity = ''
      this.selDist = ''
      this.cityList = []
      this.distList = []
      this.buildDialog = true
    },
    async loadProvinces() {
      const res = await queryChildListInBase('')
      this.provList = res.data || []
    },
    async onProvChange(code) {
      this.selCity = ''
      this.selDist = ''
      this.distList = []
      const res = await queryChildListInBase(code)
      this.cityList = res.data || []
    },
    async onCityChange(code) {
      this.selDist = ''
      const res = await queryChildListInBase(code)
      this.distList = res.data || []
    },
    async confirmBuild() {
      const code = this.selDist || this.selCity || this.selProv
      if (!code) {
        this.$message.warning('请至少选择省级')
        return
      }
      this.buildLoading = true
      try {
        const res = await addByCivilCode(code)
        const created = (res.data || []).length
        this.$message.success(created > 0 ? `已创建 ${created} 个区划节点` : '区划节点已存在')
        this.buildDialog = false
        if (this.$refs.regionTree && this.$refs.regionTree.loadTree) {
          this.$refs.regionTree.loadTree(null)
        }
      } finally {
        this.buildLoading = false
      }
    },
    openMapping() {
      this.mappingDialog = true
    },
    async loadMappings() {
      const res = await codeMappingList()
      this.mappingList = res.data || []
    },
    async saveMapping() {
      if (!this.mappingForm.regionDeviceId || !this.mappingForm.gbCode) {
        this.$message.warning('请填写区划编号与国标码')
        return
      }
      await saveCodeMapping(this.mappingForm)
      this.$message.success('已保存')
      this.mappingForm = { regionDeviceId: '', gbCode: '' }
      this.loadMappings()
    },
    async deleteMapping(row) {
      await deleteCodeMapping(row.regionDeviceId)
      this.loadMappings()
    },
    initData: function() {
      this.loadDevices()
    },
    currentChange: function(val) {
      this.currentPage = val
      this.applyPage()
    },
    handleSizeChange: function(val) {
      this.count = val
      this.currentPage = 1
      this.applyPage()
    },
    // 拉取该区划下所有通道 → 按设备归并成设备行，再用资产详情补厂家/协议。
    loadDevices: function() {
      const civil = this.regionDeviceId
      if (!civil) {
        this.allDevices = []
        this.deviceList = []
        this.total = 0
        return
      }
      Promise.all([
        queryChannels({ page: 1, count: 9999 }),
        queryAssets({ page: 1, count: 9999 })
      ]).then(([chRes, aRes]) => {
        const chans = ((chRes.data && chRes.data.list) || []).filter(c =>
          c.gb && String(c.gb.civilCode || '').trim() === String(civil)
        )
        const assetMap = {}
        ;((aRes.data && aRes.data.list) || []).forEach(a => {
          if (a.device) {
            assetMap[a.device.id] = {
              name: a.device.name,
              vendor: a.device.vendor,
              protocol: (a.binding && a.binding.protocol) || ''
            }
          }
        })
        const groups = {}
        chans.forEach(c => {
          const key = c.deviceId || ('ch' + c.resourceId)
          if (!groups[key]) {
            groups[key] = {
              deviceId: key, name: '', vendor: '', protocol: '',
              longitude: 0, latitude: 0, online: false, channelCount: 0
            }
          }
          const g = groups[key]
          g.channelCount++
          if (c.status === 'ON') g.online = true
          let lon = c.longitude
          let lat = c.latitude
          if ((!lon || !lat) && c.gb) {
            lon = c.gb.gbLongitude || lon
            lat = c.gb.gbLatitude || lat
          }
          if ((!g.longitude || !g.latitude) && lon > 0 && lat > 0) {
            g.longitude = lon
            g.latitude = lat
          }
          if (!g.name && c.deviceName) g.name = c.deviceName
          if (!g.protocol && c.protocol) g.protocol = c.protocol
        })
        let rows = Object.values(groups).map(g => {
          const a = assetMap[g.deviceId] || {}
          return {
            ...g,
            name: a.name || g.name || '未命名',
            vendor: a.vendor || '',
            protocol: a.protocol || g.protocol || ''
          }
        })
        if (this.searchStr) {
          const kw = this.searchStr.toLowerCase()
          rows = rows.filter(r => (r.name || '').toLowerCase().includes(kw))
        }
        if (this.online === 'true') rows = rows.filter(r => r.online)
        if (this.online === 'false') rows = rows.filter(r => !r.online)
        if (this.protocol) rows = rows.filter(r => r.protocol === this.protocol)
        this.allDevices = rows
        this.total = rows.length
        this.applyPage()
      }).catch(() => {
        this.allDevices = []
        this.deviceList = []
        this.total = 0
      })
    },
    applyPage: function() {
      const start = (this.currentPage - 1) * this.count
      this.deviceList = this.allDevices.slice(start, start + this.count)
      this.$nextTick(() => {
        this.$refs.deviceTable && this.$refs.deviceTable.doLayout()
      })
    },
    handleSelectionChange: function(val) {
      this.multipleSelection = val
    },
    add: function(row) {
      if (this.regionDeviceId === '') {
        this.$message.info({
          showClose: true,
          message: '请选择左侧行政区划'
        })
        return
      }
      this.$refs.deviceSelect.openDialog((rows) => {
        const deviceIds = rows.map(r => r.dbId)
        if (deviceIds.length === 0) return
        this.loading = true
        this.$store.dispatch('commonChanel/addDeviceToRegion', {
          civilCode: this.regionDeviceId,
          deviceIds: deviceIds
        }).then(() => {
          this.$message.success({ showClose: true, message: '保存成功' })
          this.loadDevices()
          this.$refs.regionTree.refresh(this.regionDeviceId)
        }).catch((error) => {
          this.$message.error({ showClose: true, message: error })
        }).finally(() => {
          this.loading = false
        })
      })
    },
    addChannelToCivilCode: function(regionDeviceId, data) {
      if (data.length === 0) {
        return
      }
      const channels = []
      for (let i = 0; i < data.length; i++) {
        channels.push(data[i].gbId)
      }
      this.loading = true
      this.$store.dispatch('commonChanel/addToRegion', {
        civilCode: regionDeviceId,
        channelIds: channels
      })
        .then(data => {
          this.$message.success({
            showClose: true,
            message: '保存成功'
          })
          this.loadDevices()
          this.$refs.regionTree.refresh(this.regionDeviceId)
        })
        .catch((error) => {
          this.$message.error({
            showClose: true,
            message: error
          })
        })
        .finally(() => {
          this.loading = false
        })
    },
    remove: function(row) {
      const deviceIds = this.multipleSelection
        .map(r => Number(r.deviceId))
        .filter(n => !isNaN(n))
      if (deviceIds.length === 0) {
        this.$message.info({
          showClose: true,
          message: '请选择设备'
        })
        return
      }
      this.loading = true
      this.$store.dispatch('commonChanel/deleteDeviceFromRegion', deviceIds)
        .then(data => {
          this.$message.success({
            showClose: true,
            message: '保存成功'
          })
          this.loadDevices()
          // 刷新树节点
          this.$refs.regionTree.refresh(this.regionDeviceId)
        })
        .catch((error) => {
          this.$message.error({
            showClose: true,
            message: error
          })
        })
        .finally(() => {
          this.loading = false
        })
    },
    showUnusualChanel: function() {
      this.$refs.unusualRegionChannelSelect.openDialog()
    },
    search: function() {
      this.currentPage = 1
      this.total = 0
      this.initData()
    },
    refresh: function() {
      this.initData()
    },
    treeNodeClickEvent: function(region) {
      this.regionDeviceId = region.deviceId
      if (region.deviceId === '') {
        this.deviceList = []
        this.allDevices = []
        this.regionParents = []
      }
      this.initData()
      this.$store.dispatch('region/queryPath', this.regionDeviceId)
        .then(data => {
          const path = []
          for (let i = 0; i < data.length; i++) {
            path.push(data[i].name)
          }
          this.regionParents = path
        })
    },
    gbChannelSelectEnd: function(selectedData) {
      console.log(selectedData)
    },
    onChannelChange: function(deviceId) {
      if (this.regionDeviceId === deviceId) {
        this.loadDevices()
      }
    }
  }
}
</script>
