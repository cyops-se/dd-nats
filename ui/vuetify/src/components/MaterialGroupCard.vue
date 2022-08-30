<template>
  <v-card
    v-bind="$attrs"
    class="v-card--material mt-4"
  >
    <v-card-title class="align-start">
      <v-sheet
        :color="color"
        width="100%"
        class="overflow-hidden mt-n9 transition-swing v-card--material__sheet d-flex justify-center"
        justify="center"
        elevation="6"
        max-width="100%"
        rounded
      >
        <div class="ml-3 my-auto">
          <v-icon large>
            mdi-tag-multiple
          </v-icon>
        </div>
        <v-spacer />
        <div class="pa-3 white--text my-auto">
          <span class="text-h3 text-no-wrap">
            {{ copy.name }}
          </span>
          <span class="text-h4 text-no-wrap">
            {{ copy.status >= 1 ? 'RUNNING' : 'STOPPED' }}
          </span>
          <div>Send count: {{ copy.counter }}</div>
        </div>
        <v-spacer />
        <div class="my-auto mr-3">
          <v-btn @click="startStop">
            <div v-html="copy.status >= 1 ? 'STOP' : 'START'" />
          </v-btn>
        </div>
      </v-sheet>

      <div class="pl-3 text-h4 v-card--material__title">
        <div class="text-subtitle-1 mb-n4 mt-4">
          <template>
            {{ copy.description }}
          </template>
        </div>
      </div>
    </v-card-title>

    <slot />

    <template>
      <v-divider class="mt-2 mx-4" />

      <v-card-actions class="px-4 text-caption grey--text">
        <v-icon
          class="mr-1"
          small
        >
          mdi-clock-outline
        </v-icon>

        <span
          class="text-caption grey--text font-weight-light"
          v-text="'Last run: ' + copy.lastrun.replace('T', ' ').substring(0, 19)"
        />
      </v-card-actions>
    </template>
  </v-card>
</template>

<script>
  import ApiService from '@/services/api.service'
  import WebsocketService from '@/services/websocket.service'
  export default {
    name: 'MaterialGroupCard',

    inheritAttrs: false,

    props: {
      group: {
        type: Object,
        default: () => ({}),
      },
      eventHandlers: {
        type: Array,
        default: () => ([]),
      },
    },

    data: () => ({
      color: 'error',
      copy: {},
    }),

    watch: {
      $route (to, from) {
        console.log('route change: ', to, from)
      },
    },

    created () {
      this.copy = Object.assign({}, this.group)
      this.color = this.group.status === 0 ? 'error' : this.group.status === 1 ? 'success' : 'warning'
      WebsocketService.topic('data.group', this, function (topic, group, target) {
        if (target.copy.ID === group.ID) target.updateGroup(target, group)
      })
      WebsocketService.topic('group.failed', this, function (topic, group, target) {
        if (target.copy.ID === group.ID) target.updateGroup(target, group)
      })
      WebsocketService.topic('group.started', this, function (topic, group, target) {
        if (target.copy.ID === group.ID) {
          target.updateGroup(target, group)
          target.$notification.success('Group started')
        }
      })
      WebsocketService.topic('group.stopped', this, function (topic, group, target) {
        if (target.copy.ID === group.ID) {
          group.status = 0
          target.updateGroup(target, group)
          target.$notification.success('Group stopped')
        }
      })
      WebsocketService.topic('group.warning', this, function (topic, group, target) {
        if (target.copy.ID === group.ID) {
          target.updateGroup(target, group)
          target.$notification.warning('Partial start of group, see logs')
        }
      })
    },

    methods: {
      startStop () {
        var action = this.copy.status >= 1 ? 'stop' : 'start'
        ApiService.get('opc/group/' + action + '/' + this.group.ID)
          .then(response => {
          }).catch(response => {
            this.$notification.error('Failed to start group')
            // console.log('ERROR response: ' + response.message)
            // this.$notification.error('Failed to start collection of group tags: ' + response.message)
          })
      },

      refresh () {
        ApiService.get('opc/group/' + this.group.ID)
          .then(response => {
            this.copy = response.data
          }).catch(response => {
            console.log('ERROR response (refresh): ' + response.message)
          })
      },

      updateGroup (target, group) {
        if (target.copy.ID === group.ID) {
          target.copy.status = group.status
          target.color = group.status === 0 ? 'error' : group.status === 1 ? 'success' : 'warning'
          target.copy = group
        }
      },
    },
  }
</script>

<style lang="sass">
.group-button
  font-size: .875rem !important
  margin-left: auto
  text-align: right
</style>
