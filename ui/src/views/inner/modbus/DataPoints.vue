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
        <v-toolbar flat>
          <v-toolbar-title>Data points</v-toolbar-title>
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
                <span class="text-h5">Data point</span>
              </v-card-title>

              <v-card-text>
                <v-container>
                  <v-row>
                    <v-col cols="12">
                      <v-text-field
                        v-model="editedItem.name"
                        label="Name"
                        readonly
                      />
                    </v-col>
                  </v-row>
                  <v-row>
                    <v-col cols="12">
                      <v-combobox
                        v-model="editedItem.slaveid"
                        :items="availableSlaves"
                        item-text="name"
                        label="slave"
                        outlined
                      />
                    </v-col>
                  </v-row>
                  <v-row>
                    <v-col cols="12">
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
          <!-- v-btn
            color="primary"
            dark
            class="ml-3"
            @click="exportCSV"
          >
            Export to CSV
          </v-btn -->
          <v-btn
            color="primary"
            dark
            class="ml-3"
            @click="uploadDialog = !uploadDialog"
          >
            Import from CSV
          </v-btn>
          <v-btn
            color="primary"
            dark
            class="ml-3"
            @click="exportCSV"
          >
            Export to CSV
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
        <!-- v-icon
          class="mr-2"
          @click="editItem(item)"
        >
          mdi-pencil
        </v-icon -->
        <v-icon @click="deleteItem(item)">
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
    name: 'DataPoints',

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
        { text: 'Name', value: 'name', width: '20%' },
        { text: 'Description', value: 'description', width: '30%' },
        { text: 'Modbus Slave', value: 'ipaddress', width: '90px' },
        { text: 'Value', value: 'value', width: '90px' },
        { text: 'Register', value: 'modbusaddress', width: '90px' },
        { text: 'FC', value: 'functioncode', width: '90px' },
        { text: '', value: 'diff', width: '60px' },
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
      availableSlaves: [],
      slaves: [],
      slavesTable: {},
    }),

    created () {
      this.refresh()
      WebsocketService.topic('process.actual', this, function (topic, p, t) {
        var item = t.items.find(i => i.name === p.n)
        if (item) Vue.set(item, 'value', p.v.toFixed(2))
      })
    },

    methods: {
      initialize () { },

      refresh () {
        this.loading = true

        var request = { subject: 'usvc.modbus.slaves.getall', payload: {} }
        ApiService.post('nats/request', request)
          .then((response) => {
            var items = response.data.items
            this.slaves = items
            this.loading = false
          })
          .catch((e) => {
            console.log('ERROR response: ' + JSON.stringify(e.message))
          })

        request = { subject: 'usvc.modbus.items.getall', payload: {} }
        ApiService.post('nats/request', request)
          .then((response) => {
            var items = response.data.items
            this.items = items
            for (var i = 0; i < this.items.length; i++) {
              for (var s = 0; s < this.slaves.length; s++) {
                if (this.items[i].modbusslaveid === this.slaves[s].id) {
                  this.items[i].modbusslave = this.slaves[s]
                  this.items[i].ipaddress = this.slaves[s].ip
                }
              }
            }
            console.log(JSON.stringify(this.items))
            this.loading = false
          })
          .catch((e) => {
            console.log('ERROR response: ' + JSON.stringify(e.message))
          })

        this.loading = false
      },

      editItem (item) {
        this.editedIndex = this.items.indexOf(item)
        this.editedItem = Object.assign({}, item)
        this.editedItem.slavename = item.modbusslave.name
        this.dialog = true
      },

      deleteItem (item) {
        var request = { subject: 'usvc.modbus.items.delete', payload: { items: [item] } }
        ApiService.post('nats/request', request)
          .then(response => {
            for (var i = 0; i < this.items.length; i++) {
              if (this.items[i].id === item.id) this.items.splice(i, 1)
            }
            this.$notification.success('Tag deleted')
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
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
        if (this.editedIndex > -1) {
          Object.assign(this.items[this.editedIndex], this.editedItem)
          ApiService.put('data/modbus_slaves', this.editedItem)
            .then(response => {
            }).catch(response => {
              this.$notification.error('Failed to update tag!' + response)
            })
        } else {
          this.items.push(this.editedItem)
          ApiService.post('data/modbus_slaves', this.editedItem)
            .then(response => {
            }).catch(response => {
              this.failureMessage('Failed to add tag!' + response)
            })
        }
        this.close()
      },

      exportCSV () {
        /*
        var tagname = record[0].trim()
        var description = record[1].trim()
        // signaltype not saved
        var ipaddress = record[3].trim()
        var datatype = record[4].trim()
        var datalengthstr = record[5].trim()
        var engunit = record[6].trim()
        var byteorder = record[7].trim()
        var functioncode = parseInt(record[8].trim())
        var modbusaddress = parseInt(record[9].trim())
        var rangestr = record[10].trim()
        var plcrangestr = record[11].trim()
      */

        let csvContent = 'data:text/csv;charset=utf-8,'
        csvContent += [
          'tagname;description;signaltype;ipaddress;datatype;datalength;engunit;byteorder;functioncode;modbusaddress;range;rangeplc;',
          ...this.items.map(item => item.name + ';' + item.description + ';;' + item.modbusslave.ip + ';' + item.datalength + ';' +
            item.datatype + ';' + item.engunit + ';' + item.byteorder + ';' + item.functioncode + ';' +
            item.modbusaddress + ';' + item.rangemin + '-' + item.rangemax + ';' + item.plcrangemin + '-' + item.plcrangemax + ';'),
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
          if (typeof record[0] !== 'string' || typeof record[3] !== 'string' || record[0] === '' || record[3] === '') {
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

          if (!found) {
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
        var request = { subject: 'usvc.modbus.items.bulkchanges', payload: { items: this.items } }
        ApiService.post('nats/request', request)
          .then((response) => {
            if (response.data.success) {
              this.$notification.success('Modbus data import successful')
              this.refresh()
              this.saveDisabled = true
            } else {
              this.$notification.error('Modbus data import failed: ' + response.data.statusmsg)
            }
          })
          .catch((response) => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Modbus data import failed: ' + response.message)
          })
        this.close()
      },
    },
  }
</script>
