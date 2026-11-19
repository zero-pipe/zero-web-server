import {
  PLAYER_MODE_STORAGE_KEY,
  normalizePlayerMode
} from '@/utils/playerStrategy'

function readSavedMode() {
  try {
    return normalizePlayerMode(window.localStorage.getItem(PLAYER_MODE_STORAGE_KEY))
  } catch (e) {
    return 'auto'
  }
}

const state = {
  mode: readSavedMode()
}

const mutations = {
  SET_PLAYER_MODE(state, mode) {
    state.mode = normalizePlayerMode(mode)
  }
}

const actions = {
  /** 全局播放模式：auto | jessibuca | webRTC，所有播放页共用并持久化。 */
  setPlayerMode({ commit }, mode) {
    const normalized = normalizePlayerMode(mode)
    commit('SET_PLAYER_MODE', normalized)
    try {
      window.localStorage.setItem(PLAYER_MODE_STORAGE_KEY, normalized)
    } catch (e) {
      // localStorage 不可用时仅本会话生效
    }
    return normalized
  }
}

export default { state, mutations, actions }
