<template>
  <div id="ChannelEdit" v-loading="locading" style="width: 100%">
    <div class="page-header">
      <div class="page-title">
        <el-page-header content="编辑通道" @back="close" />
      </div>
    </div>
    <CommonChannelEdit
      :id="id"
      :internal-code="internalCode"
      :protocol="protocol"
      ref="commonChannelEdit"
      :showCancel="true"
      @submitSuccess="close"
      @cancel="close"
    />
  </div>
</template>

<script>
import CommonChannelEdit from '../../common/CommonChannelEdit'

export default {
  name: 'ChannelEdit',
  components: {
    CommonChannelEdit
  },
  // id：国标通道主键；internalCode + protocol：统一通道列表进入时使用
  props: {
    id: { type: [Number, String], default: null },
    internalCode: { type: String, default: '' },
    protocol: { type: String, default: 'gb28181' },
    closeEdit: { type: Function, required: true }
  },
  data() {
    return {}
  },
  methods: {
    close: function() {
      this.closeEdit()
    }
  }
}
</script>
