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
            v-if="!link.sublinks && link.alive"
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
            v-else-if="link.alive"
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
              <v-list-item-title>{{ sublink.text }}</v-list-item-title>
            </v-list-item>
          </v-list-group>
        </div>
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
          icon: 'mdi-speedometer',
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
            {
              icon: 'mdi-cog',
              text: 'Settings',
              to: '/pages/opc/settings',
            },
          ],
        },
        {
          icon: 'mdi-speedometer',
          text: 'Modbus',
          usvc: 'ddnatsmodbus',
          sublinks: [
            {
              icon: 'mdi-server',
              text: 'Slaves',
              to: '/pages/modbus/slaves',
            },
            {
              icon: 'mdi-tag-multiple',
              text: 'Data points',
              to: '/pages/modbus/datapoints',
            },
            {
              icon: 'mdi-cog',
              text: 'Settings',
              to: '/pages/modbus/settings',
            },
          ],
        },
        {
          icon: 'mdi-history',
          text: 'Process cache',
          usvc: 'ddnatscache',
          sublinks: [
            {
              icon: 'mdi-transfer',
              text: 'History',
              to: '/pages/cache/history',
            },
            {
              icon: 'mdi-cog',
              text: 'Settings',
              to: '/pages/cache/settings',
            },
          ],
        },
        {
          icon: 'mdi-transfer',
          text: 'File Transfer',
          usvc: 'ddnatsfileinner',
          sublinks: [
            {
              icon: 'mdi-transfer',
              text: 'Transfer status',
              to: '/pages/innerfile/transfer',
            },
            {
              icon: 'mdi-cog',
              text: 'Settings',
              to: '/pages/innerfile/settings',
            },
          ],
        },
        {
          icon: 'mdi-server',
          text: 'Inner proxy',
          usvc: 'ddnatsinnerproxy',
          sublinks: [
            {
              icon: 'mdi-cog',
              text: 'Settings',
              to: '/pages/innerproxy/settings',
            },
          ],
        },
        {
          icon: 'mdi-server',
          text: 'Outer proxy',
          usvc: 'ddnatsouterproxy',
          sublinks: [
            {
              icon: 'mdi-cog',
              text: 'Settings',
              to: '/pages/outerproxy/settings',
            },
          ],
        },
        {
          icon: 'mdi-server',
          text: 'Timescale',
          usvc: 'ddnatstimescale',
          sublinks: [
            {
              icon: 'mdi-cog',
              text: 'Meta',
              to: '/pages/timescale/meta',
            },
            {
              icon: 'mdi-cog',
              text: 'Settings',
              to: '/pages/timescale/settings',
            },
          ],
        },
        {
          icon: 'mdi-tag',
          text: 'Filter',
          usvc: 'ddnatsprocessfilter',
          sublinks: [
            {
              icon: 'mdi-cog',
              text: 'Filtered points',
              to: '/pages/outerfilter/filteredpoints',
            },
            {
              icon: 'mdi-cog',
              text: 'Settings',
              to: '/pages/outerfilter/settings',
            },
          ],
        },
        {
          icon: 'mdi-view-list',
          to: '/tables/logs',
          text: 'System logs',
          usvc: 'ddlogger',
          sublinks: [
            {
              text: 'All',
              to: '/pages/logs/all',
            },
            {
              text: 'Info',
              to: '/pages/logs/info',
            },
            {
              text: 'Errors',
              to: '/pages/logs/errors',
            },
            {
              icon: 'mdi-cog',
              text: 'Settings',
              to: '/pages/logs/settings',
            },
          ],
        },
        {
          icon: 'mdi-cog',
          to: '/pages/systemsettings',
          text: 'Console settings',
        },
      ],
      // services: {},
    }),

    computed: {
      ...get('user', [
        'settings',
      ]),
      ...sync('app', [
        'drawer',
        'mini',
      ]),
      ...sync('usvc', [
        'services',
        'lastseen',
      ]),
    },

    watch: {
      // whenever question changes, this function will run
      lastseen (news, olds) {
        this.checkstates()
      },
    },

    created () {
      var t = this
      t.checkstates()
    },

    methods: {
      checkstates () {
        for (const i in this.links) this.showitem(i)
      },

      showitem (li) {
        var item = this.links[li]
        if (!item || !item.usvc) { item.alive = true; return true }
        var items = item.usvc.split('|')
        var result = false
        for (var i in items) {
          var usvcname = items[i]
          for (const i in this.services[usvcname]) {
            if (item.usvc && this.services[usvcname][i] && this.services[usvcname][i].alive) {
              item.alive = true
              break
            } else {
              item.alive = false
            }
          }
          this.links = { ...this.links, [li]: item }
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

  .v-list-group
    .v-list-group__header__append-icon
      margin-left: 0px

</style>
