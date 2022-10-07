<template>
  <div>
    <file-drop
      :dialog.sync="uploadDialog"
      :multiple="false"
      text="Drop your CSV meta files here!"
      @filesUploaded="processUpload($event)"
    />
    <v-data-table
      :headers="headers"
      :items="items"
      :search="search"
      sort-by.sync="sortby"
      class="elevation-1"
    >
      <template v-slot:top>
        <v-toolbar
          flat
        >
          <v-toolbar-title>Tags</v-toolbar-title>
          <v-divider
            class="mx-4"
            inset
            vertical
          />
          <v-spacer />
          <v-dialog
            v-model="dialog"
            max-width="500px"
          >
            <v-card>
              <v-card-title>
                <span class="text-h5">Filtered data point</span>
              </v-card-title>

              <v-card-text>
                <v-container>
                  <v-row>
                    <v-col cols="12">
                      <v-text-field
                        v-model="editedItem.datapoint.n"
                        label="Name"
                        readonly
                      />
                    </v-col>
                    <v-col cols="12">
                      <v-combobox
                        v-model="editedItem.type"
                        :items="availableTypeNames"
                        label="Filter types"
                        outlined
                        hide-details
                      />
                    </v-col>
                  </v-row>
                  <v-row v-if="editedItem.type.value === 1">
                    <v-col
                      cols="12"
                    >
                      <v-text-field
                        v-model.number="editedItem.interval"
                        label="Interval"
                      />
                    </v-col>
                  </v-row>
                  <v-row v-if="editedItem.type.value === 2">
                    <v-col
                      cols="12"
                    >
                      <v-text-field
                        v-model.number="editedItem.deadband"
                        label="Deadband"
                      />
                    </v-col>
                  </v-row>
                </v-container>
              </v-card-text>

              <v-card-actions>
                <v-spacer />
                <v-btn
                  color="blue darken-1"
                  text
                  @click="close"
                >
                  Cancel
                </v-btn>
                <v-btn
                  color="blue darken-1"
                  text
                  @click="save"
                >
                  Save
                </v-btn>
              </v-card-actions>
            </v-card>
          </v-dialog>
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
            class="ml-3"
            @click="exportCSV"
          >
            Export to CSV
          </v-btn>
          <v-btn
            color="primary"
            dark
            class="ml-3"
            @click="uploadDialog = !uploadDialog"
          >
            Import from CSV
          </v-btn>
          <v-btn
            color="success"
            dark
            class="ml-3"
            :disabled="saveDisabled"
            @click="saveChanges"
          >
            Save changes
          </v-btn>
          <v-btn
            color="primary"
            dark
            class="ml-3"
            @click="refresh"
          >
            Refresh
          </v-btn>
        </v-toolbar>
      </template>
      <template
        v-slot:item.actions="{ item }"
      >
        <v-icon
          v-if="!item.new"
          class="mr-2"
          @click="editItem(item)"
        >
          mdi-pencil
        </v-icon>
        <v-icon
          v-if="!item.new"
          @click="deleteItem(item)"
        >
          mdi-delete
        </v-icon>
      </template>
    </v-data-table>
  </div>
</template>

