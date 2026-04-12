using DdOpcUaLib;
using System;
using System.Diagnostics;
using System.ServiceProcess;

namespace DdOpcUaSvc
{
    public partial class SamplerSvc : ServiceBase
    {
        private static string evtSource = "GEMIT";
        private static string evtLog = "DdOpcUa";
        private EventLog eventLog;
        private DdOpcUaUsvc usvc;

        public SamplerSvc(string[] args)
        {
            InitializeComponent();
            eventLog = new EventLog();
            if (!EventLog.SourceExists(evtSource))
            {
                EventLog.CreateEventSource(evtSource, evtLog);
            }

            eventLog.Source = evtSource;
            eventLog.Log = evtLog;
            DdUsvc.DdUsvc.EventLog = eventLog;

            try
            {
                eventLog.WriteEntry($"Init arguments {args}");
                usvc = new DdOpcUaUsvc("dd-nats-opcua", args);
            } catch ( Exception ex)
            {
                eventLog.WriteEntry($"Init failed, exception: {ex.Message}");
            }
        }

        protected override void OnStart(string[] args)
        {
            try
            {
                eventLog.WriteEntry($"OnStart: starting microservice: {usvc.Name}");
                usvc.Initialize();
                usvc.Startup();
            } catch ( Exception ex )
            {
                eventLog.WriteEntry($"OnStart: failed, exception: {ex.Message}");
            }
        }

        protected override void OnStop()
        {
            try
            {
                eventLog.WriteEntry($"OnStop: shutting down microservice: {usvc.Name}");
                usvc.Shutdown();
            }
            catch (Exception ex)
            {
                eventLog.WriteEntry($"OnStop: failed, exception: {ex.Message}");
            }
        }
    }
}
