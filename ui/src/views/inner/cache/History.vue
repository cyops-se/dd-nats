<template>
  <v-container fluid>
    <v-row>
      <v-col>
        <simple-info-card
          icon="mdi-clipboard-file-outline"
          color="primary"
          :value="info.count.toString()"
          title="Files in cache"
        />
      </v-col>
      <v-col>
        <simple-info-card
          icon="mdi-harddisk"
          color="primary"
          :value="info.sizeinmb"
          title="Total size in MB"
        />
      </v-col>
      <v-col md="4">
        <simple-info-card
          icon="mdi-clipboard-clock-outline"
          color="secondary"
          :value="info.firsttime.replace('T', ' ').replace('Z','')"
          title="First available time (UTC)"
        />
      </v-col>
      <v-col md="4">
        <simple-info-card
          icon="mdi-clipboard-clock-outline"
          color="secondary"
          :value="info.lasttime.replace('T', ' ').replace('Z','')"
          title="Last available time (UTC)"
        />
      </v-col>
    </v-row>
    <v-row>
      <v-col>
        <v-data-table
          v-model="selected"
          :headers="headers"
          :items="items"
          item-key="filename"
          :search="search"
          show-select
          class="elevation-1 text-no-wrap"
        >
          <template v-slot:top>
            <v-toolbar flat>
              <v-toolbar-title>Resend tag values</v-toolbar-title>
              <v-divider
                class="mx-4"
                inset
                vertical
              />
              <v-spacer />
              <v-text-field
                v-model="search"
                append-icon="mdi-magnify"
                label="Search"
                single-line
                hide-details
              />
              <v-btn
                color="success"
                dark
                class="ml-3"
                :disabled="selected.length === 0"
                @click="resend"
              >
                Send again
              </v-btn>
            </v-toolbar>
          </template>
        </v-data-table>
      </v-col>
    </v-row>
  </v-container>
</template>

<!-- template>
  <v-data-table
    :headers="headers"
    :items="items"
    class="elevation-1"
  >
    <template v-slot:top>
      <v-toolbar flat>
        <v-toolbar-title>Process history</v-toolbar-title>
        <v-divider
          class="mx-4"
          inset
          vertical
        />
        <v-spacer />
      </v-toolbar>
    </template>
    <template v-slot:item.actions="{ item }">
      <v-icon
        class="mr-2"
        @click="editItem(item)"
      >
        mdi-pencil
      </v-icon>
      <!- - v-icon @click="deleteItem(item)">
        mdi-delete
      </v-icon - ->
    </template>
  </v-data-table>
</template -->

<script>
  import ApiService from '@/services/api.service'
  export default {
    name: 'ProcessHistory',

    data: () => ({
      search: '',
      loading: false,
      headers: [
        { text: 'Time (UTC)', value: 'time', width: '110px' },
        { text: 'Filename', value: 'filename', width: '90%' },
        { text: 'Size', value: 'size', width: '110px' },
      ],
      info: { count: '', sizeinmb: '', firsttime: '', lasttime: '' },
      items: [],
      selected: [],
      selecteditems: [],
    }),

    created () {
      this.refresh()
    },

    methods: {
      initialize () {},

      refresh () {
        this.loading = true
        var request = { subject: 'usvc.cache.getall', payload: {} }
        ApiService.post('nats/request', request)
          .then((response) => {
            // console.log('response: ' + JSON.stringify(response))
            this.info = response.data.info
            this.info.sizeinmb = (this.info.size / (1024 * 1024)).toFixed(2)

            this.items = this.info.items
            for (var i = 0; i < this.items.length; i++) {
              this.items[i].time = this.items[i].time.replace('T', ' ').replace('Z', '')
            }
            this.loading = false
          })
          .catch((e) => {
            console.log('ERROR response: ' + JSON.stringify(e.message))
          })
      },

      resend () {
        ApiService.post('system/resend', this.selected)
          .then((response) => {
            this.$notification.success(
              'Number of resent files: ' + response.data.count,
            )
          })
          .catch((e) => {
            console.log('ERROR response: ' + JSON.stringify(e.message))
          })
      },
    },
  }
</script>
