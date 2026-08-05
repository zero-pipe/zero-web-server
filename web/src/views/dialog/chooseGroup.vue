<template>
  <div id="chooseGroup">
    <el-dialog
      v-el-drag-dialog
      title="选择虚拟组织"
      width="32%"
      top="5rem"
      :append-to-body="true"
      :close-on-click-modal="false"
      :visible.sync="showDialog"
      :destroy-on-close="true"
      @close="close()"
    >
      <el-alert
        title="通道只能挂在「虚拟组织(216)」上。业务分组(215) 如「1615」需先展开，在其下新建子节点后再选。"
        type="info"
        :closable="false"
        style="margin-bottom: 10px"
      />
      <div v-if="groupName" style="margin-bottom: 10px; color: #409EFF;">
        已选：{{ groupName }}（{{ groupDeviceId }}）
      </div>
      <GroupTree
        ref="regionTree"
        :show-header="true"
        :edit="true"
        :enable-add-channel="false"
        @clickEvent="treeNodeClickEvent"
        :on-channel-change="onChannelChange"
        :tree-height="'45vh'"
      />
      <el-form>
        <el-form-item>
          <div style="text-align: right">
            <el-button type="primary" @click="onSubmit">保存</el-button>
            <el-button @click="close">取消</el-button>
          </div>
        </el-form-item>
      </el-form>
    </el-dialog>
  </div>
</template>

<script>

import elDragDialog from '@/directive/el-drag-dialog'
import GroupTree from '../common/GroupTree.vue'
import { isBusinessGroupNode } from '@/utils/gbCode'

export default {
  name: 'ChooseGroup',
  directives: { elDragDialog },
  components: { GroupTree },
  props: {},
  data() {
    return {
      showDialog: false,
      endCallback: false,
      groupDeviceId: '',
      groupName: '',
      businessGroup: ''
    }
  },
  methods: {
    openDialog: function(callback) {
      this.showDialog = true
      this.endCallback = callback
      this.groupDeviceId = ''
      this.groupName = ''
      this.businessGroup = ''
    },
    onSubmit: function() {
      if (!this.groupDeviceId) {
        this.$message.warning({
          showClose: true,
          message: '未选中有效的虚拟组织(216)。若只有「1615」这类节点，它是业务分组(215)，请先在其下新建虚拟组织子节点。',
          duration: 8000
        })
        return
      }
      if (this.endCallback) {
        this.endCallback(this.groupDeviceId, this.businessGroup, this.groupName)
      }
      this.close()
    },
    close: function() {
      this.showDialog = false
    },
    treeNodeClickEvent: function(group) {
      if (!group.deviceId) {
        return
      }
      if (isBusinessGroupNode(group)) {
        this.groupDeviceId = ''
        this.groupName = ''
        this.businessGroup = ''
        this.$message.warning({
          showClose: true,
          message: `「${group.name}」是业务分组(215)，编号 ${group.deviceId}，不能直接挂通道。请到「通道→业务分组」展开该节点，右键「新建节点」创建虚拟组织(216) 后再来选。`,
          duration: 10000
        })
        return
      }
      this.groupDeviceId = group.deviceId
      this.businessGroup = group.businessGroup
      this.groupName = group.name
      this.$message.success({
        showClose: true,
        message: `已选择虚拟组织：${group.name}`
      })
    },
    onChannelChange: function(deviceId) {
      //
    }
  }
}
</script>
