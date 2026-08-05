<template>
  <div class="console-card">
    <div class="console-card__head">
      <div class="console-card__title">
        <i class="el-icon-s-platform console-card__icon" />
        <span>媒体节点负载</span>
      </div>
      <div class="console-card__hint">按节点统计路数</div>
    </div>
    <div class="console-card__body">
      <div v-if="!chartData.rows.length" class="console-empty">
        暂无媒体节点负载数据
      </div>
      <ve-histogram
        v-else
        ref="consoleNodeLoad"
        :data="chartData"
        :extend="extend"
        :events="events"
        :settings="chartSettings"
        width="100%"
        height="100%"
        :legend-visible="true"
      />
      <HasStreamChannel ref="hasStreamChannel" />
    </div>
  </div>
</template>

<script>
import veHistogram from 'v-charts/lib/histogram'
import HasStreamChannel from '@/views/dashboard/dialog/hasStreamChannel.vue'

export default {
  name: 'ConsoleNodeLoad',
  components: {
    veHistogram,
    HasStreamChannel
  },
  data() {
    return {
      chartData: {
        columns: ['id', 'push', 'proxy', 'gbReceive', 'gbSend'],
        rows: []
      },
      chartSettings: {
        labelMap: {
          push: '直播推流',
          proxy: '拉流代理',
          gbReceive: '国标收流',
          gbSend: '国标推流'
        }
      },
      extend: {
        title: { show: false },
        grid: {
          top: '18px',
          left: '12px',
          right: '18px',
          bottom: '42px',
          containLabel: true
        },
        legend: {
          left: 'center',
          bottom: '6px'
        },
        label: {
          show: true,
          position: 'top'
        }
      },
      events: {
        click: this.onClick
      }
    }
  },
  mounted() {
    this.$nextTick(() => {
      setTimeout(() => {
        if (this.$refs.consoleNodeLoad && this.$refs.consoleNodeLoad.echarts) {
          this.$refs.consoleNodeLoad.echarts.resize()
        }
      }, 100)
    })
  },
  methods: {
    setData(data) {
      this.chartData.rows = data || []
      this.$nextTick(() => {
        if (this.$refs.consoleNodeLoad && this.$refs.consoleNodeLoad.echarts) {
          this.$refs.consoleNodeLoad.echarts.resize()
        }
      })
    },
    onClick(v) {
      if (v.seriesName === '国标收流') {
        this.$refs.hasStreamChannel.openDialog()
      }
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
  background: #f3e5f5;
  color: #8e24aa;
}
.console-card__hint {
  font-size: 12px;
  color: #94a3b8;
}
.console-card__body {
  flex: 1;
  min-height: 0;
  position: relative;
}
.console-empty {
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #94a3b8;
  font-size: 13px;
}
</style>
