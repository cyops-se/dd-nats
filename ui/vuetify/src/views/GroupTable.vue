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
                    <v-combobox
                      v-model="editedItem.diodeproxy"
                      :items="availableDiodeProxies"
                      item-text="name"
                      label="Endpoint"
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
      dialog: false,
      dialogDelete: false,
      search: '',
      loading: false,
      headers: [
        {
          text: 'ID',
          align: 'start',
          filterable: false,
          value: 'ID',
          width: 75,
        },
        { text: 'Name', value: 'name', width: '20%' },
        { text: 'Description', value: 'description', width: '30%' },
        { text: 'OPC DA Server', value: 'progid', width: '10%' },
        { text: 'Diode proxy', value: 'diodeproxy.name', width: '10%' },
        { text: 'Sampling Interval (seconds)', value: 'interval', width: '10%' },
        { text: 'Default', value: 'defaultgroup', width: '5%' },
        { text: 'Actions', value: 'actions', width: 1, sortable: false },
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
      this.loading = true
      this.editedItem = Object.assign({}, this.defaultItem)
      this.editedIndex = -1
      ApiService.get('data/opc_groups')
        .then(response => {
          this.items = response.data
          this.loading = false
        }).catch(response => {
          console.log('ERROR response: ' + JSON.stringify(response))
        })

      ApiService.get('opc/server')
        .then(response => {
          for (var i = 0; i < response.data.length; i++) {
            this.availableProgids.push(response.data[i].progid)
          }
          console.log('available progids: ' + this.availableProgids)
        }).catch(response => {
          console.log('ERROR response: ' + JSON.stringify(response))
        })

      ApiService.get('data/diode_proxies')
        .then(response => {
          this.availableDiodeProxies = response.data
          console.log('available diode proxies: ' + JSON.stringify(this.availableDiodeProxies))
        }).catch(response => {
          console.log('ERROR response: ' + JSON.stringify(response))
        })
    },

    methods: {
      initialize () {},

      editItem (item) {
        this.editedIndex = this.items.indexOf(item)
        this.editedItem = Object.assign({}, item)
        this.dialog = true
      },

      deleteItem (item) {
        ApiService.delete('data/opc_groups/' + item.ID)
          .then(response => {
            for (var i = 0; i < this.items.length; i++) {
              if (this.items[i].ID === item.ID) this.items.splice(i, 1)
            }
            this.$notification.success('Group deleted')
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
          // Object.assign(this.items[this.editedIndex], this.editedItem)
          ApiService.put('opc/group', this.editedItem)
            .then(response => {
              this.$notification.success('Group updated!')
              this.items = response.data
            }).catch(function (response) {
              console.log('Failed to update group! ' + response)
              this.$notification.error('Failed to update group!' + response)
            })
        } else {
          ApiService.post('opc/group', this.editedItem)
            .then(response => {
              this.$notification.success('Group created!')
              // this.items.push(response.data)
              this.items = response.data
            }).catch(function (response) {
              console.log('Failed to create group! ' + response.message)
              this.$notification.error('Failed to create group!' + response)
            })
        }
        this.editedItem = Object.assign({}, this.defaultItem)
        this.editedIndex = -1
        this.close()
      },
    },
  }
</script>
