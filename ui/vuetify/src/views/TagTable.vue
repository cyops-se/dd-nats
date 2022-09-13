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
                <span class="text-h5">Tag</span>
              </v-card-title>

              <v-card-text>
                <v-container>
                  <v-row>
                    <v-col
                      cols="12"
                    >
                      <v-text-field
                        v-model="editedItem.name"
                        label="Name"
                        readonly
                      />
                    </v-col>
                  </v-row>
                  <v-row>
                    <v-col
                      cols="12"
                    >
                      <v-combobox
                        v-model="editedItem.group"
                        :items="availableGroups"
                        item-text="name"
                        label="Group"
                        outlined
                      />
                    </v-col>
                  </v-row>
                  <!-- <v-row>
                    <v-col
                      cols="12"
                    >
                      <v-textarea
                        v-model="editedItem.description"
                        label="Description"
                        outlined
                      />
                    </v-col>
                  </v-row> -->
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
    name: 'TagTableView',

    data: () => ({
      dialog: false,
      dialogDelete: false,
      uploadDialog: false,
      saveDisabled: true,
      search: '',
      loading: false,
      headers: [
        {
          text: 'ID',
          align: 'start',
          filterable: false,
          value: 'id',
          width: 75,
        },
        { text: 'Name', value: 'name', width: '60%' },
        { text: 'Group', value: 'group.name', width: '20%' },
        { text: 'Value', value: 'value', width: '10%' },
        { text: '', value: 'diff', width: '10%' },
        { text: 'Actions', value: 'actions', width: 1, sortable: false },
      ],
      items: [],
      editedIndex: -1,
      editedItem: {
        fullname: '',
        email: '',
      },
      defaultItem: {
        fullname: '',
        email: '',
      },
      availableGroups: [],
      groups: [],
      groupsTable: {},
    }),

    created () {
      this.refresh()
      WebsocketService.topic('system.event.process.message', this, function (topic, msg, t) {
        for (var i = 0; i < msg.points.length; i++) {
          var p = msg.points[i]
          var item = t.items.find(i => i.name === p.n)
          if (item) Vue.set(item, 'value', p.v)
        }
      })
    },

    methods: {
      initialize () {},

      refresh () {
        this.loading = true
        var request = { subject: 'usvc.opc.tags.getall', payload: { value: parseInt(this.$route.params.serverid) } }
        ApiService.post('nats/request', request)
          .then(response => {
            this.items = response.data.items
          }).catch(response => {
            this.$notification.error('Failed to get tags: ' + response.data.statusmsg)
          })
        request = { subject: 'usvc.opc.groups.getall', payload: { value: parseInt(this.$route.params.serverid) } }
        ApiService.post('nats/request', request)
          // ApiService.get('opc/tag/names')
          .then(response => {
            if (response.data.success) {
              this.groups = response.data.items
              this.availableGroups = this.groups
            } else {
              this.$notification.error('Failed to get groups: ' + response.data.statusmsg)
            }
          }).catch(response => {
            this.$notification.error('Failed to get groups: ' + response.message)
          })
      },

      editItem (item) {
        this.editedIndex = this.items.indexOf(item)
        this.editedItem = Object.assign({}, item)
        this.editedItem.groupname = item.group.name
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
        var op = this.editedIndex > -1 ? 'update' : 'add'
        var payload = { items: [this.editedItem] }
        var request = { subject: 'usvc.opc.tags.' + op, payload }
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
