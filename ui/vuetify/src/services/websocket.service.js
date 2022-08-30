/**
 * Service to call HTTP request via Axios
 */
const WebsocketService = {
  baseURL: 'ws://' + window.location.host.replace('8080', '3000') + '/ws',
  connection: null,
  subscriptions: [],

  /**
   * Set the default HTTP request headers
   */
  connect: function (onclose) {
    this.connection = new WebSocket(this.baseURL)
    this.connection.onmessage = this.onmessage
    this.connection.onopen = this.onopen
    this.connection.onclose = onclose || this.onclose
    this.connection.subscriptions = this.subscriptions
  },

  onopen: function () {
    console.log('Websocket successfully connected')
  },

  onclose: function () {
    console.log('Websocket closed: ')
  },

  onmessage: function (event) {
    if (!event || !event.data) return
    var data = JSON.parse(event.data)
    // console.log('Websocket message: ' + JSON.stringify(data))

    if (!data || !data.topic || !data.message) return
    // var message = JSON.parse(data.message)
    var message = data.message

    // console.log('Websocket topic: ' + JSON.stringify(data.topic))
    // console.log('Websocket message: ' + JSON.stringify(message))
    if (this.subscriptions) {
      const subs = this.subscriptions[data.topic]
      if (subs) {
        for (var i = 0; i < subs.length; i++) {
          subs[i].callback(data.topic, message, subs[i].target)
        }
      }
    }
  },

  topic: function (name, target, callback) {
    if (!this.subscriptions[name]) this.subscriptions[name] = []
    var alreadyexists = false
    for (var i = 0; i < this.subscriptions[name].length; i++) {
      if (this.subscriptions[name][i].callback === callback) {
        alreadyexists = true
        break
      }
    }

    if (!alreadyexists) this.subscriptions[name].push({ target: target, callback: callback })
  },
}

export default WebsocketService
