<template>
  <v-navigation-drawer
    id="default-drawer"
    v-model="drawer"
    :right="$vuetify.rtl"
    :mini-variant.sync="mini"
    mini-variant-width="80"
    app
    width="240"
  >
    <div class="px-2">
      <default-drawer-header />

      <v-divider class="mx-3 mb-2" />

      <v-list
        nav
        dense
      >
        <div
          v-for="(link) in links"
          :key="link.text"
        >
          <v-list-item
            v-if="!link.sublinks && showitem(link)"
            :to="link.to"
            class="v-list-item"
            :active-class="`${color} lighten-3 ${theme.isDark ? 'black' : 'white'}--text`"
          >
            <v-list-item-icon>
              <v-icon>{{ link.icon }}</v-icon>
            </v-list-item-icon>

            <v-list-item-title v-text="link.text" />
          </v-list-item>

          <v-list-group
            v-else-if="showitem(link)"
            :key="link.text"
            :prepend-icon="link.icon"
            :value="false"
            no-action
            color="grey"
          >
            <template v-slot:activator>
              <v-list-item-content>
                <v-list-item-title v-text="link.text" />
              </v-list-item-content>
            </template>

            <v-list-item
              v-for="sublink in link.sublinks"
              :key="sublink.text"
              :to="sublink.to"
              :active-class="`${color} lighten-3 ${theme.isDark ? 'black' : 'white'}--text`"
            >
              <!--
              <v-list-item-icon>
                <v-icon>{{ sublink.icon }}</v-icon>
              </v-list-item-icon>
              -->
              <v-list-item-title>{{ sublink.text }}</v-list-item-title>
            </v-list-item>
          </v-list-group>
        </div>
        <!-- div
          v-for="(svc, name) in services"
          :key="name"
        >
          {{ name }} => {{ svc.state }}
        </div -->
      </v-list>
    </div>

    <template #append>
      <p
        v-for="(link, i) in about"
        :key="i"
        class="text-center"
        cols="6"
        md="auto"
      >
        <a
          :href="link.href"
          class="text-decoration-none text-uppercase text-caption font-weight-regular grey--text"
          rel="noopener"
          target="_blank"
          v-text="link.text"
        />
      </p>
    </template>

    <div class="pt-12" />
  </v-navigation-drawer>
</template>

<script>
  // Utilities
  import { get, sync } from 'vuex-pathify'
  import WebsocketService from '@/services/websocket.service'

  export default {
    name: 'DefaultDrawer',

    inject: ['theme'],

    components: {
      DefaultDrawerHeader: () => import(
        /* webpackChunkName: "default-drawer-header" */
        './widgets/DrawerHeader'
      ),
    },

    props: {
      color: {
        type: String,
        default: 'secondary',
      },
    },

    data: () => ({
      about: [
        {
          href: 'http://cyops.se/en/about',
          text: 'About cyops.se',
        },
        // {
        //   href: 'http://cyops.se/blog',
        //   text: 'Blog',
        // },
        {
          href: 'http://github.com/cyops-se/dd-opcda/LICENSES.md',
          text: 'Licenses',
        },
      ],
      links: [
        {
          to: '/',
          icon: 'mdi-view-dashboard',
          text: 'Dashboard',
        },
        {
          icon: 'mdi-server',
          text: 'OPC DA',
          usvc: 'ddnatsopcda',
          sublinks: [
            {
              icon: 'mdi-server',
              text: 'Servers',
              to: '/pages/opc/servers',
            },
            {
              icon: 'mdi-folder-multiple',
              text: 'Sampling Groups',
              to: '/pages/opc/groups',
            },
            {
              icon: 'mdi-tag-multiple',
              text: 'Tags',
              to: '/pages/opc/tags',
            },
          ],
        },
        {
          icon: 'mdi-server',
          text: 'Modbus',
          to: '/pages/servers',
          usvc: 'ddnatsmodbus',
          sublinks: [
            {
              icon: 'mdi-server',
              text: 'Servers',
              to: '/pages/servers',
            },
            {
              icon: 'mdi-folder-multiple',
              text: 'Sampling Groups',
              to: '/pages/groups',
            },
            {
              icon: 'mdi-tag-multiple',
              text: 'Tags',
              to: '/pages/tags',
            },
          ],
        },
        {
          icon: 'mdi-history',
          text: 'Tag History',
          to: '/pages/cache',
          usvc: 'ddnatscache',
        },
        {
          icon: 'mdi-transfer',
          text: 'File Transfer',
          to: '/pages/filetransfer',
          usvc: 'ddnatsfileinner',
        },
        {
          icon: 'mdi-view-list',
          to: '/tables/logs',
          text: 'Logs',
          usvc: 'ddnatslogs',
        },
        {
          icon: 'mdi-cog',
          to: '/pages/systemsettings',
          text: 'Settings',
        },
      ],
      services: {},
    }),

    computed: {
      ...get('user', [
        'settings',
      ]),
      ...sync('app', [
        'drawer',
        'mini',
      ]),
    },

    created () {
      WebsocketService.topic('system.heartbeat', this, function (topic, msg, t) {
        if (t.services) {
          var appname = msg.appname.replaceAll('-', '')
          t.services = { ...t.services, [appname]: { name: appname, state: 'alive', count: 0, lastbeat: new Date() } }
        }
      })

      var t = this
      setInterval(function () {
        var now = new Date()
        for (const p in t.services) {
          if (!t.services[p].lastbeat || t.services[p].state === 'dead') continue
          var n = now.getSeconds()
          var lb = t.services[p].lastbeat.getSeconds()
          var diff = Math.abs(n - lb)
          if (diff > 5) {
            t.services[p].state = 'stalling'
            if (t.services[p].count++ > 3) {
              console.log('deleting: ' + p)
              t.services[p].state = 'dead'
            }
          }
        }
      }, 5000)
    },

    methods: {
      showitem (item) {
        if (!item || !item.usvc) return true
        var items = item.usvc.split('|')
        var result = false
        for (var i in items) {
          var usvcname = items[i]
          if (item.usvc && this.services[usvcname] && this.services[usvcname].state !== 'dead') result = true
        }
        return result
      },
    },
  }
</script>

<style lang="sass">
#default-drawer
  .v-list-group__items
    .v-list-item
      min-height: 28px
      .v-list-item__title
        padding-left: 10px
        font-size: .75rem

</style>
