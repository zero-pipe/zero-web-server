<template>
  <div class="app-container">
    <div style="height: calc(100vh - 124px);">
      <el-form :inline="true" size="mini">
        <el-form-item>
          <el-button icon="el-icon-plus" size="mini" style="margin-right: 1rem;" type="primary" @click="openAdd">
            添加接入密钥
          </el-button>
        </el-form-item>
        <el-form-item style="float: right; color: #909399; font-size: 12px;">
          外部集成方在推/拉流 URL 上携带 <b>?token=xxx</b>；此处启用后，媒体节点按本地白名单校验。
        </el-form-item>
      </el-form>

      <el-table
        size="small"
        :data="list"
        style="width: 100%;font-size: 12px;"
        height="calc(100% - 64px)"
        header-row-class-name="table-header"
      >
        <el-table-column prop="name" label="名称" min-width="160" />
        <el-table-column label="Token" :show-overflow-tooltip="true" min-width="300">
          <template #default="scope">
            <i
              v-clipboard="scope.row.token"
              class="cpoy-btn el-icon-document-copy"
              title="点击拷贝"
              @success="$message({ type: 'success', message: '成功拷贝到剪贴板' })"
            />
            <span style="margin-left: 6px;">{{ scope.row.token }}</span>
          </template>
        </el-table-column>
        <el-table-column label="状态" width="100">
          <template #default="scope">
            <el-tag v-if="scope.row.enabled" size="small">启用</el-tag>
            <el-tag v-else size="small" type="info">停用</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="remark" label="备注" min-width="160" show-overflow-tooltip />
        <el-table-column prop="updateTime" label="更新时间" width="170" />
        <el-table-column label="操作" width="220" fixed="right">
          <template #default="scope">
            <el-button
              v-if="scope.row.enabled"
              size="medium"
              type="text"
              icon="el-icon-circle-close"
              title="停用"
              @click="setEnabled(scope.row, false)"
            />
            <el-button
              v-else
              size="medium"
              type="text"
              icon="el-icon-circle-check"
              title="启用"
              @click="setEnabled(scope.row, true)"
            />
            <el-divider direction="vertical" />
            <el-button size="medium" type="text" icon="el-icon-refresh" title="重置" @click="reset(scope.row)" />
            <el-divider direction="vertical" />
            <el-button size="medium" type="text" icon="el-icon-edit" title="编辑" @click="openEdit(scope.row)" />
            <el-divider direction="vertical" />
            <el-button
              size="medium"
              type="text"
              icon="el-icon-delete"
              style="color: #EA4335"
              title="删除"
              @click="remove(scope.row)"
            />
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

    <el-dialog :title="form.id ? '编辑接入密钥' : '添加接入密钥'" width="520px" :visible.sync="dialogVisible" :close-on-click-modal="false">
      <el-form :model="form" label-width="90px" size="small">
        <el-form-item label="名称" required>
          <el-input v-model="form.name" placeholder="如：南门摄像头供应商" clearable />
        </el-form-item>
        <el-form-item v-if="!form.id" label="Token">
          <el-input v-model="form.token" placeholder="留空自动生成 32 位随机串" clearable />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="form.remark" type="textarea" :rows="2" clearable />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="form.enabled" />
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button size="small" @click="dialogVisible = false">取消</el-button>
        <el-button size="small" type="primary" @click="submit">确定</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import { list, add, update, enable, disable, reset, remove } from '@/api/accessToken'

export default {
  name: 'AccessToken',
  data() {
    return {
      list: [],
      currentPage: 1,
      count: 15,
      total: 0,
      dialogVisible: false,
      form: { id: null, name: '', token: '', remark: '', enabled: true }
    }
  },
  created() {
    this.getList()
  },
  methods: {
    getList() {
      list({ page: this.currentPage, count: this.count }).then(data => {
        this.list = (data && data.list) || []
        this.total = (data && data.total) || 0
      })
    },
    currentChange(val) {
      this.currentPage = val
      this.getList()
    },
    handleSizeChange(val) {
      this.count = val
      this.getList()
    },
    openAdd() {
      this.form = { id: null, name: '', token: '', remark: '', enabled: true }
      this.dialogVisible = true
    },
    openEdit(row) {
      this.form = { id: row.id, name: row.name, token: row.token, remark: row.remark, enabled: row.enabled }
      this.dialogVisible = true
    },
    submit() {
      if (!this.form.name) {
        this.$message.warning('请填写名称')
        return
      }
      const req = this.form.id ? update(this.form) : add(this.form)
      req.then(() => {
        this.dialogVisible = false
        this.$message.success('保存成功')
        this.getList()
      })
    },
    setEnabled(row, val) {
      const action = val ? enable : disable
      action(row.id).then(() => {
        this.$message.success(val ? '已启用' : '已停用')
        this.getList()
      })
    },
    reset(row) {
      this.$confirm('重置后旧 token 立即失效，确定？', '提示', { type: 'warning' }).then(() => {
        reset(row.id).then(data => {
          const token = (data && data.token) || ''
          this.$alert(`新的 Token：${token}`, '重置成功', { dangerouslyUseHTMLString: false })
          this.getList()
        })
      }).catch(() => {})
    },
    remove(row) {
      this.$confirm('确定删除该接入密钥？删除后携带此 token 的推拉流将被拒绝。', '提示', { type: 'warning' }).then(() => {
        remove(row.id).then(() => {
          this.$message.success('已删除')
          this.getList()
        })
      }).catch(() => {})
    }
  }
}
</script>
