<template>
  <v-container fluid>
    <v-data-table
      :headers="headers"
      :items="items"
      :search="search"
      sort-by="path"
      class="elevation-1 text-no-wrap"
    >
      <template v-slot:top>
        <v-toolbar
          flat
        >
          <v-toolbar-title>Received files</v-toolbar-title>
          <v-spacer />
          <v-text-field
            v-model="search"
            append-icon="mdi-magnify"
            label="Search"
            single-line
            hide-details
          />
        </v-toolbar>
      </template>
    </v-data-table>
    <v-card v-if="progress && progress.file">
      <v-card-title>{{ progress.file.path }}/{{ progress.file.name }}<v-spacer />{{ progress.percentdone.toFixed(2) }}%</v-card-title>
      <v-card-text>
        <v-progress-linear
          v-model="progress.percentdone"
          color="info"
          height="25"
        />
      </v-card-text>
    </v-card>
  </v-container>
</template>

<script>
  import ApiService from '@/services/api.service'
  import WebsocketService from '@/services/websocket.service'
  export default {
    name: 'FileTransfer',
    data: () => ({
      info: {},
      progress: {},
      inprogress: false,
      uploadDialog: false,
      headers: [
        { text: 'Path', value: 'path', width: '100px' },
        { text: 'Name', value: 'name', width: '80%' },
        { text: 'Size', value: 'size', width: '20%', align: 'right', sortable: false },
        { text: 'Date', value: 'time', width: '20%' },
      ],
      items: [],
      search: '',
    }),

    created () {
      console.log('created called')
      this.refresh()

      var t = this
      WebsocketService.topic('inner.file.start', this, function (topic, info) {
        t.progress = { file: info, percentdone: 0.0 }
      })

      WebsocketService.topic('inner.file.block.*', this, function (topic, progress) {
        console.log('progress: ' + JSON.stringify(progress))
        t.progress = { file: progress, percentdone: progress.percent }
      })

      WebsocketService.topic('inner.file.end', this, function (topic, info) {
        t.progress = undefined
      })
    },

    methods: {

      refresh () {
        var request = { subject: 'usvc.filetransfer.getmanifest', payload: {} }
        ApiService.post('nats/request', request)
          .then((response) => {
            this.items = response.data.manifest.files
            this.items.forEach((item) => {
              item.time = item.time.replace('T', ' ').substring(0, 19)
              item.size = this.sizeToHuman(item.size)
            })
            this.loading = false
          })
          .catch((e) => {
            console.log('ERROR response: ' + JSON.stringify(e.message))
          })
      },

      sizeToHuman (size) {
        if (size > 1024 * 1024) return (size / (1024 * 1024)).toFixed(2) + ' MB'
        if (size > 1024) return (size / (1024)).toFixed(2) + ' KB'
        return size + ' bytes'
      },
    },
  }
</script>
