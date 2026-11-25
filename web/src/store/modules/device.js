import {
  getKeepaliveTimeStatistics,
  getRegisterTimeStatistics,
  queryDeviceSyncStatus,
  queryDevices,
  queryHasStreamChannels,
  resetGuard,
  setGuard,
  subscribeCatalog,
  subscribeForAlarm,
  subscribeMobilePosition,
  sync
} from '@/api/device'

const actions = {
  queryDeviceSyncStatus({ commit }, deviceId) {
    return new Promise((resolve, reject) => {
      queryDeviceSyncStatus(deviceId).then(response => {
        resolve(response)
      }).catch(error => {
        reject(error)
      })
    })
  },
  queryDevices({ commit }, params) {
    return new Promise((resolve, reject) => {
      queryDevices(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  sync({ commit }, deviceId) {
    return new Promise((resolve, reject) => {
      sync(deviceId).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  setGuard({ commit }, deviceId) {
    return new Promise((resolve, reject) => {
      setGuard(deviceId).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  resetGuard({ commit }, deviceId) {
    return new Promise((resolve, reject) => {
      resetGuard(deviceId).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  subscribeCatalog({ commit }, params) {
    return new Promise((resolve, reject) => {
      subscribeCatalog(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  subscribeMobilePosition({ commit }, params) {
    return new Promise((resolve, reject) => {
      subscribeMobilePosition(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  subscribeForAlarm({ commit }, params) {
    return new Promise((resolve, reject) => {
      subscribeForAlarm(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  queryHasStreamChannels({ commit }, params) {
    return new Promise((resolve, reject) => {
      queryHasStreamChannels(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  getKeepaliveTimeStatistics({ commit }, params) {
    return new Promise((resolve, reject) => {
      getKeepaliveTimeStatistics(params).then(response => {
        const { data } = response
        resolve(data)
      }).catch(error => {
        reject(error)
      })
    })
  },
  getRegisterTimeStatistics({ commit }, params) {
    return new Promise((resolve, reject) => {
      getRegisterTimeStatistics(params).then(response => {
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
