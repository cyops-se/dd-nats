<template>
  <usvc-settings usvc="ddnatsouterproxy" />
</template>

<script>
  import UsvcSettings from '../../../components/usvc/UsvcSettings.vue'
  export default {
    name: 'SettingsView',
    components: { UsvcSettings },

    created () {
    },

    methods: {
      initialize () {},
    },
  }
</script>
