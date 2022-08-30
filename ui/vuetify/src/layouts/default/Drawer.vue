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
            v-if="!link.subLinks"
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
            v-else
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
              v-for="sublink in link.subLinks"
              :key="sublink.text"
              :to="sublink.to"
              :active-class="`${color} lighten-3 ${theme.isDark ? 'black' : 'white'}--text`"
            >
              <!--v-list-item-icon>
                <v-icon>{{ sublink.icon }}</v-icon>
              </v-list-item-icon-->
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
          icon: 'mdi-server',
          text: 'OPC DA Servers',
          to: '/pages/servers',
        },
        {
          icon: 'mdi-folder-open',
          text: 'Diode Endpoints',
          to: '/pages/diodeproxies',
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
        {
          icon: 'mdi-history',
          text: 'Tag History',
          to: '/pages/cache',
        },
        {
          icon: 'mdi-transfer',
          text: 'File Transfer',
          to: '/pages/filetransfer',
        },
        {
          icon: 'mdi-view-list',
          to: '/tables/logs',
          text: 'Logs',
        },
        {
          icon: 'mdi-account-group',
          to: '/tables/users',
          text: 'Users',
        },
        {
          icon: 'mdi-cog',
          to: '/pages/systemsettings',
          text: 'Settings',
        },
      ],
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
    },
  }
</script>

<style lang="sass">
#default-drawer
  .v-list-group__items
    .v-list-item
      font-size: .8rem
</style>
