<template>
  <v-container
    id="dashboard-view"
    fluid
    tag="section"
  >
    <v-row>
      <v-col cols="12">
        <v-alert
          v-if="!groups || groups.length === 0"
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
    </v-row>
    <v-row>
      <v-col cols="12">
        <v-card>
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
          />
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
  // Utilities
  import { sync } from 'vuex-pathify'
  import ErrorLogsView from './logs/ErrorLogs'
  import ApiService from '@/services/api.service'

  export default {
    name: 'DashboardView',

    components: {
      ErrorLogsView,
    },

    data: () => ({
      stats: '',
      groups: [],
      slaves: [],
      search: '',
    }),

    computed: {
      ...sync('usvc', [
        'services',
        'lastseen',
      ]),
    },

    created () {
      this.refresh()
    },

    methods: {
      refresh () {
        var request = { subject: 'usvc.opc.groups.getall', payload: { value: parseInt(this.$route.params.serverid) } }
        ApiService.post('nats/request', request)
          .then(response => {
            this.groups = response.data.items
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
          })

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
