using System;
using System.Collections.Generic;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using MQTTnet;
using MQTTnet.Client;
using MQTTnet.Formatter;

namespace DdUsvc
{
    public class DdUsvcMqttBroker : IMessageBroker
    {
        public string Url { get => _url; set => _url = value; }
        public bool AutoReconnect { get => _autoReconnect; set => _autoReconnect = value; }

        private readonly MqttFactory _mqttFactory;
        private IMqttClient _client;
        private string _url = "mqtt://localhost:1883";
        private bool _autoReconnect = true;
        private bool _isConnecting = false;
        private readonly Dictionary<string, IMessageHandler> _subs = new Dictionary<string, IMessageHandler>(StringComparer.OrdinalIgnoreCase);
        private readonly object _sync = new object();

        public DdUsvcMqttBroker(string url)
        {
            if (!string.IsNullOrWhiteSpace(url))
                _url = url;

            _mqttFactory = new MqttFactory();
            _client = _mqttFactory.CreateMqttClient();
            _client.ConnectedAsync += client_ConnectedAsync;
            _client.DisconnectedAsync += client_DisconnectedAsync;
        }

        private Task client_DisconnectedAsync(MqttClientDisconnectedEventArgs arg)
        {
            Console.WriteLine("MQTT disconnected: " + arg.ReasonString);
            _client.ApplicationMessageReceivedAsync -= client_ApplicationMessageReceivedAsync;
            if (AutoReconnect && !_isConnecting)
            {
                Task.Run(() =>
                {
                    Thread.Sleep(2000);
                    try { Connect(); } catch { }
                });
            }
            return Task.CompletedTask;
        }

        private Task client_ConnectedAsync(MqttClientConnectedEventArgs arg)
        {
            Console.WriteLine("MQTT connected.");
            _client.ApplicationMessageReceivedAsync += client_ApplicationMessageReceivedAsync;

            foreach (var kv in _subs)
            {
                try
                {
                    _client.SubscribeAsync(new MqttTopicFilterBuilder().WithTopic(kv.Key).Build()).Wait();
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"Resubscribe failed for {kv.Key}: {ex.Message}");
                }
            }

            return Task.CompletedTask;
        }

        private Task client_ApplicationMessageReceivedAsync(MqttApplicationMessageReceivedEventArgs e)
        {
            try
            {
                IMessageHandler handler;
                if (_subs.TryGetValue(e.ApplicationMessage.Topic, out handler) && handler != null)
                {
                    var incomingTopic = e.ApplicationMessage.Topic.Replace("/", ".");
                    var replyTopic = e.ApplicationMessage.ResponseTopic;
                    handler(incomingTopic, replyTopic, e.ApplicationMessage.PayloadSegment.Array);
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine("MQTT message handler error: " + ex.Message);
            }
            return Task.CompletedTask;
        }

        public DdUsvcError Connect()
        {
            lock (_sync)
            {
                if (_client != null && _client.IsConnected) return new DdUsvcError { Code = DdUsvcErrorCode.OK };
                if (_client == null)
                {
                    _client = _mqttFactory.CreateMqttClient();
                    _client.ConnectedAsync += client_ConnectedAsync;
                    _client.DisconnectedAsync += client_DisconnectedAsync;
                }
            }

            while (true)
            {
                try
                {
                    _isConnecting = true;
                    var uri = new Uri(_url);

                    var builder = new MqttClientOptionsBuilder()
                        .WithTcpServer(uri.Host, uri.IsDefaultPort ? 1883 : uri.Port)
                        .WithProtocolVersion(MqttProtocolVersion.V500);

                    if (!string.IsNullOrEmpty(uri.UserInfo))
                    {
                        var parts = uri.UserInfo.Split(':');
                        if (parts.Length == 2)
                            builder = builder.WithCredentials(parts[0], parts[1]);
                        else
                            builder = builder.WithCredentials(uri.UserInfo);
                    }

                    var options = builder.Build();
                    _client.ConnectAsync(options, CancellationToken.None).Wait();

                    _isConnecting = false;
                    return new DdUsvcError { Code = DdUsvcErrorCode.OK };
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"Failed to connect MQTT ({ex.Message}), retrying in 5s ...");
                    _isConnecting = false;
                    if (!AutoReconnect)
                        return new DdUsvcError { Code = DdUsvcErrorCode.Error, Reason = ex.Message };
                    Thread.Sleep(5000);
                }
            }
        }

