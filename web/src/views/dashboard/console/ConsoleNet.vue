<template>
  <div class="console-card">
    <div class="console-card__head">
      <div class="console-card__title">
        <i class="el-icon-connection console-card__icon" />
        <span>网络吞吐</span>
      </div>
      <div class="console-card__hint">单位 Mbps · 上传 / 下载</div>
    </div>
    <div class="console-card__body">
      <ve-line
        ref="ConsoleNet"
        :data="chartData"
        :extend="extend"
        :settings="chartSettings"
        :events="chartEvents"
        width="100%"
        height="100%"
      />
    </div>
  </div>
</template>

<script>
import veLine from 'v-charts/lib/line'
import moment from 'moment/moment'

export default {
  name: 'ConsoleNet',
  components: { veLine },
  data() {
    return {
      chartData: {
        columns: ['time', 'out', 'in'],
        rows: []
      },
      chartSettings: {
        area: true,
        labelMap: {
          in: '下载',
          out: '上传'
        }
      },
      extend: {
        title: { show: false },
        grid: {
          show: true,
          top: '18px',
          left: '12px',
          right: '24px',
          bottom: '36px',
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
          max: 1000,
          splitNumber: 5,
          position: 'left',
          silent: true,
          name: 'Mbps',
          nameTextStyle: { color: '#94a3b8', fontSize: 11 }
        },
        tooltip: {
          trigger: 'axis',
          formatter: (data) => {
            let inSel = true
            let outSel = true
            for (const key in this.extend.legend.selected) {
              if (key === '上传') outSel = this.extend.legend.selected[key]
              if (key === '下载') inSel = this.extend.legend.selected[key]
            }
            if (outSel && inSel && data.length >= 2) {
              return (
                data[1].marker +
                '下载：' +
                parseFloat(data[1].data[1]).toFixed(2) +
                ' Mbps<br/>' +
                data[0].marker +
                '上传：' +
                parseFloat(data[0].data[1]).toFixed(2) +
                ' Mbps'
              )
            }
            if (outSel && data[0]) {
              return data[0].marker + '上传：' + parseFloat(data[0].data[1]).toFixed(2) + ' Mbps'
            }
            if (inSel && data[0]) {
              return data[0].marker + '下载：' + parseFloat(data[0].data[1]).toFixed(2) + ' Mbps'
            }
            return ''
          }
        },
        legend: {
          left: 'center',
          bottom: '6px',
          selected: {}
        }
      },
      chartEvents: {
        legendselectchanged: (item) => {
          this.extend.legend.selected = item.selected
        }
      }
    }
  },
  mounted() {
    this.$nextTick(() => {
      setTimeout(() => {
        if (this.$refs.ConsoleNet && this.$refs.ConsoleNet.echarts) {
          this.$refs.ConsoleNet.echarts.resize()
        }
      }, 100)
    })
  },
  methods: {
    setData(data, total) {
      this.chartData.rows = data || []
      if (total) this.extend.yAxis.max = total
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
  background: #e3f2fd;
  color: #1565c0;
}
.console-card__hint {
  font-size: 12px;
  color: #94a3b8;
}
.console-card__body {
  flex: 1;
  min-height: 0;
}
</style>
