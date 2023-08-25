<template>
  <v-container
    id="logs-view"
    fluid
    tag="section"
  >
    <v-card>
      <v-card-title class="text-h4">
        System logs
        <v-spacer />
        <v-text-field
          v-model="search"
          append-icon="mdi-magnify"
          label="Search"
          single-line
          hide-details
        />
        <v-btn
          color="primary"
          dark
          class="ml-4"
          @click="clearAll"
        >
          Clear all entries
        </v-btn>
      </v-card-title>
      <log-list
        method="getcategory"
        payload="{&quot;category&quot;: &quot;info&quot;}"
        :search="search"
      />
    </v-card>
  </v-container>
</template>

<script>
  import ApiService from '@/services/api.service'
  export default {
    name: 'LogsTablesView',

    data: () => ({
      search: '',
    }),

    mounted () {
    },

    methods: {
      clearAll () {
        ApiService.delete('data/logs')
          .then(response => {
            this.$notification.success('Log entries cleared!')
            this.refresh()
          }).catch(e => {
            console.log('ERROR response: ' + JSON.stringify(e.message))
          })
      },
    },
  }
</script>
