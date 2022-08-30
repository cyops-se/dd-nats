// Services
import ApiService from '@/services/api.service'

// Utilities
import { make } from 'vuex-pathify'

// Globals
import { IN_BROWSER } from '@/util/globals'

const state = {
  ID: 0,
  email: '[unknown]',
  fullname: '[unknown]',
  settings: { dark: false },
}

const mutations = make.mutations(state)

const actions = {
  refresh: ({ commit }) => {
    if (!IN_BROWSER) return

    ApiService.get('user/current')
    .then((data) => {
      var user = data.data
      for (const key in user) {
        if (state[key]) {
          commit(key, user[key])
        }
      }
    })
    .catch((e) => { console.log('user update failed', e) })

    localStorage.setItem('vuetify@user', JSON.stringify(state))
  },
  populate: ({ commit }, data) => {
    if (!IN_BROWSER) return

    for (const key in data) {
      commit(key, data[key])
    }
    localStorage.setItem('vuetify@user', JSON.stringify(state))
  },
  update: ({ commit }) => {
    if (!IN_BROWSER) return
    for (const key in state) {
      commit(key, state[key])
    }

    localStorage.setItem('vuetify@user', JSON.stringify(state))
    ApiService.put('data/settings', state.settings)
    .then((data) => { })
    .catch((e) => { console.log('user setting update failed') })
  },
}

const getters = {
  dark: (state) => {
    return (
      state.settings.dark
    )
  },
  settings: (state) => {
    return (
      state.settings
    )
  },
  user: (state) => {
    return (
      state
    )
  },
}

export default {
  namespaced: true,
  state,
  mutations,
  actions,
  getters,
}
