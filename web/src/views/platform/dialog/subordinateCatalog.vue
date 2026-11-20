<template>
  <el-dialog
    v-el-drag-dialog
    :title="`下级目录 - ${name}`"
    top="4rem"
    width="70%"
    :visible.sync="showDialog"
    :destroy-on-close="true"
    @close="close"
  >
    <div v-loading="loading" style="min-height: 360px;">
      <el-alert
        v-if="orphanCount > 0"
        type="warning"
        :closable="false"
        style="margin-bottom: 12px;"
        :title="`有 ${orphanCount} 个目录项父节点不存在，已暂不挂载（可在“全部”页签查看）`"
      />
      <el-tabs v-model="tab">
        <el-tab-pane label="目录树" name="tree">
          <el-tree
            :data="tree"
            :props="treeProps"
            node-key="deviceId"
            default-expand-all
            empty-text="暂无目录数据，请先让下级推送目录"
          >
            <span slot-scope="{ data }" class="cat-node">
              <el-tag v-if="data.itemType === 'channel'" size="mini" type="success">点位</el-tag>
              <el-tag v-else size="mini">目录</el-tag>
              <span style="margin: 0 8px;">{{ data.name }}</span>
              <span style="color:#80868B; font-size:12px;">{{ data.deviceId }}</span>
            </span>
          </el-tree>
        </el-tab-pane>
        <el-tab-pane label="全部" name="flat">
          <el-table size="small" :data="flat" height="420">
            <el-table-column prop="name" label="名称" min-width="160" />
            <el-table-column prop="deviceId" label="国标编号" min-width="200" />
            <el-table-column label="类型" width="80">
              <template v-slot:default="scope">
                <el-tag size="mini" :type="scope.row.itemType === 'channel' ? 'success' : ''">
                  {{ scope.row.itemType === 'channel' ? '点位' : '目录' }}
                </el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="parentId" label="父节点" min-width="200" />
            <el-table-column prop="status" label="状态" width="90" />
            <el-table-column label="挂载" width="90">
              <template v-slot:default="scope">
                <el-tag v-if="scope.row.orphan" size="mini" type="danger">未挂载</el-tag>
                <span v-else>—</span>
              </template>
            </el-table-column>
          </el-table>
        </el-tab-pane>
      </el-tabs>
    </div>
  </el-dialog>
</template>

<script>
import elDragDialog from '@/directive/el-drag-dialog'
import { subordinateCatalogTree, subordinateCatalog } from '@/api/subordinate'

export default {
  name: 'SubordinateCatalog',
  directives: { elDragDialog },
  data() {
    return {
      showDialog: false,
      loading: false,
      id: 0,
      name: '',
      tab: 'tree',
      tree: [],
      flat: [],
      orphanCount: 0,
      treeProps: { label: 'name', children: 'children' }
    }
  },
  methods: {
    openDialog(id, name) {
      this.id = id
      this.name = name
      this.tab = 'tree'
      this.showDialog = true
      this.load()
    },
    async load() {
      this.loading = true
      try {
        const [treeRes, flatRes] = await Promise.all([
          subordinateCatalogTree(this.id),
          subordinateCatalog(this.id, {})
        ])
        this.tree = treeRes.data || []
        this.flat = flatRes.data || []
        this.orphanCount = this.flat.filter(x => x.orphan).length
      } finally {
        this.loading = false
      }
    },
    close() {
      this.showDialog = false
    }
  }
}
</script>

<style scoped>
.cat-node {
  display: flex;
  align-items: center;
  font-size: 14px;
}
</style>
