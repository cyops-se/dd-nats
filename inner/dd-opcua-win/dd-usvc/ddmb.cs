using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;
using System.Xml.Linq;

namespace DdUsvc
{
    internal class ddmb
    {
        public static IMessageBroker NewMessageBroker(string url)
        {
            var name = url.ToLower();
            IMessageBroker mb = null;

            if (name.StartsWith("nats"))
            {
                Console.WriteLine($"Creating NATS broker: {url}");
                mb = new DdUsvcNatsBroker(name);
            }
            else if (name.StartsWith("mqtt"))
            {
                mb = new DdUsvcMqttBroker(name);
            }

            return mb;
        }
    }
}
