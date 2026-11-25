<template>
  <div class="app-container">
    <el-row :gutter="16">
      <!-- 左：字典类型 -->
      <el-col :span="6">
        <el-card shadow="never">
          <div slot="header" class="pane-header">
            <span>字典类型</span>
            <el-button type="primary" size="mini" icon="el-icon-plus" @click="openTypeDialog()">新增类型</el-button>
          </div>
          <div v-loading="typeLoading" class="type-list">
            <div
              v-for="t in types"
              :key="t.code"
              :class="['type-item', { active: current && current.code === t.code }]"
              @click="selectType(t)"
            >
              <div class="type-name">
                {{ t.name }}
                <el-tag v-if="t.builtin" size="mini" type="info">内置</el-tag>
              </div>
              <div class="type-code">{{ t.code }}</div>
              <div class="type-actions" @click.stop>
                <el-button type="text" size="mini" icon="el-icon-edit" @click="openTypeDialog(t)" />
                <el-button
                  type="text"
                  size="mini"
                  icon="el-icon-delete"
                  style="color:#EA4335"
                  :disabled="t.builtin"
                  @click="removeType(t)"
                />
              </div>
            </div>
            <div v-if="!types.length" class="empty">暂无类型</div>
          </div>
        </el-card>
      </el-col>

      <!-- 右：字典项 -->
      <el-col :span="18">
        <el-card shadow="never">
          <div slot="header" class="pane-header">
            <span>
              字典项
              <span v-if="current" class="sub">{{ current.name }} ({{ current.code }})</span>
            </span>
            <span>
              <el-button size="mini" icon="el-icon-refresh" @click="loadItems" :disabled="!current">刷新</el-button>
              <el-button type="primary" size="mini" icon="el-icon-plus" :disabled="!current" @click="openItemDialog()">新增项</el-button>
            </span>
          </div>
          <el-table v-loading="itemLoading" :data="items" size="small" border>
            <el-table-column prop="value" label="值(存库)" min-width="140" />
            <el-table-column prop="label" label="显示文本" min-width="140" />
            <el-table-column prop="sort" label="排序" width="80" align="center" />
            <el-table-column label="启用" width="90" align="center">
              <template v-slot:default="{ row }">
                <el-switch :value="row.enabled" @change="(v) => toggleEnabled(row, v)" />
              </template>
            </el-table-column>
            <el-table-column label="内置" width="80" align="center">
              <template v-slot:default="{ row }">
                <el-tag v-if="row.builtin" size="mini" type="info">内置</el-tag>
                <span v-else>—</span>
              </template>
            </el-table-column>
            <el-table-column prop="remark" label="备注" min-width="140" show-overflow-tooltip />
            <el-table-column label="操作" width="150" fixed="right">
              <template v-slot:default="{ row }">
                <el-button type="text" size="mini" icon="el-icon-edit" @click="openItemDialog(row)">编辑</el-button>
                <el-button
                  type="text"
                  size="mini"
                  icon="el-icon-delete"
                  style="color:#EA4335"
                  :disabled="row.builtin"
                  @click="removeItem(row)"
                >删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-card>
      </el-col>
    </el-row>

    <!-- 类型编辑 -->
    <el-dialog :title="typeForm.id ? '编辑类型' : '新增类型'" :visible.sync="typeDialog" width="460px">
      <el-form ref="typeForm" :model="typeForm" label-width="90px" size="small">
        <el-form-item label="类型码">
          <el-input v-model="typeForm.code" :disabled="!!typeForm.id || typeForm.builtin" placeholder="如 device_type" />
        </el-form-item>
        <el-form-item label="显示名">
          <el-input v-model="typeForm.name" placeholder="如 设备类型" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="typeForm.sort" :min="0" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="typeForm.remark" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button size="small" @click="typeDialog = false">取消</el-button>
        <el-button type="primary" size="small" :loading="saving" @click="saveType">保存</el-button>
      </div>
    </el-dialog>

    <!-- 项编辑 -->
    <el-dialog :title="itemForm.id ? '编辑字典项' : '新增字典项'" :visible.sync="itemDialog" width="460px">
      <el-form ref="itemForm" :model="itemForm" label-width="90px" size="small">
        <el-form-item label="值(存库)">
          <el-input v-model="itemForm.value" :disabled="!!itemForm.id" placeholder="如 thermal" />
        </el-form-item>
        <el-form-item label="显示文本">
          <el-input v-model="itemForm.label" placeholder="如 热成像" />
        </el-form-item>
        <el-form-item label="排序">
          <el-input-number v-model="itemForm.sort" :min="0" />
        </el-form-item>
        <el-form-item label="启用">
          <el-switch v-model="itemForm.enabled" />
        </el-form-item>
        <el-form-item label="备注">
          <el-input v-model="itemForm.remark" />
        </el-form-item>
      </el-form>
      <div slot="footer">
        <el-button size="small" @click="itemDialog = false">取消</el-button>
        <el-button type="primary" size="small" :loading="saving" @click="saveItem">保存</el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import {
  listDictTypes, createDictType, updateDictType, deleteDictType,
  listDictItems, createDictItem, updateDictItem, deleteDictItem
} from '@/api/dict'

