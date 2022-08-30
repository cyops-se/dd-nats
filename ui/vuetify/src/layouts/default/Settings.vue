<template>
  <div id="settings-wrapper">
    <v-card
      id="settings"
      class="py-2 px-4"
      color="rgba(0, 0, 0, .3)"
      dark
      flat
      link
      min-width="100"
      style="position: fixed; top: 115px; right: -35px; border-radius: 8px; z-index: 1;"
    >
      <v-icon large>
        mdi-cog
      </v-icon>
    </v-card>

    <v-menu
      v-model="menu"
      :close-on-content-click="false"
      activator="#settings"
      bottom
      content-class="v-settings"
      left
      nudge-left="8"
      offset-x
      origin="top right"
      transition="scale-transition"
    >
      <v-card
        class="text-center mb-0"
        width="300"
      >
        <v-card-text>
          <strong class="mb-3 d-inline-block">SETTINGS</strong>

          <v-row
            align="center"
            no-gutters
          >
            <v-col cols="auto">
              Dark Mode
            </v-col>

            <v-spacer />

            <v-col cols="auto">
              <v-switch
                v-model="$vuetify.theme.dark"
                class="ma-0 pa-0"
                color="secondary"
                hide-details
              />
            </v-col>
          </v-row>

          <v-divider class="my-4 secondary" />

          <v-row
            align="center"
            no-gutters
          >
            <v-col cols="auto">
              Sidebar Mini
            </v-col>

            <v-spacer />

            <v-col cols="auto">
              <v-switch
                v-model="mini"
                class="ma-0 pa-0"
                color="secondary"
                hide-details
                :change="onDarkChange()"
              />
            </v-col>
          </v-row>
          {{ settings.dark }}
        </v-card-text>
      </v-card>
    </v-menu>
  </div>
</template>

<script>
  // Mixins
  import Proxyable from 'vuetify/lib/mixins/proxyable'

  // Vuex
  import { get, sync } from 'vuex-pathify'

  export default {
    name: 'DashboardCoreSettings',

    mixins: [Proxyable],

    data: () => ({
      menu: false,
    }),

    computed: {
      ...sync('app', [
        'drawer',
        'mini',
      ]),
      settings: sync('user/settings'),
    },

    created () {
      this.$vuetify.theme.dark = this.settings.dark
    },

    methods: {
      onDarkChange (event) {
        this.settings.dark = this.$vuetify.theme.dark
        this.$store.set('user/settings', this.settings)
        this.$store.dispatch('user/update')
      },
    },
  }
</script>

<style lang="sass">
  .v-settings
    .v-item-group > *
      cursor: pointer

    &__item
      border-width: 3px
      border-style: solid
      border-color: transparent !important

      &--active
        border-color: #00cae3 !important
</style>
