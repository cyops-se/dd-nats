// Pathify
import { make } from 'vuex-pathify'

// Data
const state = {
  services: [],
}

const mutations = make.mutations(state)

const actions = {
  ...make.actions(state),
  init: async ({ dispatch }) => {
    console.log('usvc/init')
  },
}

const getters = make.getters(state)
console.log('usvc module loaded ...')

export default {
  namespaced: true,
  state,
  mutations,
  actions,
  getters,
}
