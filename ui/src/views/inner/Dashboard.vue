<template>
  <v-container
    id="dashboard-view"
    fluid
    tag="section"
  >
    <v-row>
      <v-col cols="12">
        <v-alert
          :color="opcda ? 'success' : 'error'"
        >
          OPCDA service {{ opcda ? 'RUNNING' : 'STOPPED' }}
        </v-alert>
        <v-alert
          :color="modbus ? 'success' : 'error'"
        >
          MODBUS service {{ modbus ? 'RUNNING' : 'STOPPED' }}
        </v-alert>
      </v-col>
    </v-row>
    <!-- <v-row>
      <v-col cols="12">
        <v-alert
          v-if="(!groups || groups.length === 0)"
          elevation="15"
          prmoinent
          shaped
          type="warning"
        >
          Warning! No active data collectors!
        </v-alert>
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
    </v-row> -->
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
  import ApiService from '@/services/api.service'

  export default {
    name: 'InnerDashboard',

    components: {
    },

    data: () => ({
      stats: '',
      groups: [],
      slaves: [],
      search: '',
      logger: false,
      opcda: false,
      modbus: false,
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
        this.logger = this.services.ddlogger && this.services.ddlogger.alive
        this.opcda = this.services.ddnatsopcda && this.services.ddnatsopcda.alive
        this.modbus = this.services.ddnatsmodbus && this.services.ddnatsmodbus.alive
      },
    },

    created () {
    },

    methods: {
      refresh () {
        if (this.selected) {
          var request = { subject: 'usvc.opc.' + this.selected.key + '.groups.getall', payload: { value: parseInt(this.$route.params.serverid) } }
          ApiService.post('nats/request', request)
            .then(response => {
              this.groups = response.data.items
            }).catch(response => {
              console.log('ERROR response: ' + response.message)
            })
        }

        request = { subject: 'usvc.modbus.slaves.getall', payload: { value: parseInt(this.$route.params.serverid) } }
        ApiService.post('nats/request', request)
          .then(response => {
            this.slaves = response.data.items
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
          })
      },
    },
  }
</script>
