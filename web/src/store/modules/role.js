import { addRole, deleteRole, getAll, getMenus, updateRole } from '@/api/role'

const actions = {
  getAll({ commit }) {
    return new Promise((resolve, reject) => {
      getAll().then(response => {
        resolve(response.data)
      }).catch(reject)
    })
  },
  getMenus({ commit }) {
    return new Promise((resolve, reject) => {
      getMenus().then(response => {
        resolve(response.data)
      }).catch(reject)
    })
  },
  add({ commit }, data) {
    return new Promise((resolve, reject) => {
      addRole(data).then(response => {
        resolve(response.data)
      }).catch(reject)
    })
  },
  update({ commit }, data) {
    return new Promise((resolve, reject) => {
      updateRole(data).then(response => {
        resolve(response.data)
      }).catch(reject)
    })
  },
  remove({ commit }, id) {
    return new Promise((resolve, reject) => {
      deleteRole(id).then(response => {
        resolve(response.data)
      }).catch(reject)
    })
  }
}

export default {
  namespaced: true,
  actions
}
