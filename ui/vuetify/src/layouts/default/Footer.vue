<template>
  <v-footer
    id="default-footer"
    color="transparent"
    absolute
    app
    inset
  >
    <v-list
      dense
      dark
    >
      <v-list-item>{{ lastmsg }}</v-list-item>
      <v-list-item>{{ errormsg }}</v-list-item>
    </v-list>
  </v-footer>
</template>

<script>
  // Components
  import WebsocketService from '@/services/websocket.service'

  export default {
    name: 'DefaultFooter',

    data: () => ({
      lastmsg: 'KALLE',
      errormsg: 'ERROR',
    }),

    created () {
      WebsocketService.topic('system.log.info', this, function (topic, jsonstr, t) {
        var msg = JSON.parse(jsonstr)
        t.lastmsg = msg.time.replace('T', ' ').substr(0, 22) + ': ' + msg.title + ' - ' + msg.description
      })
      WebsocketService.topic('system.log.error', this, function (topic, jsonstr, t) {
        var msg = JSON.parse(jsonstr)
        t.errormsg = msg.time.replace('T', ' ').substr(0, 22) + ': ' + msg.title + ' - ' + msg.description
      })
    },
  }
</script>
