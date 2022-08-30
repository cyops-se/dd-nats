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
      </v-toolbar>
    </template>
    <template v-slot:item.actions="{ item }">
      <v-icon
        class="mr-2"
        @click="editItem(item)"
      >
        mdi-pencil
      </v-icon>
    </template>
  </v-data-table>
</template>

<script>
  import ApiService from '@/services/api.service'
  export default {
    name: 'SystemSettingsView',

    data: () => ({
      dialog: false,
      dialogDelete: false,
      search: '',
      loading: false,
      headers: [
        { text: 'Key', value: 'key', width: '20%' },
        { text: 'Value', value: 'value', width: '20%' },
        { text: 'Description', value: 'extra', width: '60%' },
        { text: 'Actions', value: 'actions', width: 1, sortable: false },
      ],
      items: [],
      editedIndex: -1,
      editedItem: {},
    }),

    created () {
      this.loading = true
      ApiService.get('data/key_value_pairs')
        .then(response => {
          this.items = response.data
          this.loading = false
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
        ApiService.delete('data/key_value_pairs/' + item.ID)
          .then(response => {
            for (var i = 0; i < this.items.length; i++) {
              if (this.items[i].ID === item.ID) this.items.splice(i, 1)
            }
            this.$notification.success('Setting deleted')
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
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
        if (this.editedIndex > -1) {
          console.log('edited item: ' + JSON.stringify(this.editedItem))
          Object.assign(this.items[this.editedIndex], this.editedItem)
          ApiService.put('data/key_value_pairs', this.editedItem)
            .then(response => {
              this.$notification.success('Setting ' + response.data.key + ' successfully updated!')
            }).catch(response => {
              this.$notification.error('Failed to update setting!' + response.message)
            })
        } else {
          this.items.push(this.editedItem)
          ApiService.post('data/key_value_pairs', this.editedItem)
            .then(response => {
              this.$notification.success('Setting ' + response.data.key + ' successfully added!')
            }).catch(response => {
              this.$notification.error('Failed to add setting!' + response)
            })
        }
        this.editedItem = {}
        this.editedIndex = -1
        this.close()
      },
    },
  }
</script>
