// Pathify
import { make } from 'vuex-pathify'
import WebsocketService from '@/services/websocket.service'

// Data
const state = {
  services: {},
  side: '',
  lastseen: Date.now(),
  statechange: Date.now(),
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
      if (!msg.identity || msg.identity === '') msg.identity = 'default'
      if (!state.services[name]) state.services[name] = {}
      var prevstate = state.services[name] && state.services[name][msg.identity] ? state.services[name][msg.identity].state : 'unknown'
      state.services[name][msg.identity] = { appname: msg.appname, id: msg.identity, msg: msg, alive: true, lastseen: new Date(), state: 'alive', count: 0 }
      state.services[name].alive = true

      state.lastseen = Date.now() // this is a crappy workaround to have the services state updated in other components
      if (prevstate !== state.services[name][msg.identity].state) state.statechange = Date.now()

      // Also not the nicest construct, but since we're using the same UI for both sides, we need this indicator
      if (name === 'ddnatsinnerproxy') state.side = 'inner'
      if (name === 'ddnatsouterproxy') state.side = 'outer'
    })

    // Need to run independently from the heartbeat subscription to be able to
    // detect stalling and dead services (which does not send a heartbeat)
    setInterval(function () {
      var now = new Date()
      for (const p in state.services) {
        for (const i in state.services[p]) {
          if (!state.services[p][i].lastseen || state.services[p][i].state === 'dead') continue
          var diff = Math.abs(now.getTime() - state.services[p][i].lastseen.getTime()) / 1000

          if (diff > 4 && diff <= 8) {
            state.services[p][i].state = 'stalling'
            state.statechange = Date.now()
          } else if (diff > 8) {
            // console.log('service: ' + p + ', instance: ' + i + ', diff: ' + diff + ', now: ' + now + ', lastseen: ' + state.services[p][i].lastseen)
            state.services[p][i].state = 'dead'
            state.services[p][i].alive = false
            state.statechange = Date.now()
          }
        }

        // Check all instances again to set the aggregated alive state correctly
        var anyalive = false
        for (const i in state.services[p]) {
          if (state.services[p][i].state === 'alive') {
            anyalive = true
            break
          }
        }

        state.services[p].alive = anyalive

        // console.log('service ' + state.services[p].appname + ', state: ' + state.services[p].state)
      }
    }, 2000)
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
