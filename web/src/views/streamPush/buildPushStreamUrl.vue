<template>
  <div id="buildPushStreamUrl">
    <el-dialog
      v-el-drag-dialog
      title="构建推流地址"
      width="40%"
      top="2rem"
      :close-on-click-modal="false"
      :visible.sync="showDialog"
      :destroy-on-close="true"
      @close="close()"
    >
      <div id="shared" style="margin-right: 20px;">
        <el-form ref="buildFrom" status-icon label-width="80px">
          <el-form-item label="应用名" prop="app">
            <el-input v-model="app" autocomplete="off" />
          </el-form-item>
          <el-form-item label="流ID" prop="stream">
            <el-input v-model="stream" autocomplete="off" />
          </el-form-item>
          <el-form-item label="媒体节点" prop="mediaServerId">
            <el-select v-model="mediaServer" placeholder="请选择" style="width: 100%">
              <el-option
                v-for="item in mediaServerList"
                :key="item.id"
                :label="item.id"
                :value="item"
              />
            </el-select>
          </el-form-item>
          <el-form-item label="地址" prop="url">
            <div style="width: 100%" v-if="rtc" title="点击拷贝">
              <el-tag size="medium" @click="copyUrl(rtc)">
                <i class="el-icon-document-copy"/>
                {{ rtc }}
              </el-tag>
            </div>
            <div style="width: 100%" v-if="rtsp" title="点击拷贝">
              <el-tag size="medium" @click="copyUrl(rtsp)">
                <i class="el-icon-document-copy"/>
                {{ rtsp }}
              </el-tag>
            </div>
            <div style="width: 100%" v-if="rtmp" title="点击拷贝">
              <el-tag size="medium" @click="copyUrl(rtmp)">
                <i class="el-icon-document-copy"/>
                {{ rtmp }}
              </el-tag>
            </div>
            <div style="width: 100%" v-if="rtcs" title="点击拷贝">
              <el-tag size="medium" @click="copyUrl(rtcs)">
                <i class="el-icon-document-copy" />
                {{ rtcs }}
              </el-tag>
            </div>
          </el-form-item>
          <el-form-item>
            <div style="float: right;">
              <el-button @click="close">关闭</el-button>
            </div>
          </el-form-item>
        </el-form>
      </div>
    </el-dialog>
  </div>
</template>

<script>

import elDragDialog from '@/directive/el-drag-dialog'
import { first as firstAccessToken } from '@/api/accessToken'


export default {
  name: 'BuildPushStreamUrl',
  directives: { elDragDialog },
  props: {},
  data() {
    return {
      showDialog: false,
      app: null,
      stream: null,
      mediaServer: null,
      mediaServerList: [],
      accessToken: '',
      endCallback: null
    }
  },
  computed: {
    // 接入鉴权启用时，推流 URL 带上 token，ZMS 本地白名单校验。
    tokenParam() {
      return this.accessToken ? `token=${this.accessToken}` : ''
    },
    rtsp() {
      if (!this.mediaServer || !this.stream || !this.app) {
        return ''
      }
      const base = `rtsp://${this.mediaServer.streamIp}:${this.mediaServer.rtspPort}/${this.app}/${this.stream}`
      return this.tokenParam ? `${base}?${this.tokenParam}` : base
    },
    rtmp() {
      if (!this.mediaServer || !this.stream || !this.app) {
        return ''
      }
      const base = `rtmp://${this.mediaServer.streamIp}:${this.mediaServer.rtmpPort}/${this.app}/${this.stream}`
      return this.tokenParam ? `${base}?${this.tokenParam}` : base
    },
    rtc() {
      if (!this.mediaServer || !this.stream || !this.app) {
        return ''
      }
      const base = `http://${this.mediaServer.streamIp}:${this.mediaServer.httpPort}/index/api/webrtc?app=${this.app}&stream=${this.stream}`
      return this.tokenParam ? `${base}&${this.tokenParam}` : base
    },
    rtcs() {
      if (!this.mediaServer || !this.stream || !this.app) {
        return ''
      }
      const base = `https://${this.mediaServer.streamIp}:${this.mediaServer.httpSSlPort}/index/api/webrtc?app=${this.app}&stream=${this.stream}`
      return this.tokenParam ? `${base}&${this.tokenParam}` : base
    }
  },
  created() {
    this.initData()
  },
  methods: {
    openDialog: function(callback) {
      this.endCallback = callback
      this.showDialog = true
    },
    close: function() {
      this.showDialog = false
      this.app = null
      this.stream = null
      this.mediaServer = null
      this.endCallback = null
      this.mediaServerList = []
    },
    initData: function() {
      this.loading = true
      this.$store.dispatch('server/getMediaServerList').then(data => {
        this.mediaServerList = data
      })
      firstAccessToken().then(data => {
        this.accessToken = (data && data.token) || ''
      }).catch(() => { this.accessToken = '' })
    },
    copyUrl: function(dropdownItem) {
      this.$copyText(dropdownItem).then(() => {
        this.$message.success({
          showClose: true,
          message: '成功拷贝到粘贴板'
        })
      }, () => {})
    }
  }
}
</script>
