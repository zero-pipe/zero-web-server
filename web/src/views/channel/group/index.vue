<template>
  <div id="region" class="app-container" style="height: calc(100vh - 118px); display: grid; grid-template-columns: 400px auto; gap: 12px;">
    <GroupTree
      ref="groupTree"
      :show-header="true"
      :edit="true"
      @clickEvent="treeNodeClickEvent"
      @onChannelChange="onChannelChange"
      :enable-add-channel="true"
      :add-channel-to-group="addChannelToGroup"
    />
    <div style="padding: 0 12px; min-width: 0;">
      <el-form :inline="true" size="mini" ref="queryForm">
        <el-form-item>
          <el-breadcrumb v-if="regionParents.length > 0" separator="/" style="display: ruby">
            <el-breadcrumb-item v-for="key in regionParents" :key="key">{{ key }}</el-breadcrumb-item>
          </el-breadcrumb>
          <div v-else style="color: #1A73E8">未选择虚拟组织</div>
        </el-form-item>
        <div style="float: right;">
          <el-form-item>
            <el-input
              v-model="searchStr"
              style="margin-right: 1rem; width: 10rem; "
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
          <el-form-item label="协议">
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
          </el-form-item>
          <el-form-item>
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
    <UnusualGroupChannelSelect ref="unusualGroupChannelSelect" />
  </div>
</template>

<script>
import GroupTree from '../../common/GroupTree.vue'
import DeviceSelectDialog from '@/views/devices/DeviceSelectDialog.vue'
import ProtocolTag from '@/components/ProtocolTag.vue'
import UnusualGroupChannelSelect from './UnusualGroupChannelSelect.vue'
import { isBusinessGroupNode } from '@/utils/gbCode'
import { queryChannels, queryAssets } from '@/api/asset'

export default {
  name: 'Group',
  components: {
    DeviceSelectDialog,
    ProtocolTag,
    UnusualGroupChannelSelect,
    GroupTree
  },
  data() {
    return {
      deviceList: [],
      allDevices: [],
      tableHeight: 'calc(100vh - 190px)',
      searchStr: '',
      protocol: '',
      online: '',
      hasGroup: 'false',
      currentPage: 1,
      count: 15,
      total: 0,
      loading: false,
      loadSnap: {},
      groupDeviceId: '',
      groupId: '',
      businessGroup: '',
      regionParents: [],
      multipleSelection: []
    }
  },

  created() {
    this.initData()
  },
  mounted() {
    if (this.$refs.queryForm) {
      this.tableHeight = `calc(100vh - ${this.$refs.queryForm.offsetHeight}px)`
    }
  },
  destroyed() {
  },
  methods: {
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
    // 拉取该虚拟组织下所有通道 → 按设备归并成设备行，再用资产详情补厂家/协议。
    loadDevices: function() {
      const parent = this.groupDeviceId
      if (!parent) {
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
          c.gb && String(c.gb.parentId || '').trim() === String(parent)
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
      if (this.groupDeviceId === '') {
        this.$message.info({
          showClose: true,
          message: '请选择左侧虚拟组织节点'
        })
        return
      }
      this.$refs.deviceSelect.openDialog((rows) => {
        const deviceIds = rows.map(r => r.dbId)
        if (deviceIds.length === 0) return
        this.loading = true
        this.$store.dispatch('commonChanel/addDeviceToGroup', {
          parentId: this.groupDeviceId,
          businessGroup: this.businessGroup,
          deviceIds: deviceIds
        }).then(() => {
          this.$message.success({ showClose: true, message: '保存成功' })
          this.loadDevices()
          this.$refs.groupTree.refresh(this.groupDeviceId)
        }).catch((error) => {
          this.$message.error({ showClose: true, message: error })
        }).finally(() => {
          this.loading = false
        })
      })
    },
    addChannelToGroup: function(groupDeviceId, businessGroup, data) {
      if (data.length === 0) {
        return
      }
      const channels = []
      for (let i = 0; i < data.length; i++) {
        channels.push(data[i].gbId)
      }
      this.loading = true
      this.$store.dispatch('commonChanel/addToGroup', {
        parentId: groupDeviceId,
        businessGroup: businessGroup,
        channelIds: channels
      })
        .then(data => {
          this.$message.success({
            showClose: true,
            message: '保存成功'
          })
          this.loadDevices()
          this.$refs.groupTree.refresh(this.groupDeviceId)
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
      this.$store.dispatch('commonChanel/deleteDeviceFromGroup', deviceIds)
        .then(data => {
          this.$message.success({
            showClose: true,
            message: '保存成功'
          })
          this.loadDevices()
          // 刷新树节点
          this.$refs.groupTree.refresh(this.groupDeviceId)
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
    getSnap: function(row) {
      const baseUrl = window.baseUrl ? window.baseUrl : ''
      return ((process.env.NODE_ENV === 'development') ? process.env.VUE_APP_BASE_API : baseUrl) + '/api/v1/devices/gb/query/snap/' + this.deviceId + '/' + row.deviceId
    },
    search: function() {
      this.currentPage = 1
      this.total = 0
      this.initData()
    },
    refresh: function() {
      this.initData()
    },
    treeNodeClickEvent: function(group) {
      if (!group.deviceId) {
        this.deviceList = []
        this.allDevices = []
        this.regionParents = []
        this.groupDeviceId = ''
        this.businessGroup = ''
        return
      }
      if (isBusinessGroupNode(group)) {
        this.deviceList = []
        this.allDevices = []
        this.regionParents = []
        this.groupDeviceId = ''
        this.businessGroup = ''
        return
      }
      this.groupDeviceId = group.deviceId
      this.businessGroup = group.businessGroup
      this.initData()
      this.$store.dispatch('group/getPath', {
        deviceId: this.groupDeviceId,
        businessGroup: this.businessGroup
      })
        .then(data => {
          const path = []
          for (let i = 0; i < data.length; i++) {
            path.push(data[i].name)
          }
          this.regionParents = path
        })
    },
    onChannelChange: function(deviceId) {

      if (this.groupDeviceId === deviceId) {
        this.loadDevices()
      }
    },
    showUnusualChanel: function() {
      this.$refs.unusualGroupChannelSelect.openDialog()
    }
  }
}
</script>
