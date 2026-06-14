<template>
  <div class="trend-chart">
    <div v-if="seriesStats.length" class="legend-row">
      <div v-for="item in seriesStats" :key="item.name" class="legend-item">
        <span class="legend-dot" :style="{ backgroundColor: item.color }" />
        <span class="legend-name">{{ item.name }}</span>
        <span class="legend-value">{{ item.latest }}</span>
      </div>
    </div>
    <a-empty v-if="!hasData" description="暂无历史数据" />
    <div
      v-else
      class="chart-stage"
      @mouseleave="hoverIndex = null"
      @mousemove="handleMouseMove"
    >
      <svg class="chart-svg" :style="{ height: `${height}px` }" :viewBox="`0 0 ${svgWidth} ${height}`" preserveAspectRatio="none">
      <g>
        <line
          v-for="tick in yTicks"
          :key="`grid-${tick}`"
          :x1="padding.left"
          :x2="svgWidth - padding.right"
          :y1="toY(tick)"
          :y2="toY(tick)"
          class="grid-line"
        />
        <text
          v-for="tick in yTicks"
          :key="`label-${tick}`"
          :x="padding.left - 8"
          :y="toY(tick) + 4"
          class="axis-label"
          text-anchor="end"
        >
          {{ formatValue(tick) }}
        </text>
        <line
          v-for="tick in xTicks"
          :key="`x-${tick.index}`"
          :x1="toX(tick.index)"
          :x2="toX(tick.index)"
          :y1="padding.top"
          :y2="height - padding.bottom"
          class="grid-line grid-line-vertical"
        />
        <polyline
          v-for="item in series"
          :key="item.name"
          :points="buildPoints(item.values)"
          :stroke="item.color"
          class="trend-line"
        />
        <line
          v-if="hasHover"
          :x1="toX(hoverIndex!)"
          :x2="toX(hoverIndex!)"
          :y1="padding.top"
          :y2="height - padding.bottom"
          class="hover-line"
        />
        <!-- v-if 与 v-for 不应放同一元素（Vue3 中 v-if 优先级更高，无法访问 v-for 变量；
             这里用 template 包裹，hasHover 为 false 时不渲染整组 hover 圆点） -->
        <template v-if="hasHover">
          <circle
            v-for="item in series"
            :key="`${item.name}-hover`"
            :cx="toX(hoverIndex!)"
            :cy="toY(item.values[hoverIndex!] || 0)"
            :fill="item.color"
            r="4"
            class="hover-point"
          />
        </template>
        <circle
          v-for="item in series"
          :key="`${item.name}-last`"
          :cx="toX(item.values.length - 1)"
          :cy="toY(item.values[item.values.length - 1] || 0)"
          :fill="item.color"
          r="3"
        />
        <text
          v-for="tick in xTicks"
          :key="`xlabel-${tick.index}`"
          :x="toX(tick.index)"
          :y="height - 8"
          class="axis-label"
          text-anchor="middle"
        >
          {{ tick.label }}
        </text>
      </g>
      </svg>
      <div v-if="hoverDetails" class="tooltip-card" :style="tooltipStyle">
        <div class="tooltip-title">{{ hoverDetails.label }}</div>
        <div v-for="item in hoverDetails.items" :key="item.name" class="tooltip-row">
          <span class="legend-dot" :style="{ backgroundColor: item.color }" />
          <span>{{ item.name }}</span>
          <span class="tooltip-value">{{ item.value }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'

interface ChartSeries {
  name: string
  color: string
  values: number[]
}

const props = withDefaults(defineProps<{
  labels: string[]
  series: ChartSeries[]
  unit?: string
  precision?: number
  height?: number
  maxValue?: number
}>(), {
  unit: '',
  precision: 1,
  height: 220,
  maxValue: undefined,
})

const svgWidth = 720
const padding = {
  top: 16,
  right: 20,
  bottom: 32,
  left: 44,
}

const chartWidth = computed(() => svgWidth - padding.left - padding.right)
const chartHeight = computed(() => props.height - padding.top - padding.bottom)
const hoverIndex = ref<number | null>(null)

const allValues = computed(() => props.series.flatMap(item => item.values).filter(value => Number.isFinite(value) && value >= 0))
const resolvedMaxValue = computed(() => {
  if (typeof props.maxValue === 'number' && props.maxValue > 0) return props.maxValue
  const max = Math.max(...allValues.value, 0)
  return max > 0 ? Math.ceil(max * 1.1) : 1
})

const hasData = computed(() => props.labels.length > 0 && props.series.some(item => item.values.length > 0))

const yTicks = computed(() => {
  const step = resolvedMaxValue.value / 4
  return [0, step, step * 2, step * 3, resolvedMaxValue.value]
})

const xTicks = computed(() => {
  const total = props.labels.length
  if (total <= 1) {
    return total === 1 ? [{ index: 0, label: props.labels[0] }] : []
  }

  const tickCount = Math.min(6, total)
  const indexes = new Set<number>([0, total - 1])
  for (let i = 1; i < tickCount - 1; i += 1) {
    indexes.add(Math.round((i * (total - 1)) / (tickCount - 1)))
  }

  return Array.from(indexes)
    .sort((left, right) => left - right)
    .map(index => ({ index, label: props.labels[index] }))
})

const seriesStats = computed(() => {
  return props.series.map(item => {
    const latestValue = item.values[item.values.length - 1] || 0
    return {
      name: item.name,
      color: item.color,
      latest: formatValue(latestValue),
    }
  })
})

const hasHover = computed(() => hoverIndex.value !== null && hoverIndex.value >= 0 && hoverIndex.value < props.labels.length)

const hoverDetails = computed(() => {
  if (!hasHover.value) return null

  return {
    label: props.labels[hoverIndex.value!],
    items: props.series.map(item => ({
      name: item.name,
      color: item.color,
      value: formatValue(item.values[hoverIndex.value!] || 0),
    })),
  }
})

const tooltipStyle = computed(() => {
  if (!hasHover.value) return {}

  const x = toX(hoverIndex.value!)
  const left = Math.min(Math.max(x + 12, padding.left), svgWidth - 180)
  return {
    left: `${(left / svgWidth) * 100}%`,
    top: `${padding.top}px`,
  }
})

const toX = (index: number) => {
  if (props.labels.length <= 1) return padding.left + chartWidth.value / 2
  return padding.left + (chartWidth.value * index) / (props.labels.length - 1)
}

const toY = (value: number) => {
  return padding.top + chartHeight.value * (1 - value / resolvedMaxValue.value)
}

const buildPoints = (values: number[]) => {
  return values.map((value, index) => `${toX(index)},${toY(value || 0)}`).join(' ')
}

const formatValue = (value: number) => `${value.toFixed(props.precision)}${props.unit}`

const handleMouseMove = (event: MouseEvent) => {
  if (!props.labels.length) return

  const target = event.currentTarget as HTMLDivElement
  const rect = target.getBoundingClientRect()
  const ratio = svgWidth / rect.width
  const x = (event.clientX - rect.left) * ratio
  const clampedX = Math.min(Math.max(x, padding.left), svgWidth - padding.right)

  if (props.labels.length === 1) {
    hoverIndex.value = 0
    return
  }

  const index = Math.round(((clampedX - padding.left) / chartWidth.value) * (props.labels.length - 1))
  hoverIndex.value = Math.min(Math.max(index, 0), props.labels.length - 1)
}
</script>

<style scoped>
.trend-chart {
  min-height: 260px;
}

.chart-stage {
  position: relative;
}

.legend-row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  margin-bottom: 12px;
}

.legend-item {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  color: #666;
}

.legend-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.legend-name {
  color: #333;
}

.legend-value {
  font-weight: 600;
  color: #111;
}

.chart-svg {
  width: 100%;
}

.grid-line {
  stroke: #f0f0f0;
  stroke-width: 1;
}

.grid-line-vertical {
  stroke-dasharray: 2 4;
}

.trend-line {
  fill: none;
  stroke-width: 2.5;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.hover-line {
  stroke: #bfbfbf;
  stroke-width: 1;
  stroke-dasharray: 4 4;
}

.hover-point {
  stroke: #fff;
  stroke-width: 2;
}

.axis-label {
  fill: #999;
  font-size: 11px;
}

.tooltip-card {
  position: absolute;
  min-width: 140px;
  padding: 10px 12px;
  border-radius: 8px;
  background: rgba(17, 24, 39, 0.92);
  color: #fff;
  pointer-events: none;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.18);
}

.tooltip-title {
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 600;
}

.tooltip-row {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
}

.tooltip-row + .tooltip-row {
  margin-top: 4px;
}

.tooltip-value {
  margin-left: auto;
  font-weight: 600;
}
</style>
