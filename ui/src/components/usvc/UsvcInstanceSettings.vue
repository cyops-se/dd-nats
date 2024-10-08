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
        <v-toolbar-title>Settings</v-toolbar-title>
        <v-divider
          class="mx-4"
          inset
          vertical
        />
        <v-spacer />
        <v-dialog
          v-model="dialog"
          max-width="600px"
        >
          <v-card>
            <v-card-title>
              <span class="text-h5">Setting</span>
            </v-card-title>

            <v-card-text>
              <v-container>
                <v-row>
                  <v-col
                    cols="12"
                  >
                    <span class="text-h4">{{ editedItem.key }}</span>
                    <v-divider
                      class="my-3"
                    />
                    {{ editedItem.extra }}
                    <v-divider
                      class="mt-3"
                    />
                  </v-col>
                  <v-col
                    cols="12"
                  >
                    <v-text-field
                      v-model="editedItem.value"
                      label="Value"
                      outlined
                      hide-details
                      class="mb-0"
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
        <v-spacer />
        <instance-selector
          :svcname="usvc"
          @change="refresh"
        />
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
        class="mr-2"
        @click="deleteItem(item)"
      >
        mdi-delete
      </v-icon>
    </template>
  </v-data-table>
</template>

<script>
  import { sync } from 'vuex-pathify'
  import ApiService from '@/services/api.service'
  export default {
    name: 'UsvcInstanceSettings',

    props: {
      usvc: String,
    },

    data: () => ({
      dialog: false,
      dialogDelete: false,
      search: '',
      loading: false,
      headers: [
        { text: 'Key', value: 'key', width: '20%' },
        { text: 'Value', value: 'value', width: '20%' },
        { text: 'Description', value: 'extra', width: '60%' },
        { text: 'Actions', value: 'actions', width: '100px', sortable: false },
      ],
      items: [],
      editedIndex: -1,
      editedItem: {},
    }),

    computed: {
      ...sync('context', [
        'selected',
      ]),
    },

    created () {
    },

    mounted () {
      this.refresh()
    },

    methods: {
      initialize () {},

      refresh () {
        if (this.selected === '') return
        var request = { subject: 'usvc.' + this.usvc + '.' + this.selected.key + '.settings.get', payload: {} }
        ApiService.post('nats/request', request)
          .then(response => {
            this.items = []
            const keys = Object.keys(response.data.items)
            keys.forEach(key => {
              var value = response.data.items[key]
              this.items.push({ key: key, value: value })
            })
          }).catch(response => {
            console.log(response)
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Failed to get settings: ' + response.message)
          })
      },

      editItem (item) {
        this.editedIndex = this.items.indexOf(item)
        this.editedItem = Object.assign({}, item)
        this.dialog = true
      },

      deleteItem (item) {
        var request = { subject: 'usvc.' + this.usvc + '.' + this.selected.key + '.settings.delete', payload: { item: item.key } }
        ApiService.post('nats/request', request)
          .then(response => {
            if (response.data.success) {
              this.$notification.success('Setting deleted')
              this.refresh()
            } else {
              console.log('ERROR response: ' + response.data.statusmsg)
              this.$notification.error('Failed delete setting: ' + response.data.statusmsg)
            }
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Failed to delete setting: ' + response.message)
          })
      },

      close () {
        this.dialog = false
        this.$nextTick(() => {
          this.editedItem = {}
          this.editedIndex = -1
        })
      },

      save () {
        this.items[this.editedIndex] = this.editedItem
        var items = {}
        // eslint-disable-next-line no-return-assign
        this.items.forEach(o => items[o.key] = o.value)
        var request = { subject: 'usvc.' + this.usvc + '.' + this.selected.key + '.settings.set', payload: { items: items } }
        ApiService.post('nats/request', request)
          .then(response => {
            if (response.data.success) {
              this.$notification.success('Settings saved')
              this.refresh()
            } else {
              console.log('ERROR response: ' + response.data.statusmsg)
              this.$notification.error('Failed save settings: ' + response.data.statusmsg)
            }
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Failed to get settings: ' + response.message)
          })

        this.editedItem = {}
        this.editedIndex = -1
        this.close()
      },
    },
  }
</script>
