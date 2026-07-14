<template>
  <div class="console-card">
    <div class="console-card__head">
      <div class="console-card__title">
        <i class="el-icon-coin console-card__icon is-mem" />
        <span>内存使用率</span>
      </div>
      <div class="console-card__value is-mem">{{ displayPercent }}</div>
    </div>
    <div class="console-card__body">
      <ve-line
        ref="ConsoleMEM"
        :data="chartData"
        :extend="extend"
        width="100%"
        height="100%"
        :legend-visible="false"
      />
    </div>
  </div>
</template>

<script>
import veLine from 'v-charts/lib/line'
import moment from 'moment/moment'

export default {
  name: 'ConsoleMEM',
  components: { veLine },
  data() {
    return {
      latest: null,
      chartData: {
        columns: ['time', 'data'],
        rows: []
      },
      extend: {
        title: { show: false },
        grid: {
          show: true,
          top: '18px',
          left: '12px',
          right: '24px',
          bottom: '12px',
          containLabel: true
        },
        xAxis: {
          time: 'time',
          max: 'dataMax',
          boundaryGap: ['20%', '20%'],
          axisLabel: {
            formatter: (v) => moment(v).format('HH:mm:ss'),
            showMaxLabel: true
          }
        },
        yAxis: {
          type: 'value',
          min: 0,
          max: 1,
          splitNumber: 5,
          position: 'left',
          silent: true,
          axisLabel: {
            formatter: (v) => v * 100 + '%'
          }
        },
        tooltip: {
          trigger: 'axis',
          formatter: (data) => {
            if (!data || !data[0]) return ''
            return (
              moment(data[0].data[0]).format('HH:mm:ss') +
              '<br/>' +
              data[0].marker +
              '内存：' +
              (data[0].data[1] * 100).toFixed(2) +
              '%'
            )
          }
        },
        series: {
          name: '内存',
          itemStyle: { color: '#00897b' },
          areaStyle: {
            color: {
              type: 'linear',
              x: 0,
              y: 0,
              x2: 0,
              y2: 1,
              colorStops: [
                { offset: 0, color: 'rgba(0,137,123,0.45)' },
                { offset: 1, color: 'rgba(0,137,123,0.05)' }
              ]
            }
          }
        }
      }
    }
  },
  computed: {
    displayPercent() {
      if (this.latest == null || Number.isNaN(this.latest)) return '--'
      return (this.latest * 100).toFixed(1) + '%'
    }
  },
  mounted() {
    this.$nextTick(() => {
      setTimeout(() => {
        if (this.$refs.ConsoleMEM && this.$refs.ConsoleMEM.echarts) {
          this.$refs.ConsoleMEM.echarts.resize()
        }
      }, 100)
    })
  },
  methods: {
    setData(data) {
      this.chartData.rows = data || []
      if (data && data.length) {
        const last = data[data.length - 1]
        this.latest = typeof last.data === 'number' ? last.data : null
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
  background: #e0f2f1;
  color: #00897b;
}
.console-card__value {
  font-size: 16px;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  color: #00897b;
}
.console-card__body {
  flex: 1;
  min-height: 0;
}
</style>
