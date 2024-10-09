using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using NATS.Client;

namespace DdUsvc
{
    internal class DdUsvcNatsBroker : IMessageBroker
    {
        public string Url { get => _url; set => _url = Url; }
        private NATS.Client.IConnection _connection;
        private string _url;

        public DdUsvcNatsBroker(string url)
        {
            _url = url;
        }

        public DdUsvcError Connect()
        {
            var factory = new NATS.Client.ConnectionFactory();
            while (_connection == null || _connection.IsClosed())
            {
                try
                {
                    _connection = factory.CreateConnection(Url, false);
                    Console.WriteLine("NATS connected");
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"Failed to connect NATS, retrying in 5 secs ...");
                    Thread.Sleep(5000);
                }
            }

            return new DdUsvcError();
        }

        public DdUsvcError Disconnect()
        {
            throw new NotImplementedException();
        }

        public DdUsvcError Publish(string topic, byte[] data)
        {
            DdUsvcError result = new DdUsvcError();

            try
            {
                _connection.Publish(topic, data);
            }
            catch (Exception ex)
            {
                result.Code = DdUsvcErrorCode.Error;
                result.Reason = ex.Message;
            }

            return result;
        }

        public byte[] Request(string topic, byte[] data)
        {
            throw new NotImplementedException();
        }

        public void Subscribe(string topic, IMessageHandler callback)
        {
            Task.Run(() =>
            {
                Console.WriteLine($"Subscription for topic: {topic}, START!");
                while (true)
                {
                    using (var s = _connection.SubscribeSync(topic))
                    {
                        var msg = s.NextMessage();
                        callback(topic, msg.Reply, msg.Data);
                    }
                }
                Console.WriteLine($"Subscription for topic: {topic}, END!");
            });

        }
    }
}
