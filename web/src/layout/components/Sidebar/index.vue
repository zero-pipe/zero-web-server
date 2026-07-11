<template>
  <div
    class="dual-sidebar"
    :class="{ 'is-collapse': isCollapse, 'has-secondary': showSecondary }"
    :style="sidebarStyle"
  >
    <div class="primary-rail">
      <div class="brand" @click="$router.push('/map')">
        <img v-if="logo" :src="logo" class="brand-logo" alt="logo">
        <span v-if="!isCollapse" class="brand-text">Zero</span>
      </div>
      <el-scrollbar class="primary-scroll">
        <div
          v-for="item in menus"
          :key="item.id"
          class="primary-item"
          :class="{ active: activePrimary && activePrimary.id === item.id }"
          :title="item.title"
          @click="onPrimaryClick(item)"
        >
          <svg-icon v-if="!isElIcon(item.icon)" :icon-class="item.icon" class="primary-icon" />
          <i v-else :class="[item.icon, 'primary-icon el-icon']" />
          <span v-if="!isCollapse" class="primary-label">{{ item.title }}</span>
        </div>
      </el-scrollbar>
    </div>

    <div v-show="showSecondary" class="secondary-panel">
      <div class="secondary-header">
        <span class="secondary-title">{{ activePrimary ? activePrimary.title : '' }}</span>
      </div>
      <el-scrollbar class="secondary-scroll">
        <router-link
          v-for="child in secondaryItems"
          :key="child.path"
          :to="child.path"
          class="secondary-item"
          :class="{ active: isSecondaryActive(child.path) }"
        >
          <span class="icon-wrap">
            <svg-icon v-if="!isElIcon(child.icon)" :icon-class="child.icon" class="secondary-icon" />
            <i v-else :class="[child.icon, 'secondary-icon']" />
          </span>
          <span class="secondary-label">{{ child.title }}</span>
        </router-link>
      </el-scrollbar>
    </div>

    <div
      v-if="showSecondary && !isCollapse"
      class="resize-handle"
      title="拖动调整宽度"
      @mousedown="startResize"
    />
  </div>
</template>

<script>
import { mapGetters } from 'vuex'
import { primaryMenus, findPrimaryByPath } from '@/layout/menu'

export default {
  name: 'Sidebar',
  data() {
    return {
      menus: primaryMenus,
      activePrimary: null,
      logo: require('@/assets/zero-media-server-logo.png'),
      resizing: false
    }
  },
  computed: {
    ...mapGetters(['sidebar']),
    isCollapse() {
      return !this.sidebar.opened
    },
    showSecondary() {
      return !this.isCollapse && this.activePrimary && Array.isArray(this.activePrimary.children) && this.activePrimary.children.length > 0
    },
    secondaryItems() {
      return (this.activePrimary && this.activePrimary.children) || []
    },
    primaryWidth() {
      return this.isCollapse ? 64 : 84
    },
    secondaryWidth() {
      return this.showSecondary ? (this.sidebar.secondaryWidth || 176) : 0
    },
    totalWidth() {
      return this.primaryWidth + this.secondaryWidth
    },
    sidebarStyle() {
      return {
        width: this.totalWidth + 'px',
        '--primary-width': this.primaryWidth + 'px',
        '--secondary-width': this.secondaryWidth + 'px'
      }
    }
  },
  watch: {
    $route: {
      immediate: true,
      handler(route) {
        this.activePrimary = findPrimaryByPath(route.path)
        this.$store.dispatch('app/setSidebarWidth', this.totalWidth)
      }
    },
    totalWidth(val) {
      this.$store.dispatch('app/setSidebarWidth', val)
    },
    showSecondary() {
      this.$store.dispatch('app/setSidebarWidth', this.totalWidth)
    }
  },
  beforeDestroy() {
    this.stopResize()
  },
  methods: {
    isElIcon(icon) {
      return icon && icon.indexOf('el-icon') === 0
    },
    isSecondaryActive(path) {
      const current = this.$route.path
      return current === path || current.startsWith(path + '/')
    },
    onPrimaryClick(item) {
      this.activePrimary = item
      if (item.path) {
        if (this.$route.path !== item.path) {
          this.$router.push(item.path)
        }
        return
      }
      if (item.children && item.children.length) {
        const first = item.children[0]
        const alreadyIn = item.children.some(c => this.$route.path === c.path || this.$route.path.startsWith(c.path + '/'))
        if (!alreadyIn) {
          this.$router.push(first.path)
        }
      }
    },
    startResize(e) {
      this.resizing = true
      const startX = e.clientX
      const startW = this.sidebar.secondaryWidth || 176
      const onMove = (ev) => {
        const next = Math.min(280, Math.max(140, startW + (ev.clientX - startX)))
        this.$store.dispatch('app/setSecondaryWidth', next)
      }
      const onUp = () => {
        this.stopResize()
        document.removeEventListener('mousemove', onMove)
        document.removeEventListener('mouseup', onUp)
      }
      document.addEventListener('mousemove', onMove)
      document.addEventListener('mouseup', onUp)
    },
    stopResize() {
      this.resizing = false
    }
  }
}
</script>

