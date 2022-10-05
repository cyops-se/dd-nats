<template>
  <v-container fluid>
    <v-row>
      <v-col>
        <simple-info-card
          icon="mdi-clipboard-file-outline"
          color="primary"
          :value="sysinfo.cacheinfo.count.toFixed(0)"
          title="Files in cache"
        />
      </v-col>
      <v-col>
        <simple-info-card
          icon="mdi-harddisk"
          color="primary"
          :value="sysinfo.cacheinfo.sizeinmb"
          title="Total size in MB"
        />
      </v-col>
      <v-col md="4">
        <simple-info-card
          icon="mdi-clipboard-clock-outline"
          color="secondary"
          :value="sysinfo.cacheinfo.firsttime.replace('T', ' ').replace('Z','')"
          title="First available time (UTC)"
        />
      </v-col>
      <v-col md="4">
        <simple-info-card
          icon="mdi-clipboard-clock-outline"
          color="secondary"
          :value="sysinfo.cacheinfo.lasttime.replace('T', ' ').replace('Z','')"
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
          class="elevation-1"
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

<script>
  import { sync } from 'vuex-pathify'
  import ApiService from '@/services/api.service'
  export default {
    name: 'History',
    data: () => ({
      saveDisabled: false,
      search: '',
      loading: false,
      headers: [
        { text: 'Time (UTC)', value: 'ptime', width: '20%' },
        { text: 'Name', value: 'filename', width: '60%' },
        { text: 'Size (bytes)', value: 'size', width: '20%' },
      ],
      items: [],
      selected: [],
      selecteditems: [],
    }),

    computed: {
      ...sync('app', ['sysinfo']),
    },

    created () {
      ApiService.get('system/info')
        .then((response) => {
          this.sysinfo = response.data
          this.items = this.sysinfo.cacheinfo.items
          this.sysinfo.cacheinfo.sizeinmb = (this.sysinfo.cacheinfo.size / (1024 * 1024)).toFixed(2)
          if (this.items) {
            this.items.forEach((item) => { item.ptime = item.time.replace('T', ' ').replace('Z', '') })
          }
        })
        .catch((response) => {
          console.log('ERROR response: ' + JSON.stringify(response))
        })
    },

    methods: {
      resend () {
        ApiService.post('system/resend', this.selected)
          .then((response) => {
            this.$notification.success(
              'Number of resent files: ' + response.data.count,
            )
          })
          .catch((response) => {
            console.log('ERROR response: ' + JSON.stringify(response))
          })
      },
    },
  }
</script>
