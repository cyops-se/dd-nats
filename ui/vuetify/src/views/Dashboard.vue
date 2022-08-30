<template>
  <v-container
    id="dashboard-view"
    fluid
    tag="section"
  >
    <v-row>
      <v-col cols="12">
        <v-row>
          <v-col
            cols="12"
            md="6"
            lg="4"
          >
            <div>{{ stats }}</div>
          </v-col>
        </v-row>
      </v-col>
      <error-logs-tables-view />
    </v-row>
  </v-container>
</template>

<script>
  // Utilities
  import ErrorLogsTablesView from './ErrorLogs'
  import WebsocketService from '@/services/websocket.service'

  export default {
    name: 'DashboardView',

    components: {
      ErrorLogsTablesView,
    },

    data: () => ({
      stats: '',
    }),

    computed: {
    },

    created () {
      WebsocketService.topic('stats.nats.totmsgs', this, function (topic, message, t) {
        // console.log(JSON.stringify(message))
        var msg = JSON.parse(message)
        t.stats = message
      })
    },
  }
</script>
