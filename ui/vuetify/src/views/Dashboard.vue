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
        <v-row>
          <v-col
            v-for="(group, i) in groups"
            :key="`group-${i}`"
            cols="12"
            md="6"
            lg="4"
          >
            <material-group-card :group="group" />
          </v-col>
        </v-row>
      </v-col>
      <!-- error-logs-tables-view / -->
    </v-row>
  </v-container>
</template>

<script>
  // Utilities
  import ErrorLogsTablesView from './ErrorLogs'
  import ApiService from '@/services/api.service'
  import WebsocketService from '@/services/websocket.service'

  export default {
    name: 'DashboardView',

    components: {
      ErrorLogsTablesView,
    },

    data: () => ({
      stats: '',
      groups: [],
    }),

    computed: {
    },

    created () {
      var request = { subject: 'usvc.opc.groups.getall', payload: { value: parseInt(this.$route.params.serverid) } }
      ApiService.post('nats/request', request)
        .then(response => {
          this.groups = response.data.items
        }).catch(response => {
          console.log('ERROR response: ' + response.message)
          this.$notification.error('Failed to get groups: ' + response.message)
        })
    },

    createdX () {
      WebsocketService.topic('stats.nats.totmsgs', this, function (topic, message, t) {
        var msg = JSON.parse(message)
        t.stats = message
      })
    },
  }
</script>
