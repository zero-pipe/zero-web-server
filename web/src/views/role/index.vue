<template>
  <div class="app-container">
    <div style="height: calc(100vh - 124px);">
      <el-form :inline="true" size="mini">
        <el-form-item>
          <el-button icon="el-icon-plus" size="mini" type="primary" @click="openAdd">添加角色</el-button>
        </el-form-item>
      </el-form>
      <el-table
        size="small"
        :data="roleList"
        style="width: 100%;font-size: 12px;"
        height="calc(100% - 64px)"
        header-row-class-name="table-header"
      >
        <el-table-column prop="id" label="ID" width="80" />
        <el-table-column prop="name" label="角色名称" min-width="140" />
        <el-table-column label="菜单权限" min-width="360">
          <template v-slot:default="scope">
            <el-tag
              v-for="m in menuLabels(scope.row)"
              :key="m"
              size="mini"
              style="margin: 2px;"
            >{{ m }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column prop="updateTime" label="更新时间" min-width="160" />
        <el-table-column label="操作" min-width="180" fixed="right">
          <template v-slot:default="scope">
            <el-button size="medium" type="text" icon="el-icon-edit" @click="openEdit(scope.row)">编辑</el-button>
            <el-divider direction="vertical" />
            <el-button
              size="medium"
              type="text"
              icon="el-icon-delete"
              style="color: #f56c6c"
              :disabled="scope.row.id === 1"
              @click="remove(scope.row)"
            >删除</el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>

    <el-dialog
      :title="dialogTitle"
      :visible.sync="showDialog"
      width="520px"
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <el-form label-width="90px" size="small">
        <el-form-item label="角色名称" required>
          <el-input v-model="form.name" maxlength="50" placeholder="请输入角色名称" />
        </el-form-item>
        <el-form-item label="菜单权限">
          <el-checkbox
            v-model="checkAll"
            :indeterminate="indeterminate"
            :disabled="form.id === 1"
            @change="onCheckAll"
          >全选</el-checkbox>
          <div style="margin-top: 8px;">
            <el-checkbox-group v-model="form.menus" :disabled="form.id === 1" @change="onMenusChange">
              <el-checkbox
                v-for="item in menuOptions"
                :key="item.code"
                :label="item.code"
                style="display:block; margin: 6px 0;"
              >{{ item.title }}</el-checkbox>
            </el-checkbox-group>
          </div>
          <div v-if="form.id === 1" style="color:#909399;font-size:12px;margin-top:6px;">
            管理员角色固定拥有全部菜单权限
          </div>
        </el-form-item>
      </el-form>
      <span slot="footer">
        <el-button size="small" @click="showDialog = false">取消</el-button>
        <el-button type="primary" size="small" :loading="saving" @click="save">保存</el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import { parseAuthorityMenus } from '@/utils/permission'

export default {
  name: 'RoleManage',
  data() {
    return {
      roleList: [],
      menuOptions: [],
      showDialog: false,
      saving: false,
      form: {
        id: 0,
        name: '',
        menus: []
      },
      checkAll: false,
      indeterminate: false
    }
  },
  computed: {
    dialogTitle() {
      return this.form.id ? '编辑角色' : '添加角色'
    }
  },
  mounted() {
    this.loadMenus()
    this.loadRoles()
  },
  methods: {
    loadMenus() {
      this.$store.dispatch('role/getMenus').then(data => {
        this.menuOptions = data || []
      })
    },
    loadRoles() {
      this.$store.dispatch('role/getAll').then(data => {
        this.roleList = data || []
      })
    },
    menuLabels(row) {
      const codes = parseAuthorityMenus(row.authority, row.id)
      const map = {}
      ;(this.menuOptions || []).forEach(m => { map[m.code] = m.title })
      return codes.map(c => map[c] || c)
    },
    openAdd() {
      this.form = { id: 0, name: '', menus: [] }
      this.onMenusChange(this.form.menus)
      this.showDialog = true
    },
    openEdit(row) {
      this.form = {
        id: row.id,
        name: row.name,
        menus: parseAuthorityMenus(row.authority, row.id)
      }
      this.onMenusChange(this.form.menus)
      this.showDialog = true
    },
    resetForm() {
      this.form = { id: 0, name: '', menus: [] }
      this.checkAll = false
      this.indeterminate = false
    },
    onCheckAll(val) {
      this.form.menus = val ? this.menuOptions.map(m => m.code) : []
      this.indeterminate = false
    },
    onMenusChange(val) {
      const total = (this.menuOptions || []).length
      this.checkAll = total > 0 && val.length === total
      this.indeterminate = val.length > 0 && val.length < total
    },
    save() {
      if (!this.form.name || !this.form.name.trim()) {
        this.$message.warning('请输入角色名称')
        return
      }
      this.saving = true
      const payload = {
        id: this.form.id,
        name: this.form.name.trim(),
        menus: this.form.id === 1 ? this.menuOptions.map(m => m.code) : this.form.menus
      }
      const action = this.form.id ? 'role/update' : 'role/add'
      this.$store.dispatch(action, payload).then(() => {
        this.$message.success('保存成功')
        this.showDialog = false
        this.loadRoles()
      }).catch(err => {
        this.$message.error(err || '保存失败')
      }).finally(() => {
        this.saving = false
      })
    },
    remove(row) {
      this.$confirm(`确认删除角色「${row.name}」？`, '提示', { type: 'warning' }).then(() => {
        return this.$store.dispatch('role/remove', row.id)
      }).then(() => {
        this.$message.success('删除成功')
        this.loadRoles()
      }).catch(() => {})
    }
  }
}
</script>
