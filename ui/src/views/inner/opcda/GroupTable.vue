<template>
  <v-data-table
    :headers="headers"
    :items="items"
    class="elevation-1"
  >
    <template v-slot:top>
      <v-toolbar
        flat
      >
        <v-toolbar-title>Groups</v-toolbar-title>
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
          <template v-slot:activator="{ on, attrs }">
            <v-btn
              color="primary"
              dark
              class="mb-2"
              v-bind="attrs"
              v-on="on"
            >
              New Group
            </v-btn>
          </template>
          <v-card>
            <v-card-title>
              <span class="text-h5">Group</span>
            </v-card-title>

            <v-card-text>
              <v-container>
                <v-row>
                  <v-col cols="6">
                    <v-text-field
                      v-model="editedItem.name"
                      label="Name"
                      outlined
                      hide-details
                    />
                  </v-col>
                  <v-col cols="6">
                    <v-text-field
                      v-model.number="editedItem.interval"
                      label="Sampling Interval"
                      type="number"
                      outlined
                      hide-details
                    />
                  </v-col>
                  <v-col cols="12">
                    <v-combobox
                      v-model="editedItem.progid"
                      :items="availableProgids"
                      label="Server ProgID"
                      outlined
                      hide-details
                    />
                  </v-col>
                  <v-col cols="12">
                    <v-checkbox
                      v-model="editedItem.runatstart"
                      label="Start automatically"
                      hide-details
                      class="mt-n3"
                      :value="editedItem ? editedItem.runatstart : true"
                    />
                  </v-col>
                  <v-col cols="12">
                    <v-checkbox
                      v-model="editedItem.defaultgroup"
                      label="Default group"
                      hide-details
                      class="mt-n3"
                      :value="editedItem ? editedItem.defaultgroup : true"
                    />
                  </v-col>
                  <!-- <v-col cols="12">
                    <v-textarea
                      v-model="editedItem.description"
                      label="Description"
                      outlined
                    />
                  </v-col> -->
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
      <v-icon
        @click="deleteItem(item)"
      >
        mdi-delete
      </v-icon>
      <v-icon
        v-if="item.state<2"
        @click="startItem(item)"
      >
        mdi-play
      </v-icon>
      <v-icon
        v-if="item.state>1"
        @click="stopItem(item)"
      >
        mdi-stop
      </v-icon>
    </template>
    <template v-slot:item.defaultgroup="{ item }">
      <div>{{ item.defaultgroup ? "Yes": "" }}</div>
    </template>
  </v-data-table>
</template>

<script>
  import ApiService from '@/services/api.service'

  export default {
    name: 'GroupTableView',

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
        { text: 'Name', value: 'name', width: '40%' },
        { text: 'OPC DA Server', value: 'progid', width: '40%' },
        { text: 'State', value: 'statemsg', width: '10%' },
        { text: 'Sampling Interval (seconds)', value: 'interval', width: '10%' },
        { text: 'Default', value: 'defaultgroup', width: '5%' },
        { text: 'Actions', value: 'actions', width: 130, sortable: false },
      ],
      items: [],
      availableProgids: [],
      availableDiodeProxies: [],
      editedIndex: -1,
      editedItem: {},
      defaultItem: {
        runatstart: true,
      },
    }),

    created () {
      this.refresh()
    },

    methods: {
      initialize () {},

      refresh () {
        this.loading = true
        this.editedItem = Object.assign({}, this.defaultItem)
        this.editedIndex = -1
        var request = { subject: 'usvc.opc.groups.getall', payload: { value: parseInt(this.$route.params.serverid) } }
        ApiService.post('nats/request', request)
          // ApiService.get('opc/tag/names')
          .then(response => {
            this.items = response.data.items
            for (var i = 0; i < this.items.length; i++) {
              this.items[i].statemsg = this.states[this.items[i].state]
            }
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Failed to get groups: ' + response.message)
          })

        request = { subject: 'usvc.opc.servers.getall' }
        ApiService.post('nats/request', request)
          .then(response => {
            for (var i = 0; i < response.data.length; i++) {
              this.availableProgids.push(response.data[i].progid)
            }
            this.loading = false
          }).catch(response => {
            console.log('ERROR response: ' + JSON.stringify(response))
            this.$notification.error('Failed to get servers: ' + response.message)
          })
      },

      startItem (item) {
        var payload = { value: parseInt(item.id) }
        var request = { subject: 'usvc.opc.groups.start', payload }
        ApiService.post('nats/request', request)
          .then(response => {
            if (response.data.success) {
              this.$notification.success('Group started')
              this.refresh()
            } else {
              this.$notification.error('Failed to start group: ' + response.data.statusmsg)
            }
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Failed to start group: ' + response.message)
          })
      },

      stopItem (item) {
        var payload = { value: parseInt(item.id) }
        var request = { subject: 'usvc.opc.groups.stop', payload }
        ApiService.post('nats/request', request)
          .then(response => {
            if (response.data.success) {
              this.$notification.success('Group started')
              this.refresh()
            } else {
              this.$notification.error('Failed to stop group: ' + response.data.statusmsg)
            }
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Failed to stop group: ' + response.message)
          })
      },

      editItem (item) {
        this.editedIndex = this.items.indexOf(item)
        this.editedItem = Object.assign({}, item)
        this.dialog = true
      },

      deleteItem (item) {
        var payload = { items: [item] }
        var request = { subject: 'usvc.opc.groups.delete', payload }
        ApiService.post('nats/request', request)
          .then(response => {
            if (response.data.success) {
              for (var i = 0; i < this.items.length; i++) {
                if (this.items[i].id === item.id) this.items.splice(i, 1)
              }
              this.$notification.success('Group deleted')
            } else {
              this.$notification.error('Failed to delete group: ' + response.data.statusmsg)
            }
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Failed to delete group: ' + response.message)
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
        var request = { subject: 'usvc.opc.groups.' + op, payload }
        ApiService.post('nats/request', request)
          .then(response => {
            if (response.data.success) {
              this.refresh()
              console.log('Group added')
            } else {
              console.log('Failed to add group: ' + response.data.statusmsg)
            }
            this.close()
          }).catch(response => {
            this.$notification.error('Failed to add group: ' + response.message)
          })
      },
    },
  }
</script>
