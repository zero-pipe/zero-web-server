<template>
  <div class="console-card">
    <div class="console-card__head">
      <div class="console-card__title">
        <i class="el-icon-folder-opened console-card__icon" />
        <span>磁盘空间</span>
      </div>
      <div class="console-card__value">{{ summaryText }}</div>
    </div>
    <div class="console-card__body">
      <ve-bar
        ref="ConsoleDisk"
        :data="chartData"
        :extend="extend"
        :settings="chartSettings"
        width="100%"
        height="100%"
      />
    </div>
  </div>
</template>

<script>
import veBar from 'v-charts/lib/bar'

export default {
  name: 'ConsoleDisk',
  components: { veBar },
  data() {
    return {
      summaryText: '--',
      chartData: {
        columns: ['path', 'free', 'use'],
        rows: []
      },
      chartSettings: {
        stack: {
          xxx: ['free', 'use']
        },
        labelMap: {
          free: '剩余',
          use: '已使用'
        }
      },
      extend: {
        title: { show: false },
        grid: {
          show: true,
          top: '18px',
          right: '30px',
          left: '12px',
          bottom: '42px',
          containLabel: true
        },
        series: {
          barWidth: 30
        },
        legend: {
          left: 'center',
          bottom: '6px'
        },
        tooltip: {
          trigger: 'axis',
          formatter: (data) => {
            let relVal = ''
            for (let i = 0; i < data.length; i++) {
              relVal += data[i].marker + data[i].seriesName + '：' + Number(data[i].value).toFixed(2) + ' GB'
              if (i < data.length - 1) relVal += '<br/>'
            }
            return relVal
          }
        }
      }
    }
  },
  mounted() {
    this.$nextTick(() => {
      setTimeout(() => {
        if (this.$refs.ConsoleDisk && this.$refs.ConsoleDisk.echarts) {
          this.$refs.ConsoleDisk.echarts.resize()
        }
      }, 100)
    })
  },
  methods: {
    setData(data) {
      this.chartData.rows = data || []
      if (!data || !data.length) {
        this.summaryText = '--'
        return
      }
      let used = 0
      let free = 0
      data.forEach((row) => {
        used += Number(row.use) || 0
        free += Number(row.free) || 0
      })
      const total = used + free
      if (total <= 0) {
        this.summaryText = '--'
        return
      }
      this.summaryText = `已用 ${(used / total * 100).toFixed(0)}% · ${used.toFixed(0)}/${total.toFixed(0)} GB`
    }
  }
}
</script>

<style scoped>
.console-card {
  width: 100%;
  height: 100%;
  background: #fff;
  border-radius: 6px;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  box-shadow: 0 1px 2px rgba(15, 23, 42, 0.04);
}
.console-card__head {
  height: 44px;
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 14px;
  border-bottom: 1px solid #eef2f7;
}
.console-card__title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: #1e293b;
}
.console-card__icon {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  background: #fff3e0;
  color: #ef6c00;
}
.console-card__value {
  font-size: 12px;
  font-weight: 600;
  color: #ef6c00;
  font-variant-numeric: tabular-nums;
}
.console-card__body {
  flex: 1;
  min-height: 0;
}
</style>
