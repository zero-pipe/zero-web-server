<template>
  <div
    id="operationsForSystemInfo"
    v-loading="loading"
    class="app-container system-info-page"
  >
    <div class="system-info-shell">
      <div class="system-info-shell-head">
        <div class="system-info-shell-title">平台信息</div>
        <div class="system-info-shell-desc">运行环境与主机概况</div>
        <el-button
          class="system-info-refresh"
          size="mini"
          icon="el-icon-refresh"
          circle
          :loading="loading"
          @click="initData"
        />
      </div>

      <div class="system-info-grid">
        <div
          v-for="section in sectionList"
          :key="section.key"
          class="system-info-card"
        >
          <div class="system-info-card-head">
            <i :class="section.icon" class="system-info-card-icon" />
            <span>{{ section.key }}</span>
          </div>
          <div class="system-info-card-body">
            <div
              v-for="item in section.items"
              :key="item.label"
              class="system-info-row"
            >
              <div class="system-info-label">{{ item.label }}</div>
              <div class="system-info-value">
                <a
                  v-if="isLink(item.value)"
                  :href="item.value"
                  target="_blank"
                  rel="noopener noreferrer"
                >{{ item.value }}</a>
                <span v-else>{{ item.value || '-' }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>

const SECTION_META = [
  { key: '平台信息', icon: 'el-icon-monitor' },
  { key: '操作系统', icon: 'el-icon-cpu' },
  { key: '硬件信息', icon: 'el-icon-s-platform' },
  { key: '文档地址', icon: 'el-icon-document' }
]

export default {
  name: 'OperationsSystemInfo',
  data() {
    return {
      loading: false,
      systemInfoList: {}
    }
  },
  computed: {
    sectionList() {
      const list = []
      const used = {}
      SECTION_META.forEach(meta => {
        const group = this.systemInfoList[meta.key]
        if (!group || typeof group !== 'object') {
          return
        }
        used[meta.key] = true
        list.push({
          key: meta.key,
          icon: meta.icon,
          items: Object.keys(group).map(label => ({
            label,
            value: group[label]
          }))
        })
      })
      // 后端若新增分组，仍展示
      Object.keys(this.systemInfoList || {}).forEach(key => {
        if (used[key]) {
          return
        }
        const group = this.systemInfoList[key]
        if (!group || typeof group !== 'object') {
          return
        }
        list.push({
          key,
          icon: 'el-icon-info',
          items: Object.keys(group).map(label => ({
            label,
            value: group[label]
          }))
        })
      })
      return list
    }
  },
  created() {
    this.initData()
  },
  methods: {
    isLink(value) {
      return typeof value === 'string' && value.startsWith('http')
    },
    initData() {
      this.loading = true
      this.$store.dispatch('server/info')
        .then(data => {
          this.systemInfoList = data || {}
        })
        .catch(error => {
          console.log(error)
        })
        .finally(() => {
          this.loading = false
        })
    }
  }
}
</script>

<style scoped>
.system-info-page {
  padding: 16px 20px 24px;
  min-height: calc(100vh - 124px);
  background: #f0f4f8;
}

.system-info-shell {
  max-width: 1180px;
  margin: 0 auto;
  background: #fff;
  border: 1px solid #e3ebf5;
  border-radius: 10px;
  box-shadow: 0 8px 24px rgba(15, 40, 80, 0.06);
  overflow: hidden;
}

.system-info-shell-head {
  position: relative;
  padding: 18px 22px 16px;
  border-bottom: 1px solid #e8eef6;
  background: linear-gradient(180deg, #f8fbff 0%, #ffffff 100%);
}

.system-info-shell-title {
  font-size: 18px;
  font-weight: 650;
  color: #1e293b;
  line-height: 1.3;
}

.system-info-shell-desc {
  margin-top: 4px;
  font-size: 13px;
  color: #64748b;
}

.system-info-refresh {
  position: absolute;
  top: 16px;
  right: 18px;
}

.system-info-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
  padding: 16px;
}

.system-info-card {
  min-height: 160px;
  background: #f7fafc;
  border: 1px solid #e3ebf5;
  border-radius: 8px;
  overflow: hidden;
}

.system-info-card-head {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px 14px;
  background: #fff;
  border-bottom: 1px solid #e8eef6;
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
}

.system-info-card-icon {
  color: #1565c0;
  font-size: 16px;
}

.system-info-card-body {
  padding: 8px 14px 12px;
}

.system-info-row {
  display: grid;
  grid-template-columns: 96px 1fr;
  gap: 10px;
  padding: 8px 0;
  border-bottom: 1px dashed #e6edf5;
}

.system-info-row:last-child {
  border-bottom: none;
}

.system-info-label {
  font-size: 12px;
  color: #64748b;
  line-height: 1.5;
}

.system-info-value {
  font-size: 13px;
  color: #1e293b;
  line-height: 1.5;
  word-break: break-all;
}

.system-info-value a {
  color: #1565c0;
  text-decoration: none;
}

.system-info-value a:hover {
  text-decoration: underline;
}

@media (max-width: 900px) {
  .system-info-grid {
    grid-template-columns: 1fr;
  }
}
</style>
