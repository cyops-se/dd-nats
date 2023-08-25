// Pathify
import { make } from 'vuex-pathify'
import ApiService from '@/services/api.service'

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
    console.log('app/init')

    // Get system information
    ApiService.get('system/sysinfo')
      .then(response => {
        state.sysinfo = response.data
      }).catch(e => {
        console.log('ERROR response: ' + JSON.stringify(e.message))
      })
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
