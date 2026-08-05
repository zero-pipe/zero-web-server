<template>
  <div id="mediaServerEdit" v-loading="isLoging">
    <el-dialog
      v-el-drag-dialog
      title="媒体节点"
      :width="dialogWidth"
      top="2rem"
      :close-on-click-modal="false"
      :visible.sync="showDialog"
      :destroy-on-close="true"
      @close="close()"
    >
      <div id="formStep" style="margin-top: 1rem; margin-right: 20px;">
        <el-form v-if="currentStep === 1" ref="mediaServerForm" :rules="rules" :model="mediaServerForm" label-width="140px">
          <el-form-item label="节点 ID" prop="id">
            <el-input
              v-model="mediaServerForm.id"
              placeholder="如 1（mediaServerId，Hook/调度用；留空则用 IP:端口）"
              clearable
              :disabled="!!editingId"
            />
          </el-form-item>
          <el-form-item label="IP" prop="ip">
            <el-input v-model="mediaServerForm.ip" placeholder="媒体服务IP" clearable />
          </el-form-item>
          <el-form-item label="HTTP端口" prop="httpPort">
            <el-input v-model="mediaServerForm.httpPort" placeholder="媒体服务HTTP端口" clearable />
          </el-form-item>
          <el-form-item label="SECRET" prop="secret">
            <el-input v-model="mediaServerForm.secret" placeholder="媒体服务密钥" clearable />
          </el-form-item>
          <el-form-item label="类型" prop="type">
            <el-select v-model="mediaServerForm.type" style="float: left; width: 100%">
              <el-option key="zms" label="zero-media-server" value="zms" />
              <el-option key="abl" label="ABLMediaServer" value="abl" />
            </el-select>
          </el-form-item>
          <el-form-item>
            <div style="float: right;">
              <el-button v-if="currentStep === 1 && serverCheck === 1" type="primary" @click="next">下一步</el-button>
              <el-button @click="close">取消</el-button>
              <el-button type="primary" @click="checkServer">测试</el-button>
              <i v-if="serverCheck === 1" class="el-icon-success" style="color: #3caf36" />
              <i v-if="serverCheck === -1" class="el-icon-error" style="color: #c80000" />
            </div>
          </el-form-item>
        </el-form>
        <el-row :gutter="24">
          <el-col :span="12">
            <el-form v-if="currentStep === 2 || currentStep === 3" ref="mediaServerForm1" :rules="rules" :model="mediaServerForm" label-width="140px">
              <el-form-item label="节点 ID" prop="id">
                <el-input v-model="mediaServerForm.id" :disabled="!!editingId" placeholder="mediaServerId，留空则用 IP:端口" />
              </el-form-item>
              <el-form-item label="IP" prop="ip">
                <el-input v-if="currentStep === 2" v-model="mediaServerForm.ip" />
                <el-input v-if="currentStep === 3" v-model="mediaServerForm.ip" />
              </el-form-item>
              <el-form-item label="HTTP端口" prop="httpPort">
                <el-input v-if="currentStep === 2" v-model="mediaServerForm.httpPort" />
                <el-input v-if="currentStep === 3" v-model="mediaServerForm.httpPort" />
              </el-form-item>
              <el-form-item label="HOOK IP" prop="ip">
                <el-input v-model="mediaServerForm.hookIp" placeholder="请输入HOOK_IP" clearable />
              </el-form-item>
              <el-form-item label="SDP IP" prop="ip">
                <el-input v-model="mediaServerForm.sdpIp" placeholder="请输入SDP_IP" clearable />
              </el-form-item>
              <el-form-item label="流IP" prop="ip">
                <el-input v-model="mediaServerForm.streamIp" placeholder="请输入流IP" clearable />
              </el-form-item>
              <el-form-item label="HTTPS PORT" prop="httpSSlPort">
                <el-input v-model="mediaServerForm.httpSSlPort" placeholder="请输入HTTPS_PORT" clearable />
              </el-form-item>
              <el-form-item label="RTSP PORT" prop="rtspPort">
                <el-input v-model="mediaServerForm.rtspPort" placeholder="请输入RTSP_PORT" clearable />
              </el-form-item>
              <el-form-item label="RTSPS PORT" prop="rtspSSLPort">
                <el-input v-model="mediaServerForm.rtspSSLPort" placeholder="请输入RTSPS_PORT" clearable />
              </el-form-item>
            </el-form>
          </el-col>
          <el-col :span="12">
            <el-form v-if="currentStep === 2 || currentStep === 3" ref="mediaServerForm2" :rules="rules" :model="mediaServerForm" label-width="180px">
              <el-form-item label="RTMP PORT" prop="rtmpPort">
                <el-input v-model="mediaServerForm.rtmpPort" placeholder="请输入RTMP_PORT" clearable />
              </el-form-item>
              <el-form-item label="RTMPS PORT" prop="rtmpSSlPort">
                <el-input v-model="mediaServerForm.rtmpSSlPort" placeholder="请输入RTMPS_PORT" clearable />
              </el-form-item>
              <el-form-item label="SECRET" prop="secret">
                <el-input v-if="currentStep === 2" v-model="mediaServerForm.secret" />
                <el-input v-if="currentStep === 3" v-model="mediaServerForm.secret" />
              </el-form-item>
              <el-form-item label="自动配置媒体服务">
                <el-switch v-model="mediaServerForm.autoConfig" />
              </el-form-item>
              <el-form-item label="收流端口模式">
                <el-switch v-model="mediaServerForm.rtpEnable" active-text="多端口" inactive-text="单端口" @change="portRangeChange" />
              </el-form-item>
              <el-form-item v-if="!mediaServerForm.rtpEnable" label="收流端口" prop="rtpProxyPort">
                <el-input v-model.number="mediaServerForm.rtpProxyPort" clearable />
              </el-form-item>
              <el-form-item v-if="mediaServerForm.rtpEnable" label="端口范围">
                <el-input v-model="rtpPortRange1" placeholder="起" clearable style="width: 100px" prop="rtpPortRange1" @change="portRangeChange" />
                -
                <el-input v-model="rtpPortRange2" placeholder="止" clearable style="width: 100px" prop="rtpPortRange2" @change="portRangeChange" />
              </el-form-item>
              <el-form-item v-if="mediaServerForm.sendRtpEnable" label="发流端口">
                <el-input v-model="sendRtpPortRange1" placeholder="起" clearable style="width: 100px" prop="rtpPortRange1" @change="portRangeChange" />
                -
                <el-input v-model="sendRtpPortRange2" placeholder="止" clearable style="width: 100px" prop="rtpPortRange2" @change="portRangeChange" />
              </el-form-item>
              <el-form-item>
                <div style="float: right;">
                  <el-button type="primary" @click="onSubmit">提交</el-button>
                  <el-button @click="close">取消</el-button>
                </div>
              </el-form-item>
            </el-form>
          </el-col>
        </el-row>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import elDragDialog from '@/directive/el-drag-dialog'

