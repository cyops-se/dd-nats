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
        <instance-selector
          svcname="ddnatsopcda"
        />
      </v-toolbar>
    </template>
    <template v-slot:item.actions="{ item }">
      <router-link
        style="text-decoration: none; color: inherit;"
        :to="{name: 'inner/opcda/TagBrowser', params: {serverid: item.id}}"
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
  import { sync } from 'vuex-pathify'
  import ApiService from '@/services/api.service'
  import InstanceSelector from '../../../components/app/InstanceSelector.vue'
  export default {
    name: 'ServerTableView',
    components: { InstanceSelector },

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

    computed: {
      ...sync('context', [
        'selected',
      ]),
    },

    watch: {
      selected (news, olds) {
        this.refresh()
      },
    },

    created () {
      this.refresh()
    },

    methods: {
      initialize () {},

      refresh () {
        this.items = []
        if (!this.selected) return
        this.loading = true
        var body = { subject: 'usvc.opc.' + this.selected.key + '.servers.getall' }
        ApiService.post('nats/request', body)
          .then(response => {
            this.items = response.data
            this.loading = false
          }).catch(response => {
            console.log('ERROR response: ' + JSON.stringify(response))
          })
      },

      rowclick (item) {
        this.$router.push({ name: 'inner/opcda/TagBrowser', params: { serverid: item.id } })
      },
    },
  }
</script>

<style lang="sass">
.instance-selector
  width: 250px
</style>
