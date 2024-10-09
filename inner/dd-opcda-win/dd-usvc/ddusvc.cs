using DdUsvc;
using NATS.Client.Internals.SimpleJSON;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Net;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using Newtonsoft.Json;
using System.Runtime.Serialization;
using System.Runtime.CompilerServices;
using NATS.Client;

namespace DdUsvc
{
    public class StatusResponse
    {
        [JsonProperty("success")]
        public bool Success { get; set; }
        [JsonProperty("statusmsg")]
        public string StatusMessage { get; set; }
    }

    public class GetSettingsResponse : StatusResponse
    {
        [JsonProperty("items")]
        public Dictionary<string, string> Items { get; set; }

    }

    public class SetSettingsResponse
    {
        [JsonProperty("items")]
        public Dictionary<string, string> Items { get; set; }

    }

    public enum DdUsvcErrorCode
    {
        OK = 0,
        Error = 1,
    }

    public struct DdUsvcError
    {
        public DdUsvcErrorCode Code { get; set; }
        public string Reason { get; set; }
    }

    public struct DdUsvcResponse
    {
        public DdUsvcError Error { get; set; }
        public byte[] Payload { get; set; }
    }

    public struct DdUsvcHeartbeat
    {
        [JsonProperty("hostname")]
        public string Hostname { get; set; }
        [JsonProperty("appname")]
        public string Name { get; set; }
        [JsonProperty("version")]
        public string Version { get; set; }
        [JsonProperty("identity")]
        public string Identity { get; set; }
        [JsonProperty("timestamp")]
        public string Timestamp { get; set; }

        public DdUsvcHeartbeat(string hostname, string name, string version, string timestamp, string identity)
        {
            Hostname = hostname;
            Name = name;
            Version = version;
            Timestamp = timestamp;
            Identity = identity;
        }
    }

    public delegate DdUsvcError IMessageHandler(string topic, string responsetopic, byte[] data);

    public interface IMessageBroker
    {
        string Url { get; set; }
        DdUsvcError Connect();
        DdUsvcError Disconnect();
        DdUsvcError Publish(string topic, byte[] data);
        byte[] Request(string topic, byte[] data);
        void Subscribe(string topic, IMessageHandler callback);
    }

    public class DdUsvc
    {
        public string Name { get; set; }
        public string Version { get; set; }
        protected Dictionary<string, string> settings;
        protected DdUsvcError lasterror;
        protected static IMessageBroker broker;

        public DdUsvc(string name, string[] args)
        {
            this.Name = name;
            this.Version = "1";
            settings = initSettings(args);
            broker = ddmb.NewMessageBroker(settings["url"]);
            if (broker == null) throw new Exception("Failed to create broker with url: {0}");
            broker.Connect();

            var shortname = name.Replace("-", "");
            this.Subscribe($"usvc.{shortname}.{settings["instance-id"]}.settings.get", this.getSettings);
            this.Subscribe($"usvc.{shortname}.{settings["instance-id"]}.settings.set", this.setSettings);

            Task heartbeat = Task.Run(() => this.sendHeartbeat());
        }

        public DdUsvcError Publish(string topic, object payload)
        {
            var data = JsonConvert.SerializeObject(payload);
            byte[] bytes = Encoding.UTF8.GetBytes(data);
            this.lasterror = broker.Publish(topic, bytes);
            return this.lasterror;
        }
        public DdUsvcResponse Request(string topic, object payload) { return new DdUsvcResponse(); }
        public void Subscribe(string topic, IMessageHandler callback)
        {
            broker.Subscribe(topic, callback);
        }

        internal Dictionary<string, string> initSettings(string[] args)
        {
            var s = new Dictionary<string, string>();
            s["url"] = args.Length > 0 ? args[0] : "nats://localhost:4222";
            s["instance-id"] = (args.Length > 1) ? args[1] : "default";
            s["workdir"] = (args.Length > 2) ? args[2] : System.AppDomain.CurrentDomain.BaseDirectory;
            return s;
        }

        internal void sendHeartbeat()
        {
            var hostname = System.Environment.MachineName;
            var now = DateTime.UtcNow.ToString("yyyy-MM-dd'T'HH:mm:ss.fffK");
            var heartbeat = new DdUsvcHeartbeat(hostname, Name, Version, now, settings["instance-id"]);
            while (true)
            {
                heartbeat.Timestamp = DateTime.UtcNow.ToString("yyyy-MM-dd'T'HH:mm:ss.fffK");
                this.Publish("system.heartbeat", heartbeat);
                Thread.Sleep(1000);
            }
        }

        internal DdUsvcError getSettings(string topic, string responsetopic, byte[] data)
        {
            try
            {
                var response = new GetSettingsResponse();
                response.Items = settings;
                response.Success = true;

                this.Publish(responsetopic, response);
            }
            catch (Exception ex)
            {
                Console.WriteLine(ex.Message);
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;
            }

            return this.lasterror;
        }

        internal DdUsvcError setSettings(string topic, string responsetopic, byte[] data)
        {
            var response = new StatusResponse();
            response.Success = false;
            response.StatusMessage = "Sorry, it is not possible to modify settings yet in the .NET version!";
            this.Publish(responsetopic, response);
            //try
            //{
            //    var request = JsonConvert.DeserializeObject<SetSettingsResponse>(Encoding.UTF8.GetString(data));
            //    settings = request.Items;
            //    var response = new StatusResponse();
            //    response.Success = true;
            //    this.Publish(responsetopic, response);
            //}
            //catch (Exception ex)
            //{
            //    Console.WriteLine (ex.Message );
            //    lasterror.Code = DdUsvcErrorCode.Error;
            //    lasterror.Reason = ex.Message;
            //}

            return this.lasterror;
        }
    }
}
