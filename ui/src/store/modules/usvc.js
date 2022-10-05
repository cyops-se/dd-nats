// Pathify
import { make } from 'vuex-pathify'
import WebsocketService from '@/services/websocket.service'

// Data
const state = {
  services: {},
  lastseen: Date.now(),
}

const mutations = {
  ...make.mutations(state),
  setusvc (state, payload) {
    state.services[payload.name] = payload
  },
}

const actions = {
  ...make.actions(state),
  init: async ({ dispatch }) => {
    console.log('usvc/init')

    WebsocketService.topic('system.heartbeat', this, function (topic, msg, t) {
      var name = msg.appname.replaceAll('-', '')
        state.services[name] = { appname: msg.appname, msg: msg, alive: true, lastseen: new Date(msg.timestamp), state: 'alive', count: 0 }
        state.lastseen = Date.now() // this is a crappy workaround to have the services state updated in other components
    })

    setInterval(function () {
      var now = new Date()
      for (const p in state.services) {
        if (!state.services[p].lastseen || state.services[p].state === 'dead') continue
        var n = now.getSeconds()
        var lb = state.services[p].lastseen.getSeconds()
        var diff = Math.abs(n - lb)
        if (diff > 2) {
          state.services[p].state = 'stalling'
          if (state.services[p].count++ > 3) {
            state.services[p].state = 'dead'
          }
        }
      }
    }, 1000)
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
