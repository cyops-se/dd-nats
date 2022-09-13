<template>
  <v-btn
    x-large
    :color="alive ? 'success' : 'error'"
  >
    <v-icon>{{ icon }}</v-icon>
  </v-btn>
</template>

<script>
  import WebsocketService from '@/services/websocket.service'
  export default {
    name: 'UsvcMiniCard',

    props: {
      usvc: String,
    },

    data: () => ({
      alive: false,
      lastseen: undefined,
      appname: '',
      icon: 'mdi-access-point-off',
      msg: '',
      services: [],
    }),

    created () {
      WebsocketService.topic('system.heartbeat', this, function (topic, msg, t) {
        if (t.services) {
          var appname = msg.appname.replaceAll('-', '')
          t.services = { ...t.services, [appname]: { name: appname, state: 'alive', count: 0, lastbeat: new Date() } }
          if (msg.appname === t.usvc) {
            t.appname = msg.appname
            t.msg = JSON.stringify(msg)
            t.alive = true
            t.lastseen = new Date(msg.timestamp)
            t.icon = t.alive ? 'mdi-access-point' : 'mdi-access-point-off'
          }
        }
      })

      var t = this
      setInterval(function () {
        var now = new Date()
        if (!t.lastseen || Math.abs(now.getSeconds() - t.lastseen.getSeconds()) > 5) {
          t.alive = false
          t.icon = 'mdi-access-point-off'
        }
      }, 5000)
    },
  }
</script>

<style lang="sass">
  .v-card.v-card--material
    > .v-card__title
      > .v-card--material__title
        flex: 1 1 auto
        word-break: break-word
</style>
