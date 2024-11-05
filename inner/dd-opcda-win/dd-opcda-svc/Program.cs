using DdOpcDaLib;
using System;
using System.Collections.Generic;
using System.Linq;
using System.ServiceProcess;
using System.Text;
using System.Threading.Tasks;

namespace DdOpcDaSvc
{
    internal static class Program
    {
        /// <summary>
        /// The main entry point for the application.
        /// </summary>
        static void Main(string[] args)
        {
            System.IO.Directory.SetCurrentDirectory(System.AppDomain.CurrentDomain.BaseDirectory);
            ServiceBase[] ServicesToRun;
            ServicesToRun = new ServiceBase[]
            {
                new SamplerSvc(args)
            };
            try
            {
                ServiceBase.Run(ServicesToRun);
            }
            catch (Exception ex)
            {
                Console.WriteLine($"CRITICAL error: {ex.ToString()}");
            }
        }
    }
}
