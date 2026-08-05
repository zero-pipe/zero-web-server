<template>
  <div id="app" class="app-container">
    <el-tabs v-model="cascadeTab" type="border-card">
      <el-tab-pane label="上级平台（本级向上注册）" name="upstream" />
      <el-tab-pane label="下级平台（下级向本级注册）" name="downstream" />
    </el-tabs>

    <div v-if="cascadeTab === 'downstream'" style="height: calc(100vh - 180px); margin-top: 12px;">
      <el-alert
        type="info"
        :closable="false"
        style="margin-bottom: 12px;"
        title="预登记允许接入的下级平台国标编号与密码；下级平台注册成功后可在此查看在线状态。"
      />
      <el-form :inline="true" size="mini">
        <el-form-item label="搜索">
          <el-input v-model="subSearch" size="mini" placeholder="名称/编号" clearable @input="loadSubordinates" />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" icon="el-icon-plus" size="mini" @click="openSubEdit()">添加下级</el-button>
        </el-form-item>
        <el-form-item style="float: right;">
          <el-button icon="el-icon-refresh-right" circle @click="loadSubordinates" />
        </el-form-item>
      </el-form>
      <el-table size="small" :data="subList" height="calc(100% - 120px)" v-loading="subLoading">
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="deviceGBId" label="下级注册编号" min-width="200" />
        <el-table-column label="启用" width="80">
          <template v-slot:default="scope">
            <el-tag :type="scope.row.enable ? '' : 'info'" size="medium">{{ scope.row.enable ? '是' : '否' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="80">
          <template v-slot:default="scope">
            <el-tag :type="scope.row.status ? 'success' : 'info'" size="medium">{{ scope.row.status ? '在线' : '离线' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="地址" min-width="140">
          <template v-slot:default="scope">
            <span v-if="scope.row.ip">{{ scope.row.ip }}:{{ scope.row.port }}</span>
            <span v-else>—</span>
          </template>
        </el-table-column>
        <el-table-column prop="transport" label="传输" width="80" />
        <el-table-column label="操作" width="160" fixed="right">
          <template v-slot:default="scope">
            <el-button type="text" size="medium" @click="openSubEdit(scope.row)">编辑</el-button>
            <el-button type="text" size="medium" style="color:#f56c6c" @click="deleteSub(scope.row)">删除</el-button>
          </template>
        </el-table-column>
      </el-table>
      <el-dialog :title="subForm.id ? '编辑下级平台' : '添加下级平台'" :visible.sync="subDialog" width="480px">
        <el-form :model="subForm" label-width="120px" size="small">
          <el-form-item label="名称" required>
            <el-input v-model="subForm.name" />
          </el-form-item>
          <el-form-item label="下级注册编号" required>
            <el-input v-model="subForm.deviceGBId" maxlength="20" show-word-limit placeholder="20位，须与下级向上注册的 DeviceGBID 一致" />
          </el-form-item>
          <el-form-item label="注册密码" required>
            <el-input v-model="subForm.password" show-password />
          </el-form-item>
          <el-form-item label="启用">
            <el-switch v-model="subForm.enable" />
          </el-form-item>
          <el-form-item label="传输">
            <el-radio-group v-model="subForm.transport">
              <el-radio label="UDP">UDP</el-radio>
              <el-radio label="TCP">TCP</el-radio>
            </el-radio-group>
          </el-form-item>
        </el-form>
        <span slot="footer">
          <el-button size="small" @click="subDialog = false">取消</el-button>
          <el-button type="primary" size="small" :loading="subSaving" @click="saveSub">保存</el-button>
        </span>
      </el-dialog>
    </div>

    <div v-show="cascadeTab === 'upstream' && !platform" style="height: calc(100vh - 180px); margin-top: 12px;">
      <el-alert
        type="info"
        :closable="false"
        style="margin-bottom: 12px;"
        title="本级作为下级：向上级平台 REGISTER。上级平台编号填对端国标编号；本级 DeviceGBID 填本级向上注册身份（可与本级国标配置编号相同或按上级要求）。"
      />
      <el-form :inline="true" size="mini">
        <el-form-item label="搜索">
          <el-input
            v-model="searchStr"
            style="margin-right: 1rem; width: auto;"
            size="mini"
            placeholder="关键字"
            prefix-icon="el-icon-search"
            clearable
            @input="queryList"
          />
        </el-form-item>
        <el-form-item>
          <el-button
            icon="el-icon-plus"
            size="mini"
            style="margin-right: 1rem;"
            type="primary"
            @click="addParentPlatform"
          >添加
          </el-button>
        </el-form-item>
        <el-form-item style="float: right;">
          <el-button icon="el-icon-refresh-right" circle @click="refresh()" />
        </el-form-item>
      </el-form>
      <!--设备列表-->
      <el-table
        size="small"
        :data="platformList"
        style="width: 100%"
        height="calc(100% - 120px)"
        :loading="loading"
      >
        <el-table-column prop="name" label="名称" />
        <el-table-column prop="serverGBId" label="平台编号" min-width="200" />
        <el-table-column label="是否启用" min-width="80">
          <template v-slot:default="scope">
            <div slot="reference" class="name-wrapper">
              <el-tag v-if="scope.row.enable && myServerId !== scope.row.serverId" size="medium" style="border-color: #ecf1af">已启用</el-tag>
              <el-tag v-if="scope.row.enable && myServerId === scope.row.serverId" size="medium">已启用</el-tag>
              <el-tag v-if="!scope.row.enable" size="medium" type="info">未启用</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="状态" min-width="80">
          <template v-slot:default="scope">
            <div slot="reference" class="name-wrapper">
              <el-tag v-if="scope.row.status" size="medium">在线</el-tag>
              <el-tag v-if="!scope.row.status" size="medium" type="info">离线</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="地址" min-width="160">
          <template v-slot:default="scope">
            <div slot="reference" class="name-wrapper">
              <el-tag size="medium">{{ scope.row.serverIp }}:{{ scope.row.serverPort }}</el-tag>
            </div>
          </template>
        </el-table-column>
        <el-table-column prop="deviceGBId" label="设备国标编号" min-width="200" />
        <el-table-column prop="transport" label="信令传输模式" min-width="120" />
        <el-table-column prop="channelCount" label="通道数" min-width="120" />
        <el-table-column label="订阅信息" min-width="120" fixed="right">
          <template v-slot:default="scope">
            <i
              v-if="scope.row.alarmSubscribe"
              style="font-size: 20px"
              title="报警订阅"
              class="iconfont icon-gbaojings subscribe-on "
            />
            <i
              v-if="!scope.row.alarmSubscribe"
              style="font-size: 20px"
              title="报警订阅"
              class="iconfont icon-gbaojings subscribe-off "
            />
            <i v-if="scope.row.catalogSubscribe" title="目录订阅" class="iconfont icon-gjichus subscribe-on" />
            <i v-if="!scope.row.catalogSubscribe" title="目录订阅" class="iconfont icon-gjichus subscribe-off" />
            <i
              v-if="scope.row.mobilePositionSubscribe"
              title="位置订阅"
              class="iconfont icon-gxunjians subscribe-on"
            />
            <i
              v-if="!scope.row.mobilePositionSubscribe"
              title="位置订阅"
              class="iconfont icon-gxunjians subscribe-off"
            />
          </template>
        </el-table-column>

        <el-table-column label="操作" min-width="260" fixed="right">
          <template v-slot:default="scope">
            <el-button size="medium" icon="el-icon-edit" type="text" @click="editPlatform(scope.row)">编辑</el-button>
            <el-button size="medium" icon="el-icon-share" type="text" @click="chooseChannel(scope.row)">通道共享
            </el-button>
            <el-button
              size="medium"
              icon="el-icon-top"
              type="text"
              :loading="pushChannelLoading"
              @click="pushChannel(scope.row)"
            >推送通道
            </el-button>
            <el-button
              size="medium"
              icon="el-icon-delete"
              type="text"
              style="color: #f56c6c"
              @click="deletePlatform(scope.row)"
            >删除
            </el-button>
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

    <platformEdit
      v-if="cascadeTab === 'upstream' && platform"
      ref="platformEdit"
      v-model="platform"
      :close-edit="closeEdit"
      :device-ips="deviceIps"
    />
    <shareChannel ref="shareChannel" />
  </div>
</template>

<script>
import shareChannel from './dialog/shareChannel.vue'
import platformEdit from './edit.vue'
import { querySubordinate, addSubordinate, updateSubordinate, removeSubordinate } from '@/api/subordinate'
import Vue from 'vue'

export default {
  name: 'Platform',
  components: {
    shareChannel,
    platformEdit
  },
  data() {
    return {
      cascadeTab: 'upstream',
      loading: false,
      platformList: [], // 设备列表
      deviceIps: [], // 设备列表
      defaultPlatform: null,
      platform: null,
      pushChannelLoading: false,
      searchStr: '',
      currentPage: 1,
      count: 15,
      total: 0,
      subLoading: false,
      subSaving: false,
      subSearch: '',
      subList: [],
      subDialog: false,
      subForm: {
        id: 0,
        name: '',
        deviceGBId: '',
        password: '',
        enable: true,
        transport: 'UDP'
      }
    }
  },
  computed: {
    Vue() {
      return Vue
    },
    myServerId() {
      return this.$store.getters.serverId
    }
  },
  watch: {
    cascadeTab(v) {
      if (v === 'downstream') {
        this.loadSubordinates()
      }
    }
  },
  mounted() {
    this.initData()
    this.updateLooper = setInterval(this.initData, 10000)
  },
  destroyed() {
    clearTimeout(this.updateLooper)
  },
  methods: {
    loadSubordinates() {
      this.subLoading = true
      querySubordinate({ page: 1, count: 100, query: this.subSearch })
        .then(res => {
          const data = (res && res.data) || res || {}
          this.subList = data.list || []
        })
        .catch(err => this.$message.error(err || '加载下级失败'))
        .finally(() => { this.subLoading = false })
    },
    openSubEdit(row) {
      if (row) {
        this.subForm = {
          id: row.id,
          name: row.name,
          deviceGBId: row.deviceGBId,
          password: row.password,
          enable: !!row.enable,
          transport: row.transport || 'UDP'
        }
      } else {
        this.subForm = { id: 0, name: '', deviceGBId: '', password: '', enable: true, transport: 'UDP' }
      }
      this.subDialog = true
    },
    saveSub() {
      if (!this.subForm.name || !this.subForm.deviceGBId || !this.subForm.password) {
        this.$message.warning('请填写名称、编号、密码')
        return
      }
      this.subSaving = true
      const req = this.subForm.id
        ? updateSubordinate(this.subForm.id, this.subForm)
        : addSubordinate(this.subForm)
      req.then(() => {
        this.$message.success('保存成功')
        this.subDialog = false
        this.loadSubordinates()
      }).catch(err => this.$message.error(err || '保存失败'))
        .finally(() => { this.subSaving = false })
    },
    deleteSub(row) {
      this.$confirm('确认删除该下级平台？', '提示', { type: 'warning' }).then(() => {
        removeSubordinate(row.id).then(() => {
          this.$message.success('已删除')
          this.loadSubordinates()
        }).catch(err => this.$message.error(err || '删除失败'))
      }).catch(() => {})
    },
    addParentPlatform: function() {
      this.platform = this.defaultPlatform
    },
    editPlatform: function(platform) {
      this.platform = platform
    },
    closeEdit: function() {
      this.platform = null
      this.getPlatformList()
    },
    deletePlatform: function(platform) {
      this.$confirm('确认删除?', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(() => {
        this.deletePlatformCommit(platform)
      })
    },
    deletePlatformCommit: function(platform) {
      this.loading = true
      this.$store.dispatch('platform/remove', platform.id)
        .then(() => {
          this.$message.success({
            showClose: true,
            message: '删除成功'
          })
          this.initData()
        })
        .catch((error) => {
          this.loading = false
          this.$message.error({
            showClose: true,
            message: error
          })
        })
        .finally(() => {
          this.loading = false
        })
    },
    chooseChannel: function(platform) {
      this.$refs.shareChannel.openDialog(platform.id, this.initData)
    },
    pushChannel: function(row) {
      this.pushChannelLoading = true
      this.$store.dispatch('platform/pushChannel', row.id)
        .then((data) => {
          this.$message.success({
            showClose: true,
            message: '推送成功'
          })
        })
        .catch((error) => {
          this.$message.error({
            showClose: true,
            message: error
          })
        })
        .finally(() => {
          this.pushChannelLoading = false
        })
    },
    initData: function() {
      this.$store.dispatch('platform/getServerConfig')
        .then((data) => {
          this.deviceIps = data.deviceIp.split(',')
          this.defaultPlatform = {
            id: null,
            enable: true,
            ptz: true,
            rtcp: false,
            asMessageChannel: false,
            autoPushChannel: false,
            name: null,
            serverGBId: null,
            serverGBDomain: null,
            serverIp: null,
            serverPort: null,
            deviceGBId: data.username,
            deviceIp: this.deviceIps[0],
            devicePort: data.devicePort,
            username: data.username,
            password: data.password,
            expires: 3600,
            keepTimeout: 60,
            transport: 'UDP',
            characterSet: 'GB2312',
            startOfflinePush: false,
            customGroup: false,
            catalogWithPlatform: 0,
            catalogWithGroup: 0,
            catalogWithRegion: 0,
            manufacturer: null,
            model: null,
            address: null,
            secrecy: 1,
            catalogGroup: 1,
            civilCode: null,
            sendStreamIp: data.sendStreamIp
          }
        })
      this.getPlatformList()
    },
    currentChange: function(val) {
      this.currentPage = val
      this.getPlatformList()
    },
    handleSizeChange: function(val) {
      this.count = val
      this.getPlatformList()
    },
    queryList: function() {
      this.currentPage = 1
      this.total = 0
      this.getPlatformList()
    },
    getPlatformList: function() {
      this.$store.dispatch('platform/query', {
        count: this.count,
        page: this.currentPage,
        query: this.searchStr
      })
        .then((data) => {
          this.total = data.total
          this.platformList = data.list
        })
        .catch(function(error) {
          console.log(error)
        })
    },
    refresh: function() {
      this.initData()
    }
  }
}
</script>
<style>
.subscribe-on {
  color: #409EFF;
  font-size: 18px;
}

.subscribe-off {
  color: #afafb3;
  font-size: 18px;
}
</style>
