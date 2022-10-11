<template>
  <v-container
    id="dashboard-view"
    fluid
    tag="section"
  >
    <v-row>
      <v-col cols="12">
        <v-alert
          :color="timescale ? 'success' : 'error'"
        >
          TIMESCALE service {{ timescale ? 'RUNNING' : 'STOPPED' }}
        </v-alert>
        <v-alert
          :color="rabbitmq ? 'success' : 'error'"
        >
          RABBITMQ service {{ rabbitmq ? 'RUNNING' : 'STOPPED' }}
        </v-alert>
      </v-col>
    </v-row>
    <v-row>
      <v-col cols="12">
        <v-card v-if="logger">
          <v-card-title class="text-h4">
            Warnings
            <v-spacer />
            <v-text-field
              v-model="search"
              append-icon="mdi-magnify"
              label="Search"
              single-line
              hide-details
            />
          </v-card-title>
          <log-list
            method="getcategory"
            payload="{&quot;category&quot;: &quot;warning&quot;}"
            :search="search"
          />
        </v-card>
        <v-alert
          v-if="!logger"
          elevation="15"
          prmoinent
          shaped
          type="warning"
        >
          No logging service active!
        </v-alert>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
  // Utilities
  import { sync } from 'vuex-pathify'

  export default {
    name: 'OuterDashboard',

    components: {
    },

    data: () => ({
      search: '',
      logger: false,
      timescale: false,
      rabbitmq: false,
    }),

    computed: {
      ...sync('usvc', [
        'services',
        'lastseen',
        'side',
      ]),
    },

    watch: {
      lastseen (to, from) {
        if (this.side === 'outer') this.refresh()
        this.logger = this.services.ddlogger && this.services.ddlogger.alive
        this.timescale = this.services.ddnatstimescale && this.services.ddnatstimescale.alive
        this.rabbitmq = this.services.ddnatsrabbitmq && this.services.ddnatsrabbitmq.alive
      },
    },

    created () {
    },

    methods: {
      refresh () {
        // var request = { subject: 'usvc.opc.groups.getall', payload: { value: parseInt(this.$route.params.serverid) } }
        // ApiService.post('nats/request', request)
        //   .then(response => {
        //     this.groups = response.data.items
        //   }).catch(response => {
        //     console.log('ERROR response: ' + response.message)
        //   })

        // request = { subject: 'usvc.modbus.slaves.getall', payload: { value: parseInt(this.$route.params.serverid) } }
        // ApiService.post('nats/request', request)
        //   .then(response => {
        //     this.slaves = response.data.items
        //   }).catch(response => {
        //     console.log('ERROR response: ' + response.message)
        //   })
      },
    },
  }
</script>
