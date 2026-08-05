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

      <div class="system-info-card">
        <div
          v-for="group in displayGroups"
          :key="group.title"
          class="system-info-group"
        >
          <div class="system-info-group-title">{{ group.title }}</div>
          <div class="system-info-fields">
            <div
              v-for="item in group.items"
              :key="item.label"
              class="system-info-field"
              :class="{ 'is-wide': item.wide }"
            >
              <div class="system-info-label">{{ item.label }}</div>
              <div class="system-info-value">
                <a
                  v-if="isLink(item.value)"
                  :href="item.value"
                  target="_blank"
                  rel="noopener noreferrer"
                >{{ item.value }}</a>
                <span v-else>{{ item.value || '—' }}</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
/** 只保留关键运行/主机信息，按分组展示 */
const FIELD_PLAN = [
  {
    title: '运行',
    items: [
      { label: '版本', from: ['平台信息', '版本'] },
      { label: 'Go 版本', from: ['平台信息', 'Go版本'] },
      { label: '监听端口', from: ['平台信息', '监听端口'] },
      { label: '启动时间', from: ['平台信息', '启动时间'] },
      { label: '运行时长', from: ['平台信息', '运行时长'] }
    ]
  },
  {
    title: '主机',
    items: [
      { label: '操作系统', key: 'os' },
      { label: 'CPU', from: ['硬件信息', 'CPU'], wide: true },
      { label: '内存', from: ['硬件信息', '内存'] },
      { label: '网卡', from: ['硬件信息', '网卡'], wide: true }
    ]
  },
  {
    title: '文档地址',
    items: [
      { label: '本机地址', from: ['文档地址', '本机地址'], wide: true },
      { label: '项目地址', from: ['文档地址', '项目地址'], wide: true }
    ]
  }
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
    displayGroups() {
      return FIELD_PLAN.map(group => ({
        title: group.title,
        items: group.items.map(item => ({
          label: item.label,
          wide: !!item.wide,
          value: item.key === 'os'
            ? this.osLabel()
            : this.pick(item.from[0], item.from[1])
        }))
      }))
    }
  },
  created() {
    this.initData()
  },
  methods: {
    pick(section, field) {
      const group = this.systemInfoList && this.systemInfoList[section]
      if (!group || typeof group !== 'object') return ''
      return group[field] || ''
    },
    /** 操作系统只展示 windows / linux（其它系统原样小写） */
    osLabel() {
      const raw = String(this.pick('操作系统', '类型') || '').toLowerCase()
      if (raw.indexOf('win') >= 0) return 'windows'
      if (raw.indexOf('linux') >= 0) return 'linux'
      return raw || '—'
    },
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
  max-width: 720px;
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

.system-info-card {
  padding: 8px 22px 22px;
}

.system-info-group {
  padding-top: 16px;
}

.system-info-group + .system-info-group {
  margin-top: 8px;
  border-top: 1px solid #eef2f7;
}

.system-info-group-title {
  margin-bottom: 10px;
  font-size: 12px;
  font-weight: 600;
  letter-spacing: 0.04em;
  color: #94a3b8;
  text-transform: none;
}

.system-info-fields {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px 20px;
}

.system-info-field.is-wide {
  grid-column: 1 / -1;
}

.system-info-label {
  font-size: 12px;
  color: #94a3b8;
  line-height: 1.3;
  margin-bottom: 4px;
}

.system-info-value {
  font-size: 14px;
  font-weight: 550;
  color: #1e293b;
  line-height: 1.45;
  word-break: break-all;
}

.system-info-value a {
  color: #1565c0;
  text-decoration: none;
  font-weight: 500;
}

.system-info-value a:hover {
  text-decoration: underline;
}

@media (max-width: 560px) {
  .system-info-shell {
    max-width: 100%;
  }

  .system-info-fields {
    grid-template-columns: 1fr;
  }

  .system-info-field.is-wide {
    grid-column: auto;
  }
}
</style>
