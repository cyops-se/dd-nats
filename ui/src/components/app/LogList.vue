<template>
  <v-container
    fluid
  >
    <v-data-table
      :headers="headers"
      :items="items"
      :search="search"
      :loading="loading"
      loading-text="Loading... Please wait"
      sort-by="time"
      :sort-desc="sortDesc"
    />
  </v-container>
</template>

<script>
  import ApiService from '@/services/api.service'
  export default {
    name: 'LogList',

    props: {
      method: String,
      payload: String,
      search: String,
    },

    data: () => ({
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
        var payload = JSON.parse(this.payload || '{}')
        var request = { subject: 'usvc.logs.' + this.method, payload: payload }
        ApiService.post('nats/request', request)
          .then(response => {
            if (response.data.success && response.data.items !== null) {
              this.items = response.data.items
              for (const i of this.items) {
                i.time = i.time.replace('T', ' ').replace('Z', '').substring(0, 19)
              }
            }
          }).catch(e => {
            console.log('ERROR response: ' + e.message)
            this.$notification.error('Failed to get groups: ' + e.message)
          })
      },

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
