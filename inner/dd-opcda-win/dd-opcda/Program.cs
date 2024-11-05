using System;
using System.Threading.Tasks;
using DdOpcDaLib;

namespace DdOpcDaCli
{
    class Program
    { 
        static async Task Main(string[] args)
        {
            Console.WriteLine($"args.Length: {args.Length}");
            var svc = new DdOpcDaUsvc("dd-nats-opcda", args);
            svc.Startup();

            svc.LogEvent("********** Press <Enter> to close **********");
            Console.ReadLine();
            svc.Shutdown();
        }
    }
}
