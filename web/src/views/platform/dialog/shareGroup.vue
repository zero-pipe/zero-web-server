<template>
  <div id="shareGroup" style="background-color:#fff; padding: 0 1rem; min-height: 480px;">
    <el-alert
      type="info"
      :closable="false"
      style="margin-bottom: 12px;"
      title="按业务目录共享：勾选业务分组(215)或虚拟组织(216)，节点下的通道会随目录动态共享（新增通道自动跟随）。"
    />
    <div style="display:grid; grid-template-columns: minmax(0,1fr) 320px; gap:16px;">
      <el-tree
        ref="tree"
        :props="treeProps"
        :load="loadNode"
        lazy
        show-checkbox
        check-strictly
        node-key="deviceId"
        empty-text="暂无业务分组，请先在「通道管理 → 业务分组」中创建"
        @check="onCheck"
      >
        <span slot-scope="{ node, data }" class="custom-tree-node">
          <span>{{ data.name || data.deviceId }}</span>
          <span style="color:#80868B; margin-left:8px; font-size:12px;">{{ data.deviceId }}</span>
          <el-tag v-if="data.typeCode === '215'" size="mini" style="margin-left:6px;">业务分组</el-tag>
          <el-tag v-else-if="data.typeCode === '216'" size="mini" type="success" style="margin-left:6px;">虚拟组织</el-tag>
        </span>
      </el-tree>
      <div>
        <div style="font-weight:600; margin-bottom:8px;">已共享节点</div>
        <el-tag
          v-for="id in sharedList"
          :key="id"
          closable
          size="small"
          style="margin: 0 6px 6px 0;"
          @close="unshare(id)"
        >{{ id }}</el-tag>
        <div v-if="sharedList.length === 0" style="color:#80868B; font-size:12px;">尚未共享任何业务目录节点</div>
        <div style="margin-top:16px; color:#80868B; font-size:12px; line-height:1.7;">
          勾选 = 共享该节点及其子树；取消勾选 = 取消共享。<br>
          取消共享后，下次推送会给上级发送该节点下通道的删除通知。
        </div>
      </div>
    </div>
    <div style="margin-top:16px; text-align:right;">
      <el-button size="small" @click="loadShared">重置</el-button>
      <el-button size="small" type="primary" @click="save">保存共享</el-button>
    </div>
  </div>
</template>

<script>
import { getTreeList } from '@/api/group'
import { groupShareList, groupShareAdd, groupShareRemove } from '@/api/platform'

export default {
  name: 'ShareGroup',
  props: {
    platformId: { type: [Number, String], required: true }
  },
  data() {
    return {
      sharedList: [],
      treeProps: {
        label: 'name'
      }
    }
  },
  created() {
    this.loadShared()
  },
  methods: {
    async loadShared() {
      const res = await groupShareList(this.platformId)
      this.sharedList = res.data || []
      this.$nextTick(() => this.syncChecked())
    },
    syncChecked() {
      const tree = this.$refs.tree
      if (!tree) return
      tree.setCheckedKeys(this.sharedList, false)
    },
    async loadNode(node, resolve) {
      const parent = node.level === 0 ? undefined : node.data.id
      try {
        const res = await getTreeList({ parent, hasChannel: true })
        const list = (res.data || []).map(item => ({
          ...item,
          typeCode: item.deviceId && item.deviceId.length === 20 ? item.deviceId.substring(10, 13) : ''
        }))
        resolve(list)
        if (node.level === 0 && this.sharedList.length) {
          this.$nextTick(() => this.syncChecked())
        }
      } catch (e) {
        resolve([])
      }
    },
    onCheck() {
      // 勾选变化只收集，保存时统一提交
    },
    async save() {
      const checked = this.$refs.tree.getCheckedKeys()
      const added = checked.filter(id => !this.sharedList.includes(id))
      const removed = this.sharedList.filter(id => !checked.includes(id))
      if (added.length) await groupShareAdd(this.platformId, added)
      if (removed.length) await groupShareRemove(this.platformId, removed)
      await this.loadShared()
      this.$message.success('业务目录共享已更新')
    },
    async unshare(id) {
      await groupShareRemove(this.platformId, [id])
      await this.loadShared()
    }
  }
}
</script>

<style scoped>
.custom-tree-node {
  display: flex;
  align-items: center;
  font-size: 14px;
}
</style>
