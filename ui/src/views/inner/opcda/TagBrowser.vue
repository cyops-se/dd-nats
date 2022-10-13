<template>
  <v-container
    id="browser-view"
    fluid
    tag="section"
  >
    <v-row>
      <v-col cols="12">
        <material-card
          color="secondary"
          full-header
        >
          <template #heading>
            <div class="pa-6 white--text">
              <div class="text-h4 font-weight-light">
                OPC DA Tag Browser
              </div>

              <div class="text-subtitle-1">
                {{ progid }}
              </div>
            </div>
          </template>
          <v-row>
            <v-col cols="12">
              <template>
                <v-treeview
                  v-model="tree"
                  :items="items"
                  :load-children="loadBranch"
                  dense
                  hoverable
                  open-on-click
                  item-key="path"
                  :filter="filter"
                  :search="search"
                >
                  <template v-slot:prepend="{ item, open }">
                    <v-icon v-if="!item.file">
                      {{ open ? 'mdi-folder-open' : 'mdi-folder' }}
                    </v-icon>
                    <v-icon
                      v-else
                      @click="activated(item)"
                    >
                      {{ files[item.file] }}
                    </v-icon>
                  </template>
                </v-treeview>
              </template>
            </v-col>
          </v-row>
        </material-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script>
  import ApiService from '@/services/api.service'
  export default {
    name: 'TagBrowserView',
    data: () => ({
      position: '',
      breadcrumbs: [],
      files: {
        tag: 'mdi-tag',
        tagoutline: 'mdi-tag-outline',
      },
      items: [],
      progid: '',
      tags: [],
      tree: [],
      search: '',
      settings: {},
      delimiter: '.',
    }),

    computed: {
      filter () {
        return this.caseSensitive
          ? (item, search, textKey) => item[textKey].indexOf(search) > -1
          : undefined
      },
    },

    created () {
      // this.refresh()
    },

    mounted () {
      this.loadSettings()
      this.refresh()
      this.rootSelected()
    },

    methods: {
      loadSettings () {
        var request = { subject: 'usvc.ddnatsopcda.settings.get', payload: {} }
        ApiService.post('nats/request', request)
          .then(response => {
            this.settings = response.data.items
            this.delimiter = this.settings.tagpathdelimiter
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Failed to get groups: ' + response.message)
          })
      },

      refresh () {
        var request = { subject: 'usvc.opc.tags.getall', payload: { value: parseInt(this.$route.params.serverid) } }
        ApiService.post('nats/request', request)
          .then(response => {
            this.tags = response.data.items
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Failed to get tags: ' + response.message)
          })
      },

      rootSelected () {
        this.items = []
        var request = { subject: 'usvc.opc.servers.root', payload: { value: parseInt(this.$route.params.serverid) } }
        ApiService.post('nats/request', request)
          .then(response => {
            var payload = response.data
            if (payload.branches) {
              for (var i = 0; i < payload.branches.length; i++) {
                var item = { name: payload.branches[i], children: [], path: payload.branches[i] }
                this.items.push(item)
              }
            }

            if (payload.leaves) {
              for (i = 0; i < payload.leaves.length; i++) {
                this.items.push({ name: payload.leaves[i], file: 'tag' })
              }
            }
            this.tree = this.tags
          }).catch(response => {
            console.log('ERROR: ' + response.message)
          })
      },

      activated (item) {
        var op = item.file === 'tag' ? 'deletebyname' : 'add'
        var tag = item.path.replaceAll('/', this.delimiter)
        var payload = { sid: parseInt(this.$route.params.serverid), items: [{ tag: tag }] }
        var request = { subject: 'usvc.opc.tags.' + op, payload }
        ApiService.post('nats/request', request)
          .then(response => {
            if (response.data.success) {
              this.refresh()
              item.file = item.file === 'tag' ? 'tagoutline' : 'tag'
            } else {
              this.$notification.error('Failed to ' + op + ' tag: ' + response.data.statusmsg)
            }
          }).catch(response => {
            console.log('new tags ERROR response: ' + response.message)
          })
      },

      async loadBranch (item) {
        var branch = item.path.replaceAll('/', '.')
        var payload = { sid: parseInt(this.$route.params.serverid), branch: branch }
        var request = { subject: 'usvc.opc.servers.getbranch', payload }
        return ApiService.post('nats/request', request)
          .then(response => {
            if (response.data.branches) {
              for (var i = 0; i < response.data.branches.length; i++) {
                var itemname = response.data.branches[i]
                var path = item.path + '/' + itemname
                item.children.push({ name: itemname, children: [], path: path })
              }
            }

            if (response.data.leaves) {
              for (i = 0; i < response.data.leaves.length; i++) {
                itemname = response.data.leaves[i]
                path = item.path + '/' + itemname
                var icon = 'tagoutline'

                for (var tn = 0; tn < this.tags.length; tn++) {
                  var name = this.tags[tn].name.replaceAll('.', '/')
                  if (name === path) {
                    icon = 'tag'
                    break
                  }
                }
                item.children.push({ name: itemname, file: icon, path: path })
              }
            }
          }).then(data => {
            this.tree = this.tags
          }).catch(response => {
            console.log('ERROR: ' + response.message)
          })
      },
    },
  }
</script>
