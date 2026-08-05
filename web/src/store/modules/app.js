import Cookies from 'js-cookie'

const SECONDARY_WIDTH_KEY = 'sidebarSecondaryWidth'

function readSecondaryWidth() {
  const raw = Cookies.get(SECONDARY_WIDTH_KEY)
  const n = raw ? parseInt(raw, 10) : 176
  if (Number.isNaN(n)) return 176
  return Math.min(280, Math.max(140, n))
}

const state = {
  sidebar: {
    opened: Cookies.get('sidebarStatus') ? !!+Cookies.get('sidebarStatus') : true,
    withoutAnimation: false,
    secondaryWidth: readSecondaryWidth(),
    totalWidth: 84 + readSecondaryWidth()
  },
  device: 'desktop'
}

const mutations = {
  TOGGLE_SIDEBAR: state => {
    state.sidebar.opened = !state.sidebar.opened
    state.sidebar.withoutAnimation = false
    if (state.sidebar.opened) {
      Cookies.set('sidebarStatus', 1)
    } else {
      Cookies.set('sidebarStatus', 0)
    }
  },
  CLOSE_SIDEBAR: (state, withoutAnimation) => {
    Cookies.set('sidebarStatus', 0)
    state.sidebar.opened = false
    state.sidebar.withoutAnimation = withoutAnimation
  },
  TOGGLE_DEVICE: (state, device) => {
    state.device = device
  },
  SET_SECONDARY_WIDTH: (state, width) => {
    state.sidebar.secondaryWidth = width
    Cookies.set(SECONDARY_WIDTH_KEY, String(width))
  },
  SET_SIDEBAR_WIDTH: (state, width) => {
    state.sidebar.totalWidth = width
  }
}

const actions = {
  toggleSideBar({ commit }) {
    commit('TOGGLE_SIDEBAR')
  },
  closeSideBar({ commit }, { withoutAnimation }) {
    commit('CLOSE_SIDEBAR', withoutAnimation)
  },
  toggleDevice({ commit }, device) {
    commit('TOGGLE_DEVICE', device)
  },
  setSecondaryWidth({ commit }, width) {
    commit('SET_SECONDARY_WIDTH', width)
  },
  setSidebarWidth({ commit }, width) {
    commit('SET_SIDEBAR_WIDTH', width)
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
