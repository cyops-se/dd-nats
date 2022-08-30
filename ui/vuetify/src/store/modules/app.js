// Pathify
import { make } from 'vuex-pathify'

// Data
const state = {
  version: '0.0.1',
  drawer: null,
  drawerImage: false,
  mini: false,
  sysinfo: {},
}

const mutations = make.mutations(state)

const actions = {
  ...make.actions(state),
  init: async ({ dispatch }) => {
  },
}

const getters = {}

export default {
  namespaced: true,
  state,
  mutations,
  actions,
  getters,
}
