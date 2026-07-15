<template>
  <el-dialog
    title="编辑设备"
    :visible.sync="innerVisible"
    width="520px"
    :close-on-click-modal="false"
    @close="reset"
  >
    <el-form ref="form" :model="form" :rules="rules" label-width="100px" size="small">
      <el-form-item label="接入模式">
        <el-tag v-if="row && row.accessMode === 'passive'" size="mini" type="success">被动</el-tag>
        <el-tag v-else size="mini" type="warning">主动</el-tag>
        <span style="margin-left: 8px; color: #64748b; font-size: 12px;">
          {{ row && row.protocol === 'gb28181' ? 'GB28181' : 'ONVIF' }}
        </span>
      </el-form-item>
      <el-form-item label="设备名称" prop="name">
        <el-input v-model="form.name" maxlength="64" show-word-limit />
      </el-form-item>
      <el-form-item label="内码">
        <el-input :value="form.internalCode" disabled placeholder="平台自动生成" />
      </el-form-item>
      <el-form-item label="厂商">
        <el-select v-model="form.vendor" clearable placeholder="不限 / 未知" style="width: 100%;">
          <el-option label="海康" value="海康" />
          <el-option label="大华" value="大华" />
          <el-option label="宇视" value="宇视" />
          <el-option label="其他" value="其他" />
        </el-select>
      </el-form-item>

      <template v-if="row && row.protocol === 'gb28181'">
        <el-form-item label="设备编号">
          <el-input :value="form.gb.deviceId" disabled />
        </el-form-item>
        <el-form-item label="注册密码">
          <el-input v-model="form.gb.password" type="password" show-password placeholder="可空则用平台默认" />
        </el-form-item>
        <el-form-item label="收流 IP">
          <el-input v-model="form.gb.sdpIp" />
        </el-form-item>
        <el-form-item label="流媒体节点">
          <el-select v-model="form.gb.mediaServerId" style="width: 100%;">
            <el-option label="自动负载最小" value="auto" />
          </el-select>
        </el-form-item>
        <el-form-item label="字符集">
          <el-select v-model="form.gb.charset" style="width: 100%;">
            <el-option label="GB2312" value="GB2312" />
            <el-option label="UTF-8" value="UTF-8" />
          </el-select>
        </el-form-item>
      </template>

      <template v-if="row && row.protocol === 'onvif'">
        <el-form-item label="国标编号">
          <el-input v-model="form.onvif.gbCode" maxlength="20" show-word-limit placeholder="可空；级联上级前填写" />
        </el-form-item>
        <el-form-item label="IP">
          <el-input :value="form.onvif.ip" disabled />
        </el-form-item>
        <el-form-item label="端口">
          <el-input v-model.number="form.onvif.port" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="form.onvif.username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.onvif.password" type="password" show-password placeholder="不改可留空再填" />
        </el-form-item>
      </template>
    </el-form>
    <div slot="footer">
      <el-button size="small" @click="innerVisible = false">取消</el-button>
      <el-button type="primary" size="small" :loading="submitting" @click="submit">保存</el-button>
    </div>
  </el-dialog>
</template>

<script>
import { updateDevice } from '@/api/devices'

export default {
  name: 'DeviceEdit',
  props: {
    visible: { type: Boolean, default: false },
    row: { type: Object, default: null }
  },
  data() {
    return {
      submitting: false,
      form: {
        name: '',
        vendor: '',
        internalCode: '',
        gb: { deviceId: '', password: '', sdpIp: '', mediaServerId: 'auto', charset: 'GB2312' },
        onvif: { ip: '', port: 80, username: '', password: '', gbCode: '' }
      },
      rules: {
        name: [{ required: true, message: '请输入设备名称', trigger: 'blur' }]
      }
    }
  },
  computed: {
    innerVisible: {
      get() { return this.visible },
      set(v) { this.$emit('update:visible', v) }
    }
  },
  watch: {
    visible(v) {
      if (v && this.row) this.fill(this.row)
    }
  },
  methods: {
    fill(row) {
      this.form.name = row.name || ''
      this.form.vendor = row.vendor || ''
      this.form.internalCode = row.internalCode || ''
      const gb = row.gb || {}
      this.form.gb = {
        deviceId: gb.deviceId || row.rawId || '',
        password: gb.password || '',
        sdpIp: gb.sdpIp || '',
        mediaServerId: gb.mediaServerId || 'auto',
        charset: gb.charset || 'GB2312'
      }
      const ov = row.onvif || {}
      this.form.onvif = {
        ip: ov.ip || '',
        port: ov.port || 80,
        username: ov.username || '',
        password: ov.password || '',
        gbCode: ov.gbCode || ''
      }
    },
    reset() {
      this.form = {
        name: '',
        vendor: '',
        internalCode: '',
        gb: { deviceId: '', password: '', sdpIp: '', mediaServerId: 'auto', charset: 'GB2312' },
        onvif: { ip: '', port: 80, username: '', password: '', gbCode: '' }
      }
    },
    submit() {
      this.$refs.form.validate(valid => {
        if (!valid || !this.row) return
        this.submitting = true
        const payload = {
          name: (this.form.name || '').trim(),
          vendor: this.form.vendor
        }
        if (this.row.protocol === 'gb28181') {
          payload.gb = { ...this.form.gb }
        } else {
          payload.onvif = { ...this.form.onvif }
        }
        updateDevice(this.row.id, payload).then(() => {
          this.$message.success('已保存')
          this.innerVisible = false
          this.$emit('success')
        }).catch(err => {
          this.$message.error(err.message || err || '保存失败')
        }).finally(() => {
          this.submitting = false
        })
      })
    }
  }
}
</script>
