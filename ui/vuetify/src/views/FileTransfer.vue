<template>
  <v-container fluid>
    <v-btn
      color="primary"
      dark
      class="mb-3"
      @click="uploadDialog = !uploadDialog"
    >
      Upload file
    </v-btn>
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
    <file-drop
      :dialog.sync="uploadDialog"
      :multiple="false"
      text="Drop files you want to transfer through the diode here!"
      @filesUploaded="processUpload($event)"
    />
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
    }),

    created () {
      var t = this
      WebsocketService.topic('system.event.filetransfer.request', this, function (topic, info) {
        t.progress = { file: info, percentdone: 0.0 }
      })

      WebsocketService.topic('system.event.filetransfer.progress', this, function (topic, progress) {
        t.progress.percentdone = progress.percent
      })

      WebsocketService.topic('system.event.filetransfer.complete', this, function (topic, info) {
        t.progress = undefined
      })
    },

    methods: {
      processUpload (files) {
        ApiService.upload('filetransfer/upload', files)
          .then(response => {
            this.$notification.success('Files successfully uploaded!')
          }).catch(response => {
            this.$notification.error('Failed to upload file!' + response)
          })
      },
    },
  }
</script>