const emptyType = () => ({ id: null, code: '', name: '', sort: 0, remark: '', builtin: false })
const emptyItem = () => ({ id: null, value: '', label: '', sort: 0, enabled: true, remark: '', builtin: false })

export default {
  name: 'DictManage',
  data() {
    return {
      typeLoading: false,
      itemLoading: false,
      saving: false,
      types: [],
      current: null,
      items: [],
      typeDialog: false,
      typeForm: emptyType(),
      itemDialog: false,
      itemForm: emptyItem()
    }
  },
  created() {
    this.loadTypes()
  },
  methods: {
    loadTypes() {
      this.typeLoading = true
      listDictTypes().then(res => {
        this.types = (res.data && res.data.list) || []
        if (!this.current && this.types.length) {
          this.selectType(this.types[0])
        } else if (this.current) {
          const hit = this.types.find(t => t.code === this.current.code)
          if (hit) this.current = hit
        }
      }).finally(() => { this.typeLoading = false })
    },
    selectType(t) {
      this.current = t
      this.loadItems()
    },
    loadItems() {
      if (!this.current) return
      this.itemLoading = true
      listDictItems(this.current.code, false).then(res => {
        this.items = (res.data && res.data.list) || []
      }).finally(() => { this.itemLoading = false })
    },
    openTypeDialog(t) {
      this.typeForm = t ? { ...t } : emptyType()
      this.typeDialog = true
    },
    saveType() {
      const f = this.typeForm
      if (!f.code) { this.$message.warning('请填写类型码'); return }
      this.saving = true
      const req = f.id
        ? updateDictType(f.id, { name: f.name, remark: f.remark, sort: f.sort })
        : createDictType({ code: f.code, name: f.name, remark: f.remark, sort: f.sort })
      req.then(() => {
        this.$message.success('已保存')
        this.typeDialog = false
        this.loadTypes()
      }).catch(e => this.$message.error((e && e.message) || '保存失败')).finally(() => { this.saving = false })
    },
    removeType(t) {
      this.$confirm(`删除类型「${t.name}」及其所有项？`, '提示', { type: 'warning' }).then(() => {
        deleteDictType(t.id).then(() => {
          this.$message.success('已删除')
          if (this.current && this.current.code === t.code) { this.current = null; this.items = [] }
          this.loadTypes()
        }).catch(e => this.$message.error((e && e.message) || '删除失败'))
      }).catch(() => {})
    },
    openItemDialog(row) {
      this.itemForm = row ? { ...row } : emptyItem()
      this.itemDialog = true
    },
    saveItem() {
      const f = this.itemForm
      if (!f.value) { this.$message.warning('请填写值'); return }
      this.saving = true
      const req = f.id
        ? updateDictItem(f.id, { label: f.label, sort: f.sort, enabled: f.enabled, remark: f.remark })
        : createDictItem({ typeCode: this.current.code, value: f.value, label: f.label, sort: f.sort, enabled: f.enabled, remark: f.remark })
      req.then(() => {
        this.$message.success('已保存')
        this.itemDialog = false
        this.loadItems()
      }).catch(e => this.$message.error((e && e.message) || '保存失败')).finally(() => { this.saving = false })
    },
    removeItem(row) {
      this.$confirm(`删除字典项「${row.label}」？`, '提示', { type: 'warning' }).then(() => {
        deleteDictItem(row.id).then(() => {
          this.$message.success('已删除')
          this.loadItems()
        }).catch(e => this.$message.error((e && e.message) || '删除失败'))
      }).catch(() => {})
    },
    toggleEnabled(row, val) {
      updateDictItem(row.id, { label: row.label, sort: row.sort, enabled: val, remark: row.remark })
        .then(() => { row.enabled = val })
        .catch(e => this.$message.error((e && e.message) || '操作失败'))
    }
  }
}
</script>

<style scoped>
.pane-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.pane-header .sub {
  color: #909399;
  font-size: 12px;
  margin-left: 6px;
}
.type-list {
  min-height: 200px;
}
.type-item {
  position: relative;
  padding: 8px 10px;
  border-radius: 4px;
  cursor: pointer;
  margin-bottom: 4px;
  border: 1px solid transparent;
}
.type-item:hover {
  background: #f5f7fa;
}
.type-item.active {
  background: #ecf5ff;
  border-color: #b3d8ff;
}
.type-name {
  font-size: 13px;
  font-weight: 500;
}
.type-code {
  font-size: 12px;
  color: #909399;
}
.type-actions {
  position: absolute;
  right: 6px;
  top: 8px;
}
.empty {
  color: #909399;
  text-align: center;
  padding: 24px 0;
}
</style>
