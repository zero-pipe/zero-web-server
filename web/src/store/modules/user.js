import crypto from 'crypto'
import {
  add,
  changePassword,
  changePasswordForAdmin,
  changePushKey,
  getUserInfo,
  login,
  logout,
  queryList,
  removeById
} from '@/api/user'
import {
  getToken,
  setToken,
  setName,
  removeToken,
  removeName,
  setServerId,
  removeServerId,
  getMenus,
  setMenus,
  removeMenus,
  getRoleId,
  setRoleId,
  removeRoleId
} from '@/utils/auth'
import { resetRouter } from '@/router'

const getDefaultState = () => {
  return {
    token: getToken(),
    name: '',
    serverId: '',
    menus: getMenus(),
    roleId: getRoleId(),
    showConfirmBoxForLoginLose: true
  }
}

const state = getDefaultState()

const mutations = {
  RESET_STATE: (state) => {
    Object.assign(state, getDefaultState())
  },
  SET_TOKEN: (state, token) => {
    state.token = token
  },
  SET_NAME: (state, name) => {
    state.name = name
  },
  SET_SERVER_ID: (state, serverId) => {
    state.serverId = serverId
  },
  SET_MENUS: (state, menus) => {
    state.menus = menus || []
  },
  SET_ROLE_ID: (state, roleId) => {
    state.roleId = roleId || 0
  },
  SET_CONFIRM_BOX: (state, status) => {
    state.showConfirmBoxForLoginLose = status
  }
}

function applyProfile(commit, data) {
  const menus = (data && data.menus) || []
  const roleId = (data && data.role && data.role.id) || 0
  commit('SET_MENUS', menus)
  commit('SET_ROLE_ID', roleId)
  setMenus(menus)
  setRoleId(roleId)
  if (data && data.username) {
    commit('SET_NAME', data.username)
    setName(data.username)
  }
  if (data && data.serverId) {
    commit('SET_SERVER_ID', data.serverId)
    setServerId(data.serverId)
  }
}

const actions = {
  login({ commit }, userInfo) {
    const { username, password } = userInfo
    return new Promise((resolve, reject) => {
      login({
        username: username.trim(),
        password: crypto.createHash('md5').update(password, 'utf8').digest('hex')
      }).then(response => {
        const { data } = response
        commit('SET_TOKEN', data.accessToken)
        commit('SET_CONFIRM_BOX', true)
        setToken(data.accessToken)
        applyProfile(commit, data)
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  logout({ commit, state }) {
    return new Promise((resolve, reject) => {
      logout(state.token).then(() => {
        removeToken()
        removeServerId()
        removeName()
        removeMenus()
        removeRoleId()
        resetRouter()
        commit('RESET_STATE')
        resolve()
      }).catch(error => {
        reject(error)
      })
    })
  },

  resetToken({ commit }) {
    return new Promise(resolve => {
      removeToken()
      removeMenus()
      removeRoleId()
      commit('RESET_STATE')
      resolve()
    })
  },

  getUserInfo({ commit }) {
    return new Promise((resolve, reject) => {
      getUserInfo().then(response => {
        const { data } = response
        applyProfile(commit, data)
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },

  changePushKey({ commit }, params) {
    return new Promise((resolve, reject) => {
      changePushKey(params).then(response => {
        resolve(response.data)
      }).catch(reject)
    })
  },

  queryList({ commit }, params) {
    return new Promise((resolve, reject) => {
      queryList(params).then(response => {
        resolve(response.data)
      }).catch(reject)
    })
  },

  removeById({ commit }, id) {
    return new Promise((resolve, reject) => {
      removeById(id).then(response => {
        resolve(response.data)
      }).catch(reject)
    })
  },

  add({ commit }, params) {
    return new Promise((resolve, reject) => {
      add(params).then(response => {
        resolve(response.data)
      }).catch(reject)
    })
  },

  changePassword({ commit }, params) {
    return new Promise((resolve, reject) => {
      changePassword(params).then(response => {
        resolve(response.data)
      }).catch(reject)
    })
  },

  changePasswordForAdmin({ commit }, params) {
    return new Promise((resolve, reject) => {
      changePasswordForAdmin(params).then(response => {
        resolve(response.data)
      }).catch(reject)
    })
  },
  closeConfirmBoxForLoginLose({ commit }) {
    commit('SET_CONFIRM_BOX', false)
  }
}

export default {
  namespaced: true,
  state,
  mutations,
  actions
}
