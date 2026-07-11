<template>
  <div class="stream-media-panel">
    <el-form label-width="88px" size="small" class="stream-media-form">
      <el-form-item label="网页分享">
        <el-input v-model="playerUrl" :disabled="true" placeholder="暂无">
          <template slot="append">
            <i
              class="copy-btn el-icon-document-copy"
              title="复制网页分享链接"
              @click="copyUrl(playerUrl)"
            />
          </template>
        </el-input>
      </el-form-item>

      <el-form-item label="嵌入代码">
        <el-input v-model="sharedIframe" :disabled="true" placeholder="暂无">
          <template slot="append">
            <i
              class="copy-btn el-icon-document-copy"
              title="复制 iframe 嵌入代码"
              @click="copyUrl(sharedIframe)"
            />
          </template>
        </el-input>
      </el-form-item>

      <el-form-item label="直链拉流">
        <el-input v-model="playUrl" :disabled="true" placeholder="暂无">
          <el-dropdown
            v-if="addressList.length"
            slot="prepend"
            trigger="click"
            placement="bottom-start"
            @command="copyUrl"
          >
            <el-button size="mini">
              协议列表<i class="el-icon-arrow-down el-icon--right" />
            </el-button>
            <el-dropdown-menu slot="dropdown" class="stream-addr-menu">
              <el-dropdown-item
                v-for="item in addressList"
                :key="item.key"
                :command="item.url"
                class="stream-addr-item"
              >
                <span class="stream-addr-proto">{{ item.label }}</span>
                <span class="stream-addr-url" :title="item.url">{{ item.url }}</span>
              </el-dropdown-item>
            </el-dropdown-menu>
          </el-dropdown>
          <el-button
            slot="append"
            icon="el-icon-document-copy"
            title="复制当前直链"
            @click="copyUrl(playUrl)"
          />
        </el-input>
      </el-form-item>
    </el-form>
  </div>
</template>

<script>
const ADDRESS_DEFS = [
  { key: 'flv', label: 'FLV' },
  { key: 'https_flv', label: 'FLV HTTPS' },
  { key: 'ws_flv', label: 'FLV WS' },
  { key: 'wss_flv', label: 'FLV WSS' },
  { key: 'fmp4', label: 'FMP4' },
  { key: 'https_fmp4', label: 'FMP4 HTTPS' },
  { key: 'ws_fmp4', label: 'FMP4 WS' },
  { key: 'wss_fmp4', label: 'FMP4 WSS' },
  { key: 'hls', label: 'HLS' },
  { key: 'https_hls', label: 'HLS HTTPS' },
  { key: 'ws_hls', label: 'HLS WS' },
  { key: 'wss_hls', label: 'HLS WSS' },
  { key: 'ts', label: 'TS' },
  { key: 'https_ts', label: 'TS HTTPS' },
  { key: 'ws_ts', label: 'TS WS' },
  { key: 'wss_ts', label: 'TS WSS' },
  { key: 'rtc', label: 'WebRTC' },
  { key: 'rtcs', label: 'WebRTC TLS' },
  { key: 'rtmp', label: 'RTMP' },
  { key: 'rtmps', label: 'RTMPS' },
  { key: 'rtsp', label: 'RTSP' },
  { key: 'rtsps', label: 'RTSPS' }
]

export default {
  name: 'StreamMediaPanel',
  props: {
    playerUrl: { type: String, default: '' },
    playUrl: { type: String, default: '' },
    streamInfo: { type: Object, default: null }
  },
  computed: {
    sharedIframe() {
      if (!this.playerUrl) return ''
      return `<iframe src="${this.playerUrl}"></iframe>`
    },
    addressList() {
      if (!this.streamInfo) return []
      return ADDRESS_DEFS
        .filter((def) => this.streamInfo[def.key])
        .map((def) => ({
          key: def.key,
          label: def.label,
          url: this.streamInfo[def.key]
        }))
    }
  },
  methods: {
    copyUrl(text) {
      if (!text) {
        this.$message.warning({ showClose: true, message: '暂无地址可复制' })
        return
      }
      this.$copyText(text).then(() => {
        this.$message.success({ showClose: true, message: '已复制到剪贴板' })
      }, () => {})
    }
  }
}
</script>

<style lang="scss">
.stream-media-panel {
  .copy-btn {
    cursor: pointer;
    color: #1565c0;

    &:hover {
      color: #0d47a1;
    }
  }

  .el-form-item {
    margin-bottom: 10px;
  }

  .el-input.is-disabled .el-input__inner {
    color: #334155;
    cursor: text;
  }
}

.stream-addr-menu {
  max-width: min(640px, 90vw);
  padding: 6px 0 !important;
}

.stream-addr-menu .stream-addr-item {
  padding: 0 !important;
  line-height: normal !important;
}

.stream-addr-menu .stream-addr-item.el-dropdown-menu__item {
  display: grid !important;
  grid-template-columns: 96px minmax(0, 1fr);
  align-items: center;
  gap: 10px;
  padding: 8px 14px !important;
  height: auto !important;
}

.stream-addr-menu .stream-addr-proto {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-height: 22px;
  padding: 0 8px;
  border-radius: 4px;
  background: #e8f1fb;
  color: #1565c0;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}

.stream-addr-menu .stream-addr-url {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-family: Consolas, 'Courier New', monospace;
  font-size: 12px;
  color: #475569;
}
</style>
