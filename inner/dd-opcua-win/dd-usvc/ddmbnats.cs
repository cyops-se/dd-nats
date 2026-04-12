using System;
using System.Threading;
using System.Threading.Tasks;
using NATS.Client;

namespace DdUsvc
{
    public class DdUsvcNatsBroker : IMessageBroker
    {
        public string Url
        {
            get => _url;
            set => _url = value;
        }

        public bool AutoReconnect
        {
            get => _autoReconnect;
            set => _autoReconnect = value;
        }

        private IConnection _connection;
        private string _url = "nats://127.0.0.1:4222";
        private bool _autoReconnect = true;
        private readonly object _connLock = new object();

        public DdUsvcNatsBroker(string url)
        {
            if (!string.IsNullOrWhiteSpace(url))
                _url = url;
        }

        public DdUsvcError Connect()
        {
            try
            {
                EnsureConnected();
                return new DdUsvcError { Code = DdUsvcErrorCode.OK };
            }
            catch (Exception ex)
            {
                return new DdUsvcError { Code = DdUsvcErrorCode.Error, Reason = ex.Message };
            }
        }

        public DdUsvcError Disconnect()
        {
            lock (_connLock)
            {
                try
                {
                    _connection?.Drain();
                    _connection?.Close();
                    _connection = null;
                    return new DdUsvcError { Code = DdUsvcErrorCode.OK };
                }
                catch (Exception ex)
                {
                    return new DdUsvcError { Code = DdUsvcErrorCode.Error, Reason = ex.Message };
                }
            }
        }

        public DdUsvcError Publish(string topic, byte[] data)
        {
            var result = new DdUsvcError { Code = DdUsvcErrorCode.OK };
            try
            {
                EnsureConnected();
                _connection.Publish(topic, data);
                _connection.Flush(5000);
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
            const int timeoutMs = 60000;
            try
            {
                EnsureConnected();
                var msg = _connection.Request(topic, data, timeoutMs);
                return msg?.Data;
            }
            catch (NATSTimeoutException)
            {
                Console.WriteLine($"Request timeout on subject '{topic}' after {timeoutMs}ms");
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Request error on '{topic}': {ex.Message}");
            }
            return null;
        }

        public void Subscribe(string topic, IMessageHandler callback)
        {
            Task.Run(() =>
            {
                while (true)
                {
                    try
                    {
                        EnsureConnected();
                        using (var sub = _connection.SubscribeAsync(topic))
                        {
                            sub.MessageHandler += (s, args) =>
                            {
                                try
                                {
                                    callback(topic, args.Message.Reply, args.Message.Data);
                                }
                                catch (Exception handlerEx)
                                {
                                    Console.WriteLine($"Subscriber callback error on '{topic}': {handlerEx.Message}");
                                }
                            };
                            sub.Start();
                            while (_connection != null && !_connection.IsClosed())
                                Thread.Sleep(500);
                        }
                    }
                    catch (Exception ex)
                    {
                        Console.WriteLine($"Subscription loop error on '{topic}': {ex.Message}");
                        Thread.Sleep(2000);
                    }
                }
            });
        }

        private void EnsureConnected()
        {
            if (_connection != null && !_connection.IsClosed()) return;

            lock (_connLock)
            {
                if (_connection != null && !_connection.IsClosed()) return;

                var cf = new ConnectionFactory();
                var opts = ConnectionFactory.GetDefaultOptions();
                opts.Url = _url;
                opts.AllowReconnect = _autoReconnect;
                opts.MaxReconnect = Options.ReconnectForever;
                opts.ReconnectWait = 2000;
                opts.Timeout = 5000;
                opts.ClosedEventHandler += (s, a) =>
                {
                    Console.WriteLine("NATS connection closed.");
                };
                opts.DisconnectedEventHandler += (s, a) =>
                {
                    Console.WriteLine("NATS disconnected.");
                };
                opts.ReconnectedEventHandler += (s, a) =>
                {
                    Console.WriteLine("NATS reconnected.");
                };

                int attempt = 0;
                while (true)
                {
                    attempt++;
                    try
                    {
                        _connection = cf.CreateConnection(opts);
                        Console.WriteLine("NATS connected");
                        break;
                    }
                    catch (Exception ex)
                    {
                        if (!_autoReconnect)
                            throw new InvalidOperationException("Failed to connect and AutoReconnect disabled: " + ex.Message, ex);

                        Console.WriteLine($"Failed to connect NATS (attempt {attempt}), retrying in 5s ... {ex.Message}");
                        Thread.Sleep(5000);
                    }
                }
            }
        }
    }
}