export default {
  name: 'MediaServerEdit',
  directives: { elDragDialog },
  props: {},
  data() {
    const isValidIp = (rule, value, callback) => {
      var reg = /^(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])\.(\d{1,2}|1\d\d|2[0-4]\d|25[0-5])$/
      if (!reg.test(this.mediaServerForm.ip)) {
        return callback(new Error("请输入正确的IP地址"))
      }
      callback()
    }
    const isValidPort = (rule, value, callback) => {
      var reg = /^(([0-9]|[1-9]\d{1,3}|[1-5]\d{4}|6[0-5]{2}[0-3][0-5]))$/
      if (!reg.test(String(this.mediaServerForm.httpPort))) {
        return callback(new Error("请输入正确的端口号"))
      }
      callback()
    }
    return {
      dialogWidth: 0,
      defaultWidth: 1000,
      listChangeCallback: null,
      showDialog: false,
      isLoging: false,
      dialogLoading: false,
      currentStep: 1,
      editingId: '',
      platformList: [],
      serverCheck: 0,
      recordServerCheck: 0,
      mediaServerForm: {
        id: '',
        ip: '',
        autoConfig: true,
        hookIp: '',
        sdpIp: '',
        streamIp: '',
        secret: '',
        httpPort: '',
        httpSSlPort: '',
        recordAssistPort: '',
        rtmpPort: '',
        rtmpSSlPort: '',
        rtpEnable: false,
        rtpPortRange: '',
        sendRtpPortRange: '',
        rtpProxyPort: '',
        rtspPort: '',
        rtspSSLPort: '',
        type: 'zms'
      },
      rtpPortRange1: 30000,
      rtpPortRange2: 30500,
      sendRtpPortRange1: 50000,
      sendRtpPortRange2: 60000,
      rules: {
        ip: [{ required: true, validator: isValidIp, message: "请输入正确的IP地址", trigger: 'blur' }],
        httpPort: [{ required: true, validator: isValidPort, message: "请输入正确的端口号", trigger: 'blur' }],
        httpSSlPort: [{ required: true, validator: isValidPort, message: "请输入正确的端口号", trigger: 'blur' }],
        recordAssistPort: [{ required: true, validator: isValidPort, message: "请输入正确的端口号", trigger: 'blur' }],
        rtmpPort: [{ required: true, validator: isValidPort, message: "请输入正确的端口号", trigger: 'blur' }],
        rtmpSSlPort: [{ required: true, validator: isValidPort, message: "请输入正确的端口号", trigger: 'blur' }],
        rtpPortRange1: [{ required: true, validator: isValidPort, message: "请输入正确的端口号", trigger: 'blur' }],
        rtpPortRange2: [{ required: true, validator: isValidPort, message: "请输入正确的端口号", trigger: 'blur' }],
        rtpProxyPort: [{ required: true, validator: isValidPort, message: "请输入正确的端口号", trigger: 'blur' }],
        rtspPort: [{ required: true, validator: isValidPort, message: "请输入正确的端口号", trigger: 'blur' }],
        rtspSSLPort: [{ required: true, validator: isValidPort, message: "请输入正确的端口号", trigger: 'blur' }],
        secret: [{ required: false, message: "请输入secret", trigger: 'blur' }],
        timeout_ms: [{ required: true, message: "请输入FFmpeg推流的超时时间", trigger: 'blur' }],
        ffmpeg_cmd_key: [{ required: false, message: "请输入FFmpeg推流命令模板关键字", trigger: 'blur' }]
      }
    }
  },
  created() {
    this.setDialogWidth()
  },
  methods: {
    setDialogWidth() {
      const val = document.body.clientWidth
      if (val < this.defaultWidth) {
        this.dialogWidth = '100%'
      } else {
        this.dialogWidth = this.defaultWidth + 'px'
      }
    },
    openDialog(param, callback) {
      this.showDialog = true
      this.listChangeCallback = callback
      this.editingId = (param && param.id) || ''
      if (param != null) {
        this.mediaServerForm = Object.assign({}, this.mediaServerForm, param)
        this.currentStep = 3
        if (param.rtpPortRange) {
          const rtpPortRange = String(this.mediaServerForm.rtpPortRange).split(',')
          const sendRtpPortRange = String(this.mediaServerForm.sendRtpPortRange || '').split(',')
          if (rtpPortRange.length > 0) {
            this.rtpPortRange1 = rtpPortRange[0]
            this.rtpPortRange2 = rtpPortRange[1]
          }
          if (sendRtpPortRange.length > 1) {
            this.sendRtpPortRange1 = sendRtpPortRange[0]
            this.sendRtpPortRange2 = sendRtpPortRange[1]
          }
        }
      }
    },
    checkServer() {
      this.serverCheck = 0
      this.$store.dispatch('server/checkMediaServer', this.mediaServerForm)
        .then(data => {
          if (parseInt(this.mediaServerForm.httpPort) !== parseInt(data.httpPort)) {
            this.$message({
              showClose: true,
              message: "检测到可能使用 docker 部署，请保留映射端口",
              type: 'warning',
              duration: 0
            })
          }
          const httpPort = this.mediaServerForm.httpPort
          this.mediaServerForm = Object.assign({}, this.mediaServerForm, data)
          this.mediaServerForm.httpPort = httpPort
          this.mediaServerForm.autoConfig = true
          if (!this.mediaServerForm.type) this.mediaServerForm.type = 'zms'
          this.rtpPortRange1 = 30000
          this.rtpPortRange2 = 30500
          this.sendRtpPortRange1 = 50000
          this.sendRtpPortRange2 = 60000
          this.serverCheck = 1
        })
        .catch(() => {
          this.serverCheck = -1
          this.$message({
            showClose: true,
            message: "连接失败，请检查地址与端口是否正确",
            type: 'warning'
          })
        })
    },
    next() {
      this.currentStep = 2
      this.defaultWidth = 900
      this.setDialogWidth()
    },
    toPort(v, fallback) {
      const n = parseInt(v, 10)
      return Number.isFinite(n) && n > 0 ? n : (fallback || 0)
    },
    buildSavePayload() {
      const f = this.mediaServerForm
      this.portRangeChange()
      return {
        id: f.id || (f.ip + ':' + f.httpPort),
        ip: f.ip,
        hookIp: f.hookIp || f.ip,
        sdpIp: f.sdpIp || f.ip,
        streamIp: f.streamIp || f.ip,
        httpPort: this.toPort(f.httpPort),
        httpSSlPort: this.toPort(f.httpSSlPort, 443),
        rtmpPort: this.toPort(f.rtmpPort, 1935),
        rtmpSSlPort: this.toPort(f.rtmpSSlPort, 19350),
        rtspPort: this.toPort(f.rtspPort, 8554),
        rtspSSLPort: this.toPort(f.rtspSSLPort, 8322),
        rtpProxyPort: this.toPort(f.rtpProxyPort, 10000),
        secret: f.secret || '',
        type: f.type || 'zms',
        autoConfig: !!f.autoConfig,
        rtpEnable: !!f.rtpEnable,
        rtpPortRange: f.rtpPortRange || '',
        sendRtpPortRange: f.sendRtpPortRange || '',
        defaultServer: false
      }
    },
    onSubmit() {
      if (!this.mediaServerForm.ip || !this.mediaServerForm.httpPort) {
        this.$message({ showClose: true, message: "请填写 IP 与 HTTP 端口", type: 'warning' })
        return
      }
      this.dialogLoading = true
      const payload = this.buildSavePayload()
      this.$store.dispatch('server/saveMediaServer', payload)
        .then(() => {
          this.$message({
            showClose: true,
            message: "保存成功",
            type: 'success'
          })
          if (this.listChangeCallback) this.listChangeCallback()
          this.close()
        })
        .catch((err) => {
          this.dialogLoading = false
          const msg = typeof err === 'string' ? err : ((err && err.message) || 'save failed')
          this.$message({ showClose: true, message: msg, type: 'error' })
        })
    },
    close() {
      this.showDialog = false
      this.dialogLoading = false
      this.serverCheck = 0
      this.editingId = ''
      this.mediaServerForm = {
        id: '',
        ip: '',
        autoConfig: true,
        hookIp: '',
        sdpIp: '',
        streamIp: '',
        secret: '',
        httpPort: '',
        httpSSlPort: '',
        recordAssistPort: '',
        rtmpPort: '',
        rtmpSSlPort: '',
        rtpEnable: false,
        rtpPortRange: '',
        sendRtpPortRange: '',
        rtpProxyPort: '',
        rtspPort: '',
        rtspSSLPort: '',
        type: 'zms'
      }
      this.rtpPortRange1 = 30000
      this.rtpPortRange2 = 30500
      this.sendRtpPortRange1 = 50000
      this.sendRtpPortRange2 = 60000
      this.listChangeCallback = null
      this.currentStep = 1
    },
    portRangeChange() {
      if (this.mediaServerForm.rtpEnable) {
        this.mediaServerForm.rtpPortRange = this.rtpPortRange1 + ',' + this.rtpPortRange2
        this.mediaServerForm.sendRtpPortRange = this.sendRtpPortRange1 + ',' + this.sendRtpPortRange2
      }
    }
  }
}
</script>
