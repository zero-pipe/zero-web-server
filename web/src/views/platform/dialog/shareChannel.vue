<template>
  <div id="chooseChannel">
    <el-dialog v-if="showDialog" v-el-drag-dialog v-loading="loading" title="通道共享" top="2rem" width="80%" :close-on-click-modal="false" :visible.sync="showDialog" :destroy-on-close="true" @close="close()">
      <el-tabs v-model="tab">
        <el-tab-pane label="按通道共享" name="channel">
          <shareChannelAdd ref="shareChannelAdd" :platform-id="platformId" />
        </el-tab-pane>
        <el-tab-pane label="按业务目录共享" name="group" lazy>
          <shareGroup ref="shareGroup" :platform-id="platformId" />
        </el-tab-pane>
      </el-tabs>
    </el-dialog>
  </div>
</template>

<script>

import elDragDialog from '@/directive/el-drag-dialog'
import shareChannelAdd from './shareChannelAdd.vue'
import shareGroup from './shareGroup.vue'

export default {
  name: 'ChooseChannel',
  directives: { elDragDialog },
  components: { shareChannelAdd, shareGroup },
  props: {},
  data() {
    return {
      loading: false,
      tabActiveName: 'gbChannel',
      catalogTabActiveName: 'addShare',
      platformId: '',
      tab: 'channel',
      showDialog: false,
      chooseData: {}
    }
  },
  computed: {},
  methods: {
    openDialog(platformId, closeCallback) {
      this.platformId = platformId
      this.showDialog = true
      this.closeCallback = closeCallback
    },
    close: function() {
      this.closeCallback()
    }
  }
}
</script>

<style>

</style>
