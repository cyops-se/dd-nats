<template>
  <v-data-table
    :headers="headers"
    :items="items"
    class="elevation-1"
    style="cursor: pointer"
    @click:row="rowclick"
  >
    <template v-slot:top>
      <v-toolbar
        flat
      >
        <v-toolbar-title>Servers</v-toolbar-title>
        <v-divider
          class="mx-4"
          inset
          vertical
        />
        <v-spacer />
      </v-toolbar>
    </template>
    <template v-slot:item.actions="{ item }">
      <router-link
        style="text-decoration: none; color: inherit;"
        :to="{name: 'TagBrowser', params: {serverid: item.id}}"
      >
        <v-icon
          class="mr-2"
        >
          mdi-magnify
        </v-icon>
      </router-link>
    </template>
  </v-data-table>
</template>

<script>
  import ApiService from '@/services/api.service'
  export default {
    name: 'ServerTableView',

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
        { text: 'Program ID', value: 'progid', width: '100%' },
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
    }),

    created () {
      this.loading = true
      var body = { subject: 'usvc.opc.servers.getall' }
      ApiService.post('nats/request', body)
        .then(response => {
          this.items = response.data
          this.loading = false
        }).catch(response => {
          console.log('ERROR response: ' + JSON.stringify(response))
        })
    },

    methods: {
      initialize () {},
      rowclick (item) {
        console.log('row clicked: ' + JSON.stringify(item))
        this.$router.push({ name: 'TagBrowser', params: { serverid: item.id } })
      },

      editItem (item) {
        this.editedIndex = this.items.indexOf(item)
        this.editedItem = Object.assign({}, item)
        this.dialog = true
      },

      close () {
        this.dialog = false
        this.$nextTick(() => {
          this.editedItem = Object.assign({}, this.defaultItem)
          this.editedIndex = -1
        })
      },

      save () {
        var kalle = this.editedItem
        if (this.editedIndex > -1) {
          Object.assign(this.items[this.editedIndex], this.editedItem)
          ApiService.put('nats/request/usvc.opc.servers.getall', this.editedItem)
            .then(response => {
              this.$notification.success('User ' + response.data.fullname + ' successfully updated!')
            }).catch(response => {
              this.$notification.error('Failed to update user!' + response + ', ' + JSON.stringify(kalle))
            })
        } else {
          this.items.push(this.editedItem)
          ApiService.post('data/users', this.editedItem)
            .then(response => {
              this.$notification.success('User ' + response.data.fullname + ' successfully added!')
            }).catch(response => {
              this.$notification.error('Failed to add user!' + response + ', ' + JSON.stringify(kalle))
            })
        }
        this.close()
      },
    },
  }
</script>