<style lang="scss" scoped>
.dual-sidebar {
  display: flex;
  height: 100%;
  position: relative;
  background: #fff;
  box-shadow: 2px 0 8px rgba(21, 101, 192, 0.08);
  overflow: hidden;
  user-select: none;
}

.primary-rail {
  width: var(--primary-width, 84px);
  flex-shrink: 0;
  background: linear-gradient(180deg, #1565c0 0%, #0d47a1 100%);
  display: flex;
  flex-direction: column;
  color: #fff;
}

.brand {
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  cursor: pointer;
  border-bottom: 1px solid rgba(255, 255, 255, 0.12);
  flex-shrink: 0;

  .brand-logo {
    width: 28px;
    height: 28px;
    border-radius: 6px;
    background: #fff;
    object-fit: contain;
  }

  .brand-text {
    font-size: 15px;
    font-weight: 700;
    letter-spacing: 0.5px;
  }
}

.primary-scroll {
  flex: 1;
  height: 0;

  ::v-deep .el-scrollbar__wrap {
    overflow-x: hidden;
  }
}

.primary-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  min-height: 64px;
  padding: 10px 6px;
  cursor: pointer;
  color: rgba(255, 255, 255, 0.78);
  transition: background 0.2s, color 0.2s;

  &:hover {
    background: rgba(255, 255, 255, 0.1);
    color: #fff;
  }

  &.active {
    background: rgba(255, 255, 255, 0.18);
    color: #fff;
    box-shadow: inset 3px 0 0 #90caf9;
  }

  .primary-icon {
    font-size: 22px;
    width: 22px;
    height: 22px;
  }

  .primary-label {
    font-size: 11px;
    line-height: 1.2;
    text-align: center;
    max-width: 72px;
  }
}

.secondary-panel {
  width: var(--secondary-width, 176px);
  flex-shrink: 0;
  background: #f5f8fc;
  border-right: 1px solid #e3ebf5;
  display: flex;
  flex-direction: column;
}

.secondary-header {
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 16px;
  border-bottom: 1px solid #e3ebf5;
  flex-shrink: 0;

  .secondary-title {
    font-size: 14px;
    font-weight: 600;
    color: #1565c0;
  }
}

.secondary-scroll {
  flex: 1;
  height: 0;
  padding: 8px 0;

  ::v-deep .el-scrollbar__wrap {
    overflow-x: hidden;
  }
}

.secondary-item {
  display: flex !important;
  flex-direction: row;
  align-items: center;
  justify-content: flex-start;
  gap: 12px;
  box-sizing: border-box;
  height: 44px;
  min-height: 44px;
  margin: 4px 10px;
  padding: 0 10px 0 8px;
  border-radius: 8px;
  color: #4a5568;
  font-size: 13px;
  font-weight: 400;
  line-height: 44px;
  background: transparent;
  box-shadow: none;
  transition: background 0.15s ease, color 0.15s ease;

  &:hover {
    background: #eef4fb;
    color: #1565c0;

    .icon-wrap {
      background: #d9e8f8;
      color: #1565c0;
    }
  }

  &.active {
    background: #dcecff;
    color: #0d47a1;
    font-weight: 600;
    box-shadow: inset 3px 0 0 #1565c0;

    .icon-wrap {
      background: #1565c0;
      color: #fff;
      box-shadow: 0 2px 6px rgba(21, 101, 192, 0.28);
    }

    .secondary-label {
      color: #0d47a1;
    }

    .secondary-icon {
      color: #fff;
      fill: currentColor;
    }
  }

  .icon-wrap {
    width: 28px;
    height: 28px;
    border-radius: 7px;
    background: #e7edf5;
    color: #5f6f82;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex: 0 0 28px;
    margin: 0;
    padding: 0;
    line-height: 1;
    transition: background 0.15s ease, color 0.15s ease, box-shadow 0.15s ease;
  }

  .secondary-icon {
    font-size: 15px !important;
    width: 15px !important;
    height: 15px !important;
    margin: 0 !important;
    color: inherit;
    fill: currentColor;
    vertical-align: middle;
  }

  .secondary-label {
    flex: 1;
    min-width: 0;
    height: 44px;
    line-height: 44px;
    margin: 0;
    padding: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    text-align: left;
  }
}

.resize-handle {
  position: absolute;
  top: 0;
  right: 0;
  width: 4px;
  height: 100%;
  cursor: col-resize;
  z-index: 2;

  &:hover,
  &:active {
    background: rgba(21, 101, 192, 0.35);
  }
}

.is-collapse {
  .primary-item {
    min-height: 52px;
    padding: 8px 4px;
  }
}
</style>
