<template>
  <v-data-table
    :headers="headers"
    :items="items"
    class="elevation-1"
  >
    <template v-slot:top>
      <v-toolbar flat>
        <v-toolbar-title>Modbus Slaves</v-toolbar-title>
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
          <!-- template v-slot:activator="{ on, attrs }">
            <v-btn
              color="primary"
              dark
              class="mb-2"
              v-bind="attrs"
              v-on="on"
            >
              New Modbus slave
            </v-btn>
          </template -->
          <v-card>
            <v-card-title>
              <span class="text-h5">Modbus slave</span>
            </v-card-title>

            <v-card-text>
              <v-container>
                <v-row>
                  <v-col cols="12">
                    <v-text-field
                      v-model="editedItem.name"
                      label="Name"
                      outlined
                      hide-details
                    />
                  </v-col>
                  <v-col cols="12">
                    <v-text-field
                      v-model="editedItem.ip"
                      label="Modbus slave IP"
                      outlined
                      hide-details
                    />
                  </v-col>
                  <v-col cols="12">
                    <v-text-field
                      v-model.number="editedItem.offset"
                      label="Register address offset"
                      outlined
                      hide-details
                      type="number"
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
      </v-toolbar>
    </template>
    <template v-slot:item.actions="{ item }">
      <v-icon
        class="mr-2"
        @click="editItem(item)"
      >
        mdi-pencil
      </v-icon>
      <!-- v-icon @click="deleteItem(item)">
        mdi-delete
      </v-icon -->
    </template>
  </v-data-table>
</template>

<script>
  import ApiService from '@/services/api.service'
  export default {
    name: 'ModbusSlaves',

    data: () => ({
      states: ['unknown', 'stopped', 'running', 'warning'],
      dialog: false,
      dialogDelete: false,
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
        { text: 'Name', value: 'name', width: '110px' },
        { text: 'IP Address', value: 'ip', width: '150px' },
        { text: 'State', value: 'statemsg', width: '100px' },
        { text: 'Error', value: 'errormsg', width: '40%' },
        { text: 'Last Error', value: 'lasterror', width: '200px' },
        { text: 'Last Success', value: 'lastrun', width: '200px' },
        { text: 'Actions', value: 'actions', width: 1, sortable: false },
      ],
      items: [],
      availableDiodeProxies: [],
      connections: [],
      editedIndex: -1,
      editedItem: {},
      defaultItem: {
        name: '',
        ipaddress: '',
      },
    }),

    created () {
      this.refresh()
    },

    methods: {
      initialize () {},

      refresh () {
        this.loading = true
        var request = { subject: 'usvc.modbus.slaves.getall', payload: {} }
        ApiService.post('nats/request', request)
          .then((response) => {
            var items = response.data.items
            for (var i = 0; i < items.length; i++) {
              items[i].statemsg = this.states[items[i].state]
            }

            this.items = items
            this.loading = false
          })
          .catch((e) => {
            console.log('ERROR response: ' + JSON.stringify(e.message))
          })
      },

      editItem (item) {
        this.editedIndex = this.items.indexOf(item)
        this.editedItem = Object.assign({}, item)
        this.dialog = true
      },

      deleteItem (item) {
        var payload = { items: [item] }
        var request = { subject: 'usvc.modbus.slaves.delete', payload: payload }
        ApiService.post('nats/request', request)
          .then((response) => {
            if (response.data.success) {
              for (var i = 0; i < this.items.length; i++) {
                if (this.items[i].id === item.id) this.items.splice(i, 1)
              }
              this.$notification.success('Modbus slave deleted')
            } else {
              this.$notification.error('Failed to delete Modbus slave: ' + response.message)
            }
          })
          .catch((e) => {
            console.log('ERROR response: ' + e.message)
            this.$notification.error('Failed to delete Modbus slave: ' + e.message)
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
        var op = this.editedItem.id ? 'update' : 'add'
        var request = { subject: 'usvc.modbus.slaves.' + op, payload: { items: [this.editedItem] } }
        ApiService.post('nats/request', request)
          .then((response) => {
            if (response.data.success) {
              this.$notification.success('Modbus slave added/updated')
              this.refresh()
            } else {
              this.$notification.error('Failed to add/update Modbus slave: ' + response.data.statusmsg)
            }
          })
          .catch((response) => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Failed to add/update Modbus slave: ' + response.message)
          })
        this.close()
      },
    },
  }
</script>
