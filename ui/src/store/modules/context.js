// Pathify
import { make } from 'vuex-pathify'

// Data
const state = {
  selected: '',
}

const mutations = {
  ...make.mutations(state),
}

const actions = {
  ...make.actions(state),
  init: async ({ dispatch }) => {
    console.log('context/init')
  },
}

const getters = make.getters(state)

console.log('context module loaded ...')

export default {
  namespaced: true,
  state,
  mutations,
  actions,
  getters,
}
