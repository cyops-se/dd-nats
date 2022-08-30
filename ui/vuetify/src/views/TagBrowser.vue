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
        ApiService.get('data/key_value_pairs/field/key/tagpathdelimiter')
          .then(response => {
            if (response.data) this.delimiter = response.data[0].value
            console.log('delimiter: ', this.delimiter)
          }).catch(response => {
            console.log('Failed to get delimiter: ' + response.message)
          })
      },

      refresh () {
        ApiService.get('opc/tag/names')
          .then(response => {
            this.tags = response.data
            console.log('tags: ', JSON.stringify(this.tags))
          }).catch(response => {
            console.log('ERROR response: ' + response.message)
            this.$notification.error('Failed to get tags: ' + response.message)
          })
      },

      rootSelected () {
        console.log('root selected')
        this.items = []
        ApiService.get('opc/server/' + this.$route.params.serverid + '/root')
          .then(response => {
            if (response.data.branches) {
              for (var i = 0; i < response.data.branches.length; i++) {
                var item = { name: response.data.branches[i], children: [], path: response.data.branches[i] }
                this.items.push(item)
              }
            }

            if (response.data.leaves) {
              for (i = 0; i < response.data.leaves.length; i++) {
                this.items.push({ name: response.data.leaves[i], file: 'tag' })
              }
            }
            this.tree = this.tags
          }).catch(response => {
            console.log('ERROR: ' + response.message)
          })
      },

      tagSelected (selecteditems) {
        if (selecteditems && selecteditems.length > 0 && selecteditems.length !== this.tags.length) {
          console.log('posting tags: ' + JSON.stringify(selecteditems))
          ApiService.post('opc/tag/names', selecteditems)
            .then(({ data }) => {
              console.log('new tags response: ' + JSON.stringify(data))
            }).then(data => {
              this.refresh()
            }).catch(response => {
              console.log('new tags ERROR response: ' + response.message)
            })
        }
      },

      activated (item) {
        console.log('item activated' + JSON.stringify(item))
        if (item.file === 'tag') {
          // delete
          ApiService.delete('data/opc_tags/field/name/' + encodeURIComponent(item.path))
            .then(({ data }) => {
              console.log('deleted tag response: ' + JSON.stringify(data))
            }).then(data => {
              this.refresh()
            }).catch(response => {
              console.log('deleted tag ERROR response: ' + response.message)
            })
        } else {
          // add
          var tag = item.path.replaceAll('/', this.delimiter)
          ApiService.post('opc/tag/names', [tag])
            .then(({ data }) => {
              console.log('new tags response: ' + JSON.stringify(data))
            }).then(data => {
              this.refresh()
            }).catch(response => {
              console.log('new tags ERROR response: ' + response.message)
            })
        }

        item.file = item.file === 'tag' ? 'tagoutline' : 'tag'
      },

      async loadBranch (item) {
        console.log('branch item: ' + JSON.stringify(item))
        var branch = item.path.replaceAll('/', '.')
        console.log('loading', branch)
        let uri = 'opc/server/' + this.$route.params.serverid + '/list/' + branch
        if (branch === 'root') uri = 'opc/server/' + this.$route.params.serverid + '/root'
        return ApiService.get(uri)
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
                  if (this.tags[tn] === path) {
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
