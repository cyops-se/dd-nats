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
        <v-toolbar-title>Diode endpoints</v-toolbar-title>
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
              New End-point
            </v-btn>
          </template>
          <v-card>
            <v-card-title>
              <span class="text-h5">Group</span>
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
                    <v-textarea
                      v-model="editedItem.description"
                      label="Description"
                      outlined
                    />
                  </v-col>
                  <v-col cols="12">
                    <v-text-field
                      v-model="editedItem.ip"
                      label="Endpoint IP (receiver on other side of diode)"
                      outlined
                      hide-details
                    />
                  </v-col>
                  <v-col cols="4">
                    <v-text-field
                      v-model.number="editedItem.metaport"
                      label="Meta data port"
                      outlined
                      hide-details
                    />
                  </v-col>
                  <v-col cols="4">
                    <v-text-field
                      v-model.number="editedItem.dataport"
                      label="Process data port"
                      outlined
                      hide-details
                    />
                  </v-col>
                  <v-col cols="4">
                    <v-text-field
                      v-model.number="editedItem.fileport"
                      label="File transfer port"
                      outlined
                      hide-details
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
  </v-data-table>
</template>

<script>
  import ApiService from '@/services/api.service'

  export default {
    name: 'DiodeEndpoints',

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
          value: 'id',
          width: 75,
        },
        { text: 'Name', value: 'name', width: '20%' },
        { text: 'Description', value: 'description', width: '30%' },
        { text: 'Enpoint IP', value: 'ip', width: '10%' },
        { text: 'Meta port', value: 'metaport', width: '10%' },
        { text: 'Data port', value: 'dataport', width: '10%' },
        { text: 'File port', value: 'fileport', width: '10%' },
        { text: 'Actions', value: 'actions', width: 1, sortable: false },
      ],
      items: [],
      editedIndex: -1,
      editedItem: {},
      defaultItem: {
        name: '',
        endpointip: '192.168.0.174',
        metaport: 4356,
        dataport: 4357,
        fileport: 4358,
      },
    }),

    created () {
      this.loading = true
      this.editedItem = Object.assign({}, this.defaultItem)
      this.editedIndex = -1
      ApiService.get('data/diode_proxies')
        .then(response => {
          this.items = response.data
          this.loading = false
        }).catch(e => {
          console.log('ERROR response: ' + JSON.stringify(e.message))
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
        ApiService.delete('data/diode_proxies/' + item.ID)
          .then(response => {
            for (var i = 0; i < this.items.length; i++) {
              if (this.items[i].ID === item.ID) this.items.splice(i, 1)
            }
            this.$notification.success('Diode proxy deleted')
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
        console.log('edit item' + JSON.stringify(this.editedItem))
        if (this.editedIndex > -1) {
          Object.assign(this.items[this.editedIndex], this.editedItem)
          ApiService.put('data/diode_proxies', this.editedItem)
            .then(response => {
              this.$notification.success('Diode proxy updated!')
            }).catch(function (response) {
              console.log('Failed to update diode proxy! ' + response)
              this.$notification.error('Failed to update diode proxy!' + response)
            })
        } else {
          ApiService.post('data/diode_proxies', this.editedItem)
            .then(response => {
              this.$notification.success('Diode proxy created!')
              this.items.push(response.data)
            }).catch(function (response) {
              console.log('Failed to create diode proxy! ' + response.message)
              this.$notification.error('Failed to create diode proxy!' + response)
            })
        }
        this.editedItem = Object.assign({}, this.defaultItem)
        this.editedIndex = -1
        this.close()
      },
    },
  }
</script>
