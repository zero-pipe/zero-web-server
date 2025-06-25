<template>
  <div style="height: calc(100vh - 124px);">
    <el-form :inline="true" size="mini">
      <el-form-item label="搜索">
        <el-input
          v-model="searchStr"
          placeholder="名称/IP"
          prefix-icon="el-icon-search"
          clearable
          @input="loadDevices"
        />
      </el-form-item>
      <el-form-item>
        <el-button type="primary" icon="el-icon-search" @click="handleDiscover">局域网发现</el-button>
        <el-button type="primary" icon="el-icon-plus" @click="showAddDialog = true">手动添加</el-button>
        <el-button icon="el-icon-refresh" @click="handleProbe">在线探测</el-button>
      </el-form-item>
      <el-form-item style="float: right;">
        <el-button icon="el-icon-refresh-right" circle :loading="loading" @click="loadDevices" />
      </el-form-item>
    </el-form>

    <el-table v-loading="loading" :data="deviceList" size="small" height="calc(100% - 64px)">
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column label="地址" min-width="160">
        <template v-slot:default="scope">
          {{ scope.row.ip }}:{{ scope.row.port }}
        </template>
      </el-table-column>
      <el-table-column prop="manufacturer" label="厂商" min-width="120" />
      <el-table-column prop="model" label="型号" min-width="120" />
      <el-table-column label="在线" width="80">
        <template v-slot:default="scope">
          <el-tag v-if="scope.row.onLine" size="mini" type="success">在线</el-tag>
          <el-tag v-else size="mini" type="info">离线</el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="280" fixed="right">
        <template v-slot:default="scope">
          <el-button size="mini" type="primary" @click="$emit('show-channel', scope.row.id)">通道</el-button>
          <el-button size="mini" @click="handleSync(scope.row.id)">同步</el-button>
          <el-button size="mini" type="danger" @click="handleDelete(scope.row.id)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog title="手动添加 ONVIF 设备" :visible.sync="showAddDialog" width="480px">
      <el-form label-width="90px" size="small">
        <el-form-item label="IP">
          <el-input v-model="form.ip" placeholder="192.168.1.100" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input v-model.number="form.port" placeholder="80" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="form.username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" show-password />
        </el-form-item>
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="可选" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button @click="showAddDialog = false">取消</el-button>
        <el-button type="primary" :loading="adding" @click="handleAdd">确定</el-button>
      </div>
    </el-dialog>

    <el-dialog title="发现设备" :visible.sync="showDiscoverDialog" width="720px">
      <el-table :data="discoveredList" size="small" max-height="400">
        <el-table-column prop="name" label="名称" />
        <el-table-column label="地址">
          <template v-slot:default="scope">{{ scope.row.ip }}:{{ scope.row.port }}</template>
        </el-table-column>
        <el-table-column prop="location" label="位置" />
        <el-table-column label="操作" width="100">
          <template v-slot:default="scope">
            <el-button size="mini" type="primary" @click="addDiscovered(scope.row)">添加</el-button>
          </template>
        </el-table-column>
      </el-table>
    </el-dialog>
  </div>
</template>

<script>
import { addDevice, deleteDevice, discoverDevices, probeDevices, queryDevices, syncDevice } from '@/api/onvif'

export default {
  name: 'OnvifDeviceList',
  data() {
    return {
      loading: false,
      adding: false,
      searchStr: '',
      deviceList: [],
      showAddDialog: false,
      showDiscoverDialog: false,
      discoveredList: [],
      form: {
        ip: '',
        port: 80,
        username: 'admin',
        password: '',
        name: ''
      }
    }
  },
  mounted() {
    this.loadDevices()
  },
  methods: {
    loadDevices() {
      this.loading = true
      queryDevices({ page: 1, count: 100, query: this.searchStr }).then(res => {
        this.deviceList = (res.data && res.data.list) || []
      }).finally(() => {
        this.loading = false
      })
    },
    handleDiscover() {
      this.loading = true
      discoverDevices(5).then(res => {
        this.discoveredList = res.data || []
        this.showDiscoverDialog = true
      }).finally(() => {
        this.loading = false
      })
    },
    handleAdd() {
      this.adding = true
      addDevice(this.form).then(() => {
        this.$message.success('添加成功')
        this.showAddDialog = false
        this.loadDevices()
      }).finally(() => {
        this.adding = false
      })
    },
    addDiscovered(row) {
      this.form.ip = row.ip
      this.form.port = row.port || 80
      this.form.name = row.name
      this.showDiscoverDialog = false
      this.showAddDialog = true
    },
    handleSync(id) {
      syncDevice(id).then(() => {
        this.$message.success('同步成功')
      })
    },
    handleDelete(id) {
      this.$confirm('确认删除该设备？', '提示', { type: 'warning' }).then(() => {
        deleteDevice(id).then(() => {
          this.$message.success('删除成功')
          this.loadDevices()
        })
      })
    },
    handleProbe() {
      probeDevices().then(() => {
        this.$message.success('探测完成')
        this.loadDevices()
      })
    }
  }
}
</script>
