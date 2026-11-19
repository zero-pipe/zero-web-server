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
          <div v-else style="color: #1565c0">未选择行政区划</div>
        </el-form-item>
        <div style="float: right;">
          <el-form-item label="搜索">
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
          <el-form-item label="类型" >
            <el-select
              v-model="channelType"
              style="width: 8rem; margin-right: 1rem;"
              placeholder="请选择"
              default-first-option
              @change="getChannelList"
            >
              <el-option label="全部" value="" />
              <el-option v-for="item in Object.values($channelTypeList)" :key="item.id" :label="item.name" :value="item.id" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="add()">
              添加通道
            </el-button>
            <el-button :disabled="multipleSelection.length === 0" type="danger" @click="remove()">
              移除通道
            </el-button>
            <el-button plain type="warning" @click="showUnusualChanel()">
              异常挂载通道
            </el-button>
            <el-button plain @click="openBuild">按区划建树</el-button>
            <el-button plain @click="openMapping">编码映射</el-button>
          </el-form-item>
          <el-form-item >
            <el-button icon="el-icon-refresh-right" circle @click="getChannelList()" />
          </el-form-item>
        </div>
      </el-form>
      <el-table
        ref="channelListTable"
        size="medium"
        :data="channelList"
        :height="tableHeight"
        style="width: 100%"
        header-row-class-name="table-header"
        @selection-change="handleSelectionChange"
        @row-dblclick="rowDblclick"
      >
        <el-table-column type="selection" width="55" />
        <el-table-column prop="gbName" label="名称" min-width="180" />
        <el-table-column prop="gbManufacturer" label="厂家" min-width="100" />
        <el-table-column label="类型" min-width="100">
          <template v-slot:default="scope">
            <div slot="reference" class="name-wrapper">
              <el-tag
                v-if="$channelTypeList[scope.row.dataType]"
                size="medium"
                effect="plain"
                type="success"
                :style="$channelTypeList[scope.row.dataType].style"
              >{{ $channelTypeList[scope.row.dataType].name }}</el-tag>
              <el-tag v-else size="medium" effect="plain" type="info">未知</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="位置信息" min-width="150">
          <template v-slot:default="scope">
            <span v-if="scope.row.gbLongitude && scope.row.gbLatitude">{{ scope.row.gbLongitude }}<br>{{ scope.row.gbLatitude }}</span>
            <span v-if="!scope.row.gbLongitude || !scope.row.gbLatitude">无</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="100">
          <template v-slot:default="scope">
            <div slot="reference" class="name-wrapper">
              <el-tag v-if="scope.row.gbStatus === 'ON'" size="medium">在线</el-tag>
              <el-tag v-if="scope.row.gbStatus !== 'ON'" size="medium" type="info">离线</el-tag>
            </div>
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
    <GbChannelSelect ref="gbChannelSelect" data-type="civilCode" />
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
            <el-button type="text" style="color:#f56c6c" @click="deleteMapping(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script>
import RegionTree from '../..//common/RegionTree.vue'
import GbChannelSelect from '../../dialog/GbChannelSelect.vue'
import UnusualRegionChannelSelect from './UnusualRegionChannelSelect.vue'
import { addByCivilCode, queryChildListInBase, codeMappingList, saveCodeMapping, deleteCodeMapping } from '@/api/region'

export default {
  name: 'Region',
  components: {
    GbChannelSelect,
    RegionTree,
    UnusualRegionChannelSelect
  },
  data() {
    return {
      channelList: [],
      tableHeight: 'calc(100vh - 190px)',
      searchStr: '',
      channelType: '',
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
    this.tableHeight = `calc(100vh - ${this.$refs.queryForm.offsetHeight}px)`
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
      this.getChannelList()
    },
    currentChange: function(val) {
      this.currentPage = val
      this.initData()
    },
    handleSizeChange: function(val) {
      this.count = val
      this.getChannelList()
    },
    getChannelList: function() {
      this.$store.dispatch('commonChanel/getCivilCodeList', {
        page: this.currentPage,
        count: this.count,
        query: this.searchStr,
        online: this.online,
        channelType: this.channelType,
        civilCode: this.regionDeviceId
      })
        .then(data => {
          this.total = data.total
          this.channelList = data.list
          // 防止出现表格错位
          this.$nextTick(() => {
            this.$refs.channelListTable.doLayout()
          })
        })
    },
    handleSelectionChange: function(val) {
      this.multipleSelection = val
    },
    rowDblclick: function(row, rowIndex) {
      // if (row.gbCivilCode) {
      //   this.$refs.regionTree.refresh(row.gbCivilCode)
      // }
    },
    add: function(row) {
      if (this.regionDeviceId === '') {
        this.$message.info({
          showClose: true,
          message: '请选择左侧行政区划'
        })
        return
      }
      this.$refs.gbChannelSelect.openDialog((data) => {
        console.log('选择的数据')
        console.log(data)
        this.addChannelToCivilCode(this.regionDeviceId, data)
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
          this.getChannelList()
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
      const channels = []
      for (let i = 0; i < this.multipleSelection.length; i++) {
        channels.push(this.multipleSelection[i].gbId)
      }
      if (channels.length === 0) {
        this.$message.info({
          showClose: true,
          message: '请选择通道'
        })
        return
      }
      this.loading = true
      this.$store.dispatch('commonChanel/deleteFromRegion', channels)
        .then(data => {
          this.$message.success({
            showClose: true,
            message: '保存成功'
          })
          this.getChannelList()
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
        this.channelList = []
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
        this.getChannelList()
      }
    }
  }
}
</script>
