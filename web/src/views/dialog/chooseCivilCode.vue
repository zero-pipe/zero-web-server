<template>
  <div id="chooseCivilCode">
    <el-dialog
      v-el-drag-dialog
      title="选择行政区划"
      width="30%"
      top="5rem"
      :close-on-click-modal="false"
      :visible.sync="showDialog"
      :destroy-on-close="true"
      @close="close()"
    >
      <el-alert
        title="在下方树中点选一个行政区划节点，再点保存。若树为空，请先到「行政区划」页面创建，或关闭本窗留空。"
        type="info"
        :closable="false"
        show-icon
        style="margin-bottom: 12px"
      />
      <RegionTree
        ref="regionTree"
        :show-header="true"
        :edit="false"
        :enable-add-channel="false"
        @clickEvent="treeNodeClickEvent"
        :on-channel-change="onChannelChange"
        :tree-height="'45vh'"
      />
      <div v-if="regionDeviceId" style="margin: 8px 0; color: #606266; font-size: 13px;">
        已选：{{ regionName }}（{{ regionDeviceId }}）
      </div>
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
import RegionTree from '../common/RegionTree.vue'

export default {
  name: 'ChooseCivilCode',
  directives: { elDragDialog },
  components: { RegionTree },
  props: {},
  data() {
    return {
      showDialog: false,
      endCallback: false,
      regionDeviceId: '',
      regionName: ''
    }
  },
  computed: {},
  created() {},
  methods: {
    openDialog: function(callback) {
      this.showDialog = true
      this.endCallback = callback
      this.regionDeviceId = ''
      this.regionName = ''
    },
    onSubmit: function() {
      if (this.endCallback) {
        this.endCallback(this.regionDeviceId, this.regionName)
      }
      this.close()
    },
    close: function() {
      this.showDialog = false
    },
    treeNodeClickEvent: function(region) {
      this.regionDeviceId = region.deviceId
      this.regionName = region.name
    },
    onChannelChange: function(deviceId) {
      //
    }
  }
}
</script>