        public DdUsvcError Disconnect()
        {
            try
            {
                if (_client != null && _client.IsConnected)
                    _client.DisconnectAsync().Wait();
                return new DdUsvcError { Code = DdUsvcErrorCode.OK };
            }
            catch (Exception ex)
            {
                return new DdUsvcError { Code = DdUsvcErrorCode.Error, Reason = ex.Message };
            }
        }

        public DdUsvcError Publish(string topic, byte[] data)
        {
            var err = new DdUsvcError { Code = DdUsvcErrorCode.OK };
            try
            {
                if (_client == null || !_client.IsConnected)
                    throw new InvalidOperationException("MQTT client not connected.");

                var mqttTopic = ToMqttTopic(topic);
                var msg = new MqttApplicationMessageBuilder()
                    .WithTopic(mqttTopic)
                    .WithPayload(data)
                    .Build();
                _client.PublishAsync(msg, CancellationToken.None).Wait();
            }
            catch (Exception ex)
            {
                err.Code = DdUsvcErrorCode.Error;
                err.Reason = ex.Message;
            }
            return err;
        }

        public byte[] Request(string topic, byte[] data)
        {
            if (_client == null || !_client.IsConnected)
                throw new InvalidOperationException("MQTT client not connected.");

            var replyTopic = $"_rr/{Guid.NewGuid():N}";
            var mqttReplyTopic = replyTopic;
            var tcs = new TaskCompletionSource<byte[]>(TaskCreationOptions.RunContinuationsAsynchronously);

            _client.ApplicationMessageReceivedAsync += LocalHandler;
            _client.SubscribeAsync(new MqttTopicFilterBuilder().WithTopic(mqttReplyTopic).Build()).Wait();

            try
            {
                var mqttTopic = ToMqttTopic(topic);
                var msg = new MqttApplicationMessageBuilder()
                    .WithTopic(mqttTopic)
                    .WithResponseTopic(mqttReplyTopic)
                    .WithPayload(data)
                    .Build();

                _client.PublishAsync(msg, CancellationToken.None).Wait();

                if (tcs.Task.Wait(TimeSpan.FromSeconds(60)))
                    return tcs.Task.Result;
                return null;
            }
            finally
            {
                _client.ApplicationMessageReceivedAsync -= LocalHandler;
                try { _client.UnsubscribeAsync(mqttReplyTopic).Wait(); } catch { }
            }

            Task LocalHandler(MqttApplicationMessageReceivedEventArgs e)
            {
                if (string.Equals(e.ApplicationMessage.Topic, mqttReplyTopic, StringComparison.OrdinalIgnoreCase))
                {
                    tcs.TrySetResult(e.ApplicationMessage.PayloadSegment.Array);
                }
                return Task.CompletedTask;
            }
        }

        public void Subscribe(string topic, IMessageHandler callback)
        {
            if (_client == null || !_client.IsConnected)
            {
                var connectErr = Connect();
                if (connectErr.Code == DdUsvcErrorCode.Error)
                {
                    Console.WriteLine("Subscribe aborted (connect failed): " + connectErr.Reason);
                    return;
                }
            }

            var mqttTopic = ToMqttTopic(topic);
            _client.SubscribeAsync(new MqttTopicFilterBuilder().WithTopic(mqttTopic).Build()).Wait();
            _subs[mqttTopic] = callback;
        }

        private string ToMqttTopic(string dotted)
        {
            if (string.IsNullOrWhiteSpace(dotted)) return dotted;
            return dotted.Replace('.', '/');
        }
    }
}
