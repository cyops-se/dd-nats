using DdOpcUaLib;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;

namespace dd_opcua_cli
{
    class Program
    {
        static async Task Main(string[] args)
        {
            Console.WriteLine($"args.Length: {args.Length}");
            var svc = new DdOpcUaUsvc("dd-nats-opcua", args);
            svc.Initialize();
            svc.Startup();

            svc.LogEvent("********** Press <Enter> to close **********");
            Console.ReadLine();
            svc.Shutdown();
        }
    }
}