<script>
  import Vue from 'vue'
  import ApiService from '@/services/api.service'
  import WebsocketService from '@/services/websocket.service'
  export default {
    name: 'FilteredPoints',

    data: () => ({
      dialog: false,
      dialogDelete: false,
      uploadDialog: false,
      saveDisabled: true,
      search: '',
      loading: false,
      headers: [
        { text: 'Name', value: 'datapoint.n', width: '60%' },
        { text: 'Actual Value', value: 'datapoint.v', width: '10%' },
        { text: 'Type', value: 'filtertype', width: '50px' },
        { text: 'Deadband', value: 'deadband', width: '150px' },
        { text: 'Threshold', value: 'threshold', width: '150px' },
        { text: 'Integrator', value: 'integrator', width: '150px' },
        { text: 'Previous Value', value: 'previousvalue', width: '10%' },
        { text: 'Actions', value: 'actions', width: 1, sortable: false },
      ],
      items: [],
      editedIndex: -1,
      editedItem: {
        datapoint: { n: '' },
        type: { value: 0 },
      },
      defaultItem: {
        datapoint: { n: '' },
        type: { value: 0 },
      },
      availableTypeNames: [{ text: 'Passthrough', value: 0 }, { text: 'Interval', value: 1 }, { text: 'Deadband', value: 2 }],
      sortyby: 'datapoint.n',
    }),

    created () {
      this.refresh()
      WebsocketService.topic('process.filtermeta', this, function (topic, fp, t) {
        var item = t.items.find(i => i.datapoint.n === fp.datapoint.n)
        if (item) {
          Vue.set(item, 'threshold', fp.deadband * (fp.max - fp.min).toFixed(2))
          Vue.set(item, 'integrator', fp.integrator.toFixed(2))
          Vue.set(item, 'previousvalue', fp.previousvalue.toFixed(2))
          Vue.set(item.datapoint, 'v', fp.datapoint.v.toFixed(2))
        }
      })
    },

    methods: {
      initialize () {},

      refresh () {
        this.loading = true
        var request = { subject: 'usvc.process.filter.getall', payload: { value: parseInt(this.$route.params.serverid) } }
        ApiService.post('nats/request', request)
          .then(response => {
            this.items = response.data.items
            this.items.forEach(i => { i.type = this.availableTypeNames[i.filtertype] })
          }).catch(response => {
            this.$notification.error('Failed to get tags: ' + response.data.statusmsg)
          })
      },

      editItem (item) {
        this.editedIndex = this.items.indexOf(item)
        this.editedItem = Object.assign({}, item)
        console.log('item being edited: ' + JSON.stringify(this.editedItem))
        this.dialog = true
      },

      deleteItem (item) {
        var payload = { items: [item] }
        var request = { subject: 'usvc.opc.tags.delete', payload }
        ApiService.post('nats/request', request)
          .then(response => {
            this.refresh()
            if (response.data.success) {
              this.$notification.success('Tag deleted')
            } else {
              this.$notification.error('Failed to delete tag: ' + response.data.statusmsg)
            }
          }).catch(response => {
            this.$notification.error('Failed to delete tag:' + response.message)
          })
      },

      close () {
        this.dialog = false
        this.$nextTick(() => {
          this.editedItem = Object.assign({}, this.defaultItem)
          this.editedIndex = -1
        })
      },

      save () {
        this.editedItem.filtertype = parseInt(this.editedItem.type.value)
        this.editedItem.deadband = parseFloat(this.editedItem.deadband)
        this.editedItem.min = 0
        this.editedItem.max = 5
        this.editedItem.datapoint.v = 0.0
        this.editedItem.integrator = 0.0
        this.editedItem.previousvalue = 0.0
        var request = { subject: 'usvc.process.filter.setfilter', payload: this.editedItem }
        ApiService.post('nats/request', request)
          .then(response => {
            if (response.data.success) {
              this.refresh()
              this.$notification.success('Tag saved')
            } else {
              this.$notification.error('Failed to save tag: ' + response.data.statusmsg)
            }
          }).catch(response => {
            this.$notification.error('Failed to save tag: ' + response.message)
          })
        this.close()
      },

      exportCSV () {
        let csvContent = 'data:text/csv;charset=utf-8,'
        csvContent += [
          'name;groupid;',
          ...this.items.map(item => item.name + ';' + item.groupid + ';'),
        ]
          .join('\n')
          .replace(/(^\[)|(\]$)/gm, '')

        const data = encodeURI(csvContent)
        const link = document.createElement('a')
        link.setAttribute('href', data)
        link.setAttribute('download', 'export.csv')
        link.click()
      },

      processUpload (files) {
        var reader = new FileReader()
        var t = this
        reader.onload = function (event) {
          var j = t.csvJSON(event.target.result)
          t.content = j
          t.processResponse(j)
        }
        console.log('loading file: ' + files[0].name)
        reader.readAsText(files[0])
      },

      csvJSON (csv) {
        var lines = csv.split('\n')
        var result = []

        lines.map((line, indexLine) => {
          if (indexLine < 1) return // Skip header line
          var currentline = line.split(';')
          result.push(currentline)
        })

        // result.pop() // remove the last item because undefined values
        return result // JavaScript object
      },

      processResponse (records) {
        // iterate through all existing items and compare content
        // assume the following column format:
        // col 0: tag name
        // col 1: tag group id

        for (var mi = 0; mi < records.length; mi++) {
          var record = records[mi]
          var tagname = record[0].trim()
          var groupid = parseInt(record[1])
          var found = false
          var group

          if (tagname === '') continue

          for (var g = 0; g < this.groups.length; g++) {
            if (this.groups[g].id === groupid) {
              group = this.groups[g]
              break
            }
          }

          for (var i = 0; i < this.items.length; i++) {
            var item = this.items[i]

            if (item.name !== tagname) continue
            found = true
            var same = item.groupid === groupid

            if (!same) {
              // Update group
              item.groupid = groupid
              item.group = group
              item.changed = true
              item.diff = 'changed'
            } else {
              item.changed = false
            }
            break
          }

          if (!found) {
            var newitem = { name: tagname, groupid: groupid, group: group, new: true, diff: 'new' }
            this.items.push(newitem)
          }
        }

        // keep changed items in the table
        this.items = this.items.filter(item => (item?.changed === true || item?.new === true) || false)

        if (this.items.length === 0) {
          this.$notification.error('No new or changed items identified')
          this.refresh()
        } else {
          this.saveDisabled = false
        }
      },

      saveChanges () {
        var payload = { items: this.items }
        var request = { subject: 'usvc.opc.tags.update', payload }
        ApiService.post('nats/request', request)
          .then(response => {
            if (response.data.success) {
              this.refresh()
              this.$notification.success('Changes saved')
            } else {
              this.$notification.error('Failed to save changes: ' + response.data.statusmsg)
            }
          }).catch(response => {
            this.$notification.error('Failed to save changes: ' + response.message)
          })
        this.close()
      },

      saveChangesOld () {
        var t = this
        ApiService.post('opc/tag/changes', this.items)
          .then(response => {
            t.$notification.success('Changes saved')
            t.refresh()
          }).catch(function (response) {
            t.$notification.error('Failed to save changes: ' + response)
          })
      },
    },
  }
</script>
