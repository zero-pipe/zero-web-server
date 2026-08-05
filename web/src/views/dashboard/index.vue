<template>
  <div id="app" class="app-container" style="height: calc(100vh - 118px); background-color: rgba(242,242,242,0.50)">
    <el-row style="width: 100%;height: 100%;">
      <el-col :xl="{ span: 8 }" :lg="{ span: 8 }" :md="{ span: 12 }" :sm="{ span: 12 }" :xs="{ span: 24 }">
        <div id="ThreadsLoad" class="control-cell">
          <div style="width:100%; height:100%; ">
            <consoleCPU ref="consoleCPU" />
          </div>
        </div>
      </el-col>
      <el-col :xl="{ span: 8 }" :lg="{ span: 8 }" :md="{ span: 12 }" :sm="{ span: 12 }" :xs="{ span: 24 }">
        <div id="WorkThreadsLoad" class="control-cell">
          <div style="width:100%; height:100%; ">
            <consoleResource ref="consoleResource" />
          </div>
        </div>
      </el-col>
      <el-col :xl="{ span: 8 }" :lg="{ span: 8 }" :md="{ span: 12 }" :sm="{ span: 12 }" :xs="{ span: 24 }">
        <div id="WorkThreadsLoad" class="control-cell">
          <div style="width:100%; height:100%; ">
            <consoleNet ref="consoleNet" />
          </div>
        </div>
      </el-col>
      <el-col :xl="{ span: 8 }" :lg="{ span: 8 }" :md="{ span: 12 }" :sm="{ span: 12 }" :xs="{ span: 24 }">
        <div id="WorkThreadsLoad" class="control-cell">
          <div style="width:100%; height:100%; ">
            <consoleMem ref="consoleMem" />
          </div>
        </div>
      </el-col>
      <el-col :xl="{ span: 8 }" :lg="{ span: 8 }" :md="{ span: 12 }" :sm="{ span: 12 }" :xs="{ span: 24 }">
        <div id="WorkThreadsLoad" class="control-cell">
          <div style="width:100%; height:100%; ">
            <consoleNodeLoad ref="consoleNodeLoad" />
          </div>
        </div>
      </el-col>
      <el-col :xl="{ span: 8 }" :lg="{ span: 8 }" :md="{ span: 12 }" :sm="{ span: 12 }" :xs="{ span: 24 }">
        <div id="WorkThreadsLoad" class="control-cell">
          <div style="width:100%; height:100%; ">
            <consoleDisk ref="consoleDisk" />
          </div>
        </div>
      </el-col>
    </el-row>
  </div>
</template>

<script>
import consoleCPU from './console/ConsoleCPU.vue'
import consoleMem from './console/ConsoleMEM.vue'
import consoleNet from './console/ConsoleNet.vue'
import consoleNodeLoad from './console/ConsoleNodeLoad.vue'
import consoleDisk from './console/ConsoleDisk.vue'
import consoleResource from './console/ConsoleResource.vue'

export default {
  name: 'Dashboard',
  components: {
    consoleCPU,
    consoleMem,
    consoleNet,
    consoleNodeLoad,
    consoleDisk,
    consoleResource
  },
  data() {
    return {
      timer: null,
      polling: false
    }
  },
  created() {
    this.refreshAll()
    this.scheduleLoop()
  },
  activated() {
    this.refreshAll()
    this.scheduleLoop()
  },
  deactivated() {
    this.stopLoop()
  },
  destroyed() {
    this.stopLoop()
  },
  methods: {
    isConsoleRoute() {
      return this.$route.name === '资源监控' || this.$route.path === '/dashboard'
    },
    stopLoop() {
      if (this.timer != null) {
        window.clearTimeout(this.timer)
        this.timer = null
      }
    },
    scheduleLoop() {
      this.stopLoop()
      this.timer = setTimeout(() => {
        if (!this.isConsoleRoute()) {
          return
        }
        this.refreshAll().finally(() => {
          if (this.isConsoleRoute()) {
            this.scheduleLoop()
          }
        })
      }, 2000)
    },
    refreshAll() {
      if (this.polling) {
        return Promise.resolve()
      }
      this.polling = true
      return Promise.all([
        this.getSystemInfo(),
        this.getLoad(),
        this.getResourceInfo()
      ]).finally(() => {
        this.polling = false
      })
    },
    getSystemInfo() {
      return this.$store.dispatch('server/getSystemInfo')
        .then(data => {
          if (this.$refs.consoleCPU) this.$refs.consoleCPU.setData(data.cpu)
          if (this.$refs.consoleMem) this.$refs.consoleMem.setData(data.mem)
          if (this.$refs.consoleNet) this.$refs.consoleNet.setData(data.net, data.netTotal)
          if (this.$refs.consoleDisk) this.$refs.consoleDisk.setData(data.disk)
        })
        .catch(() => {})
    },
    getLoad() {
      return this.$store.dispatch('server/getMediaServerLoad')
        .then(data => {
          if (this.$refs.consoleNodeLoad) this.$refs.consoleNodeLoad.setData(data)
        })
        .catch(() => {})
    },
    getResourceInfo() {
      return this.$store.dispatch('server/getResourceInfo')
        .then(data => {
          if (this.$refs.consoleResource) this.$refs.consoleResource.setData(data)
        })
        .catch(() => {})
    }
  }
}
</script>

<style>
#app {
  height: 100%;
}
.control-cell {
  padding-top: 10px;
  padding-left: 5px;
  padding-right: 10px;
  height: 360px;
}
</style>
