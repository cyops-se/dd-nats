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
          <v-toolbar-title>Meta data</v-toolbar-title>
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
                <span class="text-h5">Meta item</span>
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
                      cols="6"
                    >
                      <v-text-field
                        v-model.number="editedItem.min"
                        label="Min"
                      />
                    </v-col>
                    <v-col
                      cols="6"
                    >
                      <v-text-field
                        v-model.number="editedItem.max"
                        label="Max"
                      />
                    </v-col>
                  </v-row>
                  <v-row>
                    <v-col
                      cols="12"
                    >
                      <v-textarea
                        v-model="editedItem.description"
                        label="Description"
                        outlined
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
            Export
          </v-btn>
          <v-btn
            color="primary"
            dark
            class="ml-3"
            @click="uploadDialog = !uploadDialog"
          >
            Import
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
      <template v-slot:item.actions="{ item }">
        <v-icon
          class="mr-2"
          @click="editItem(item)"
        >
          mdi-pencil
        </v-icon>
        <v-icon
          @click="deleteItem(item)"
        >
          mdi-delete
        </v-icon>
      </template>
    </v-data-table>
  </div>
</template>

<script>
  import ApiService from '@/services/api.service'
  export default {
    name: 'Meta',

    data: () => ({
      dialog: false,
      dialogDelete: false,
      uploadDialog: false,
      saveDisabled: true,
      search: '',
      headers: [
        { text: 'Name', value: 'name', width: '40%' },
        { text: 'Description', value: 'description', width: '50%' },
        { text: 'Unit', value: 'engunit', width: '90px' },
        { text: 'Min', value: 'min', width: '90px' },
        { text: 'Max', value: 'max', width: '90px' },
        { text: '', value: 'changed', width: '90px' },
        { text: 'Actions', value: 'actions', width: 1, sortable: false },
      ],
      items: [],
      editedIndex: -1,
      editedItem: {
        tag: '',
      },
      defaultItem: {
        tag: '',
      },
      selected: [],
    }),

    created () {
      this.refresh()
    },

    methods: {
      initialize () {},

      refresh () {
        var request = { subject: 'usvc.timescale.meta.getall', payload: {} }
        ApiService.post('nats/request', request)
          .then((response) => {
            if (response.data.success) {
              var items = response.data.items
              this.items = items
            } else {
              console.log('ERROR response: ' + JSON.stringify(response.data))
            }
          })
          .catch((response) => {
            console.log('ERROR response: ' + JSON.stringify(response))
          })
      },

      editItem (item) {
        console.log('item: ' + JSON.stringify(item))
        this.editedIndex = this.items.indexOf(item)
        this.editedItem = Object.assign({}, item)
        this.dialog = true
      },

      deleteItem (item) {
        var request = { subject: 'usvc.timescale.meta.delete', payload: { items: [item] } }
        ApiService.post('nats/request', request)
          .then((response) => {
            if (response.data.success) {
              this.refresh()
              this.$notification.success('Meta data deleted!')
            } else {
              console.log('ERROR response: ' + JSON.stringify(response.data))
            }
          })
          .catch((response) => {
            console.log('ERROR response: ' + JSON.stringify(response))
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
        Object.assign(this.items[this.editedIndex], this.editedItem)
        var request = { subject: 'usvc.timescale.meta.updateall', payload: { items: [this.editedItem] } }
        ApiService.post('nats/request', request)
          .then(response => {
            if (response.data.success) {
              this.$notification.success('Meta data updated!')
            } else {
              this.$notification.error('Failed to update meta data: ' + response.data.statusmsg)
            }
          }).catch(response => {
            this.$notification.error('Failed to update meta data: ' + response)
          })
        this.close()
      },

      exportCSV () {
        let csvContent = 'data:text/csv;charset=utf-8,'
        csvContent += [
          'inuse;name;description;min;max;unit;',
          ...this.items.map(item => 'x;' + item.name + ';' + item.description + ';' + item.min + ';' + item.max + ';' + item.engunit + ';'),
        ]
          .join('\n')
          .replace(/(^\[)|(\]$)/gm, '')

        const data = encodeURI(csvContent)
        const link = document.createElement('a')
        link.setAttribute('href', data)
        link.setAttribute('download', 'meta.csv')
        link.click()
      },

      processUpload (files) {
        var reader = new FileReader()
        var t = this
        reader.onload = function (event) {
          // console.log('file content loaded: ' + event.target.result)
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
        // col 0: x indicates in use
        // col 1: tag name
        // col 2: tag description
        // col 3: tag min value
        // col 4: tag max value
        // col 5: tag unit

        for (var mi = 0; mi < records.length; mi++) {
          var record = records[mi]
          var inuse = record[0]
          var tagname = record[1]
          var description = record[2]
          var min = parseFloat(record[3])
          var max = parseFloat(record[4])
          var unit = record[5]

          if (inuse !== 'x') continue

          for (var i = 0; i < this.items.length; i++) {
            var item = this.items[i]

            if (item.name.indexOf(tagname) === -1) continue
            var same = item.description === description
            if (same) same = item.engunit === unit
            if (same) same = item.min === min
            if (same) same = item.max === max

            if (!same) {
              item.description = description
              item.engunit = unit
              item.min = min
              item.max = max
              item.changed = true
            } else {
              item.changed = false
            }
            break
          }
        }

        // keep changed items in the table
        this.items = this.items.filter(item => item?.changed === true || false)

        this.saveDisabled = this.items.length === 0
      },

      processResponseModbus (records) {
        // iterate through all existing items and compare content
        // assume the following column format:
        // col 0: tag name
        // col 1: tag description
        // col 2: signal type (not relevant)
        // col 3: ip address (of modbus slave (server))
        // col 4: data type (U-Int)
        // col 5: data length (16-bit)
        // col 6: engineering unit (m3/h)
        // col 7: byte order (AB CD)
        // col 8: function code
        // col 9: modbus address (40107)
        // col 10: actual range (0-60, string where min and max are separated by a -)
        // col 11: PLC range (3965-20000, string where min and max are separated by a -)

        for (var mi = 0; mi < records.length; mi++) {
          var record = records[mi]
          // Required fields are tagname [0], ipaddress [3]
          if (typeof record[0] !== 'string' || typeof record[3] !== 'string' || record[0] === '' || record[0] === 'Värde' || record[3] === '') {
            console.log('skipping record: ' + JSON.stringify(record))
            continue
          }

          var tagname = record[0].trim()
          var description = record[1].trim()
          var ipaddress = record[3].trim()
          var datatype = record[4].trim()
          var datalengthstr = record[5].trim()
          var engunit = record[6].trim()
          var byteorder = record[7].trim()
          var functioncode = parseInt(record[8].trim())
          var modbusaddress = parseInt(record[9].trim())
          var rangestr = record[10].trim()
          var plcrangestr = record[11].trim()
          var found = false
          var slave

          // for (var g = 0; g < this.slaves.length; g++) {
          //   if (this.slaves[g].ID === slaveid) {
          //     slave = this.slaves[g]
          //     break
          //   }
          // }

          var ranges = rangestr.split('-')
          var plcranges = plcrangestr.split('-')
          var datalength = parseInt(datalengthstr.split('-')[0])

          for (var i = 0; i < this.items.length; i++) {
            var item = this.items[i]

            if (item.name !== tagname) continue
            found = true
            var same = item.modbusaddress === modbusaddress
            if (same) same = item.description === description

            if (!same) {
              // Update slave
              // item.slaveid = slaveid
              item.slave = slave
              item.changed = true
              item.diff = 'changed'
              item.description = description
            } else {
              item.changed = false
            }
            break
          }

          console.log('tag: ' + tagname + ', found: ' + found)
          if (!found) {
            console.log('NEW tag: ' + tagname + ', found: ' + found)
            var newitem = { name: tagname, description: description, modbusaddress: modbusaddress, engunit: engunit, functioncode: functioncode, new: true, diff: 'new' }
            newitem.datatype = datatype
            newitem.datalength = datalength
            newitem.byteorder = byteorder
            newitem.rangemin = parseInt(ranges[0])
            newitem.rangemax = parseInt(ranges[1])
            newitem.plcrangemin = parseInt(plcranges[0])
            newitem.plcrangemax = parseInt(plcranges[1])
            newitem.ipaddress = ipaddress
            this.items.push(newitem)
          }
        }

        // keep changed items in the table
        this.items = this.items.filter(item => (item?.changed === true || item?.new === true) || false)

        if (this.items.length === 0) {
          this.$notification.error('No new or changed items identified')
          this.refresh()
          this.saveDisabled = true
        } else {
          this.saveDisabled = false
        }
      },

      saveChanges () {
        // console.log('bulk changing items: ' + JSON.stringify(this.items))
        var request = { subject: 'usvc.timescale.meta.updateall', payload: { items: this.items } }
        ApiService.post('nats/request', request)
          .then((response) => {
            if (response.data.success) {
              this.$notification.success('Meta data change successful')
              this.refresh()
              this.saveDisabled = true
            } else {
              this.$notification.error('Meta data change failed: ' + response.data.statusmsg)
            }
          })
          .catch((response) => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Modbus bulk data change failed: ' + response.message)
          })
        this.close()

        // var t = this
        // console.log('posting: ' + JSON.stringify(this.items))
        // ApiService.post('modbus/tag/changes', this.items)
        //   .then(response => {
        //     t.$notification.success('Changes saved')
        //     t.update()
        //   }).catch(function (response) {
        //     t.$notification.error('Failed to save changes: ' + response)
        //   })
      },
    },
  }
</script>
