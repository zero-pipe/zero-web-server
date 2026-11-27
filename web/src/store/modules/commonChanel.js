import {
  update,
  add,
  reset,
  queryOne,
  addDeviceToGroup,
  deleteDeviceFromGroup,
  addDeviceToRegion,
  deleteDeviceFromRegion,
  getCivilCodeList,
  getParentList,
  getUnusualParentList,
  clearUnusualParentList,
  getUnusualCivilCodeList,
  clearUnusualCivilCodeList,
  getIndustryList,
  getTypeList,
  getNetworkIdentificationList,
  addToRegion,
  deleteFromRegion,
  addToGroup,
  deleteFromGroup,
  getList,
  getAllForMap, test, saveLevel, resetLevel, clearThin, thinProgress, drawThin, saveThin
} from '@/api/commonChannel'

// 说明：播放/云台/预置位/回放已统一收敛到 /api/v1/resources/{resourceId}/*，
// 对应调用直接走 api/asset.js 与 api/playback.js 的封装，不再经 vuex 转发。
// 本模块只保留通道管理类（增删改查/区划/分组/地图）动作。

const actions = {
  update({ commit }, formData) {
    return new Promise((resolve, reject) => {
      update(formData).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  add({ commit }, formData) {
    return new Promise((resolve, reject) => {
      add(formData).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  reset({ commit }, data) {
    return new Promise((resolve, reject) => {
      reset(data).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  queryOne({ commit }, id) {
    return new Promise((resolve, reject) => {
      queryOne(id).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  addDeviceToGroup({ commit }, params) {
    return new Promise((resolve, reject) => {
      addDeviceToGroup(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  addToGroup({ commit }, params) {
    return new Promise((resolve, reject) => {
      addToGroup(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  deleteDeviceFromGroup({ commit }, deviceIds) {
    return new Promise((resolve, reject) => {
      deleteDeviceFromGroup(deviceIds).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  deleteFromGroup({ commit }, channels) {
    return new Promise((resolve, reject) => {
      deleteFromGroup(channels).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  addDeviceToRegion({ commit }, params) {
    return new Promise((resolve, reject) => {
      addDeviceToRegion(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  addToRegion({ commit }, params) {
    return new Promise((resolve, reject) => {
      addToRegion(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  deleteDeviceFromRegion({ commit }, deviceIds) {
    return new Promise((resolve, reject) => {
      deleteDeviceFromRegion(deviceIds).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  deleteFromRegion({ commit }, channels) {
    return new Promise((resolve, reject) => {
      deleteFromRegion(channels).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  getCivilCodeList({ commit }, params) {
    return new Promise((resolve, reject) => {
      getCivilCodeList(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  getParentList({ commit }, params) {
    return new Promise((resolve, reject) => {
      getParentList(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  getUnusualParentList({ commit }, params) {
    return new Promise((resolve, reject) => {
      getUnusualParentList(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  clearUnusualParentList({ commit }, params) {
    return new Promise((resolve, reject) => {
      clearUnusualParentList(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  getUnusualCivilCodeList({ commit }, params) {
    return new Promise((resolve, reject) => {
      getUnusualCivilCodeList(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  clearUnusualCivilCodeList({ commit }, params) {
    return new Promise((resolve, reject) => {
      clearUnusualCivilCodeList(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  getIndustryList({ commit }) {
    return new Promise((resolve, reject) => {
      getIndustryList().then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  getTypeList({ commit }) {
    return new Promise((resolve, reject) => {
      getTypeList().then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  getNetworkIdentificationList({ commit }) {
    return new Promise((resolve, reject) => {
      getNetworkIdentificationList().then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  getList({ commit }, param) {
    return new Promise((resolve, reject) => {
      getList(param).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  getAllForMap({ commit }, params) {
    return new Promise((resolve, reject) => {
      getAllForMap(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  saveLevel({ commit }, data) {
    return new Promise((resolve, reject) => {
      saveLevel(data).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  resetLevel({ commit }) {
    return new Promise((resolve, reject) => {
      resetLevel().then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  clearThin({ commit }, id) {
    return new Promise((resolve, reject) => {
      clearThin(id).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  thinProgress({ commit }, id) {
    return new Promise((resolve, reject) => {
      thinProgress(id).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  saveThin({ commit }, id) {
    return new Promise((resolve, reject) => {
      saveThin(id).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  drawThin({ commit }, param) {
    return new Promise((resolve, reject) => {
      drawThin(param).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },

  test({ commit }) {
    return new Promise((resolve, reject) => {
      test().then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  }
}

export default {
  namespaced: true,
  actions
}
