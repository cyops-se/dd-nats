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
      <v-list-item v-if="lastmsg !== 'KALLE'">
        {{ lastmsg }}
      </v-list-item>
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
    }),

    created () {
      WebsocketService.topic('system.log.*', this, function (topic, msg, t) {
        t.lastmsg = msg.time.replace('T', ' ').substr(0, 22) + ': ' + msg.title + ' - ' + msg.description
      })
    },
  }
</script>
