using System;
using System.Collections.Generic;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using MQTTnet;
using MQTTnet.Client;
using MQTTnet.Formatter;
using MQTTnet.Server;

namespace DdUsvc
{
    internal class DdUsvcMqttBroker : IMessageBroker
    {
        public string Url { get => _url; set => _url = Url; }
        public bool AutoReconnect { get => _autoReconnect; set => _autoReconnect = value; }
        private MqttFactory _mqttFactory = null;
        private IMqttClient _client = null;
        private string _url = "mqtt://localhost:1883";
        private bool _autoReconnect = true;
        private bool _isConnecting = false;
        private Dictionary<string, IMessageHandler> _subs = new Dictionary<string, IMessageHandler>();

        public DdUsvcMqttBroker(string url) {
            _url = url;

            _mqttFactory = new MQTTnet.MqttFactory();
            _client = _mqttFactory.CreateMqttClient();
            _client.ConnectedAsync += client_ConnectedAsync;
            _client.DisconnectedAsync += client_DisconnectedAsync;
        }

        private Task client_DisconnectedAsync(MqttClientDisconnectedEventArgs arg)
        {
            Console.WriteLine("Disconnect event received, killing client task");
            _client.ApplicationMessageReceivedAsync -= client_ApplicationMessageReceivedAsync;
            if (!_isConnecting) Connect();
            return Task.CompletedTask;
        }

        private Task client_ConnectedAsync(MqttClientConnectedEventArgs arg)
        {
            _client.ApplicationMessageReceivedAsync += client_ApplicationMessageReceivedAsync;
            return Task.CompletedTask;
        }

        internal DdUsvcError heartbeat(string topic, string responsetopic, byte[] data)
        {
            Console.WriteLine($"heartbeat: {Encoding.UTF8.GetString(data)}");
            return new DdUsvcError();
        }

        private Task client_ApplicationMessageReceivedAsync(MqttApplicationMessageReceivedEventArgs e)
        {
            IMessageHandler handler = null;
            if (_subs.TryGetValue(e.ApplicationMessage.Topic, out handler))
            {
                handler(e.ApplicationMessage.Topic, e.ApplicationMessage.ResponseTopic, e.ApplicationMessage.PayloadSegment.Array);
            }

            return Task.CompletedTask;
        }

        public DdUsvcError Connect()
        {
            while (!_client.IsConnected)
            {
                try
                {
                    _isConnecting = true;
                    var uri = new Uri(_url);
                    var mqttClientOptions = new MqttClientOptionsBuilder().WithTcpServer(uri.Host).WithProtocolVersion(MqttProtocolVersion.V500).Build();
                    _client.ConnectAsync(mqttClientOptions, CancellationToken.None).Wait();
                    Console.WriteLine("The MQTT client is connected!");
                }
                catch (Exception e)
                {
                    Console.WriteLine($"Failed to connect MQTT, retrying in 5 secs ...");
                    Thread.Sleep(5000);
                }
            }

            _isConnecting = false;
            return new DdUsvcError();
        }

        public DdUsvcError Disconnect()
        {
            if (!_client.IsConnected) _client.DisconnectAsync();
            return new DdUsvcError();
        }

        public DdUsvcError Publish(string topic, byte[] data)
        {
            DdUsvcError error = new DdUsvcError();
            if (_client.IsConnected)
            {
                topic = topic.Replace(".", "/");
                var msg = new MqttApplicationMessageBuilder().WithTopic(topic).WithPayload(data).Build();
                _client.PublishAsync(msg, CancellationToken.None);
            } else
            {
                error.Code = DdUsvcErrorCode.Error;
                error.Reason = "Failed to publish message: MQTT broker not connected";
            }
            return error;
        }

        public byte[] Request(string topic, byte[] data)
        {
            throw new NotImplementedException();
        }

        public void Subscribe(string topic, IMessageHandler callback)
        {
            topic = topic.Replace(".", "/");
            var options = new MqttTopicFilterBuilder().WithTopic(topic).Build();
            _client.SubscribeAsync(options, CancellationToken.None).Wait();
            _subs[topic] = callback;
        }
    }
}
