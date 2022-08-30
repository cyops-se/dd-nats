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
      <v-data-table
        :headers="headers"
        :items="items"
        :search="search"
        :loading="loading"
        loading-text="Loading... Please wait"
        sort-by="time"
        :sort-desc="sortDesc"
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
      loading: false,
      headers: [
        {
          text: 'Time',
          align: 'start',
          filterable: true,
          value: 'time',
          width: 200,
        },
        { text: 'Category', value: 'category', width: '10%' },
        { text: 'Title', value: 'title', width: '20%' },
        { text: 'Description', value: 'description', width: '60%' },
      ],
      items: [],
      sortDesc: true,
    }),

    mounted () {
      this.refresh()
    },

    methods: {
      refresh () {
        ApiService.get('data/logs')
          .then(response => {
            for (const i of response.data) {
              i.time = i.time.replace('T', ' ').replace('Z', '').substring(0, 19)
            }
            this.items = response.data
            this.loading = false
          }).catch(response => {
            console.log('ERROR response: ' + JSON.stringify(response))
          })
      },

      clearAll () {
        ApiService.delete('data/logs')
          .then(response => {
            this.$notification.success('Log entries cleared!')
            this.refresh()
          }).catch(response => {
            console.log('ERROR response: ' + JSON.stringify(response))
          })
      },
    },
  }
</script>
