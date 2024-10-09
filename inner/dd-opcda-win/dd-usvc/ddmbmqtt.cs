using System;
using MQTTnet;
using MQTTnet.Client;
using MQTTnet.Formatter;

namespace DdUsvc
{
    internal class DdUsvcMqttBroker : IMessageBroker
    {
        public string Url { get => _url; set => _url = Url; }
        private MQTTnet.Client.IMqttClient _client = null;
        private string _url;

        public DdUsvcMqttBroker(string url) {
            _url = url;
        }

        public DdUsvcError Connect()
        {
            throw new NotImplementedException();
        }

        public DdUsvcError Disconnect()
        {
            throw new NotImplementedException();
        }

        public DdUsvcError Publish(string topic, byte[] data)
        {
            throw new NotImplementedException();
        }

        public byte[] Request(string topic, byte[] data)
        {
            throw new NotImplementedException();
        }

        public void Subscribe(string topic, IMessageHandler callback)
        {
            throw new NotImplementedException();
        }
    }
}
