using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Net.Sockets;
using System.Runtime.CompilerServices;
using System.Runtime.InteropServices;
using System.Security.Policy;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using System.Timers;
using DdUsvc;
using MQTTnet;
using MQTTnet.Client;
using MQTTnet.Formatter;
using NATS.Client;
using Newtonsoft.Json;
using OPC.Common;
using OPC.Data;

namespace DdOpcDaLib
{
    public struct DataPoint
    {
        [JsonProperty("t")]
        public DateTime Time { get; set; }
        [JsonProperty("n")]
        public string Name { get; set; }
        [JsonProperty("v")]
        public double Value { get; set; }
        [JsonProperty("q")]
        public int Quality { get; set; }
    }

    public struct DataMessage
    {
        [JsonProperty("version")]
        public int Version { get; set; }
        [JsonProperty("group")]
        public string Group { get; set; }
        [JsonProperty("interval")]
        public int Interval { get; set; }
        [JsonProperty("sequence")]
        public int Sequence { get; set; }
        [JsonProperty("count")]
        public int Count { get; set; }
        [JsonProperty("points")]
        public DataPoint[] Points { get; set; }
    }

    public struct SamplingGroup
    {
        public int Id { get; set; }
        public string Name { get; set; }
        public int SamplingTime { get; set; }
        public string ProgId { get; set; }
        public List<string> Tags { get; set; }
    }

    public class DdOpcDa
    {
        private static System.Timers.Timer aTimer;
        private static OpcServer opcServer;
        private static List<OpcGroup> opcGroups;
        private static List<SamplingGroup> groups;
        private static string progID;
        private static UdpClient diodeClient;
        private static Dictionary<int, SamplingGroup> samplingGroups;
        private static IConnection nc = null;
        private static MQTTnet.Client.IMqttClient mqttClient = null;
        private static string version;
        private static EventLog eventLog;
        private static string url;

        public string Url { get => throw new NotImplementedException(); set => throw new NotImplementedException(); }
        public static EventLog EventLog { get => eventLog; set => eventLog = value; }

        public static async Task Init(string[] args, EventLog el)
        {
            EventLog = el;
            url = args.Length >= 0 ? args[0] : "nats://nats-server:4222";
            // version = args.Length > 0 ? args[1] : "1";
            version = "1";

            Console.WriteLine($"args[0]: {args[0]}, args[1]: {args[1]}");
            Console.WriteLine($"url: {url}, version: {version}");

            // Parse url argument to see if publishing over diode directly, or over NATS
            if (url.StartsWith("nats"))
            {
                if (!ConnectNats(url)) return;
            }
            else if (url.StartsWith("mqtt"))
            {
                await ConnectMqtt(url);
            }
            else
            {
                if (!ConnectDiode(url)) return;
            }

            ReadGroups();
            ReadTags();

            SetTimer();
        }

        // groupid;groupname;samplingtime;
        private static List<SamplingGroup> ReadGroups()
        {
            groups = new List<SamplingGroup>();
            string csvData = File.ReadAllText("groups.csv");
            foreach (string row in csvData.Split('\n'))
            {
                if (!string.IsNullOrEmpty(row))
                {
                    if (row.StartsWith("groupid;")) continue;
                    var fields = row.Split(';');
                    var group = new SamplingGroup();
                    group.Tags = new List<string>();
                    group.Id = int.Parse(fields[0]); // 1 based numbering
                    group.Name = fields[1];
                    group.SamplingTime = int.Parse(fields[2]);
                    group.ProgId = fields[3];
                    groups.Add(group);

                    if (group.ProgId != "") progID = group.ProgId;
                }
            }

            if (progID == "")
            {
                LogEvent("No program identity specified (opc da server name)");
            }

            opcGroups = new List<OpcGroup>();
            return groups;
        }

        // tagname;groupid;
        private static void ReadTags()
        {
            string csvData = File.ReadAllText("tags.csv");
            foreach (string row in csvData.Split('\n'))
            {
                if (!string.IsNullOrEmpty(row))
                {
                    try
                    {
                        if (row.StartsWith("name;")) continue;
                        var fields = row.Split(';');
                        var groupid = int.Parse(fields[1]); // 1 based numbering
                        if (groupid <= 0 || groupid > groups.Count)
                        {
                            LogEvent($"Tag row item refers to group id out of range. Is 1 <= {groupid} <= {groups.Count}. Row ignored");
                            continue;
                        }

                        groups[groupid - 1].Tags.Add(fields[0]);
                    } catch (Exception ex) {
                        LogEvent($"Failed to read tag line: {row}, {ex.Message}");
                    }
                }
            }
        }

        private static bool ConnectDiode(string diodeIp)
        {
            diodeClient = new UdpClient();
            try
            {
                diodeClient.Connect(diodeIp, 4357);
                return true;
            }
            catch (Exception e)
            {
                LogEvent($"Failed to connect diode at {diodeIp}:4357, error: {e.Message}");
            }
            return false;
        }

        private static bool ConnectNats(string natsUrl)
        {
            try
            {
                ConnectionFactory cf = new ConnectionFactory();
                nc = cf.CreateConnection(natsUrl);
                return true;
            }
            catch (Exception e)
            {
                LogEvent($"Failed to connect nats at {natsUrl}, error: {e.Message}");
            }
            return false;
        }

        private static async Task ConnectMqtt(string mqttUrl)
        {
            var success = false;
            while (!success)
            {
                try
                {
                    var mqttFactory = new MQTTnet.MqttFactory();
                    mqttClient = mqttFactory.CreateMqttClient();

                    Console.WriteLine("The MQTT client is connecting ....");
                    var uri = new Uri(mqttUrl);
                    var mqttClientOptions = new MqttClientOptionsBuilder().WithTcpServer(uri.Host).WithProtocolVersion(MqttProtocolVersion.V500).Build();
                    var response = await mqttClient.ConnectAsync(mqttClientOptions, CancellationToken.None);
                    Console.WriteLine("The MQTT client is connected!");
                    success = true;
                }
                catch (Exception e)
                {
                    LogEvent($"Failed to connect MQTT at {mqttUrl}, error: {e.Message}. Trying to reconnect in 5 secs ...");
                    Thread.Sleep(5000);
                    success = false;
                }
            }
        }

        public static bool Startup()
        {
            opcServer = new OpcServer();
            try
            {
                LogEvent($"Connecting to OPC DA server with prog id: {progID}");
                opcServer.Connect(progID);
            }
            catch (Exception ex)
            {
                LogEvent($"Failed to connect OPC DA server: {progID}, error: {ex.Message}");
                Shutdown();
                return false;
            }


            System.Threading.Thread.Sleep(500); // we are faster than some servers!
            opcServer.ShutdownRequested += opcServer_ShutdownRequested;

            samplingGroups = new Dictionary<int, SamplingGroup>();

            foreach (var group in groups)
            {
                // LogEvent($"Adding group {group.Name} with samling time {group.SamplingTime}");
                OpcGroup opcGroup = opcServer.AddGroup($"opcda-win-group-{group.Id}", false, group.SamplingTime * 1000);
                opcGroups.Add(opcGroup);
                samplingGroups.Add(opcGroup.HandleClient, group);

                List<OpcItemDefinition> opcItemDefs = new List<OpcItemDefinition>();
                for (int i = 0; i < group.Tags.Count; i++)
                {
                    // LogEvent($"Adding tag {group.Tags[i]} to group {group.Name}");
                    opcItemDefs.Add(new OpcItemDefinition(group.Tags[i], true, i, VarEnum.VT_EMPTY));
                }

                var results = new OpcItemResult[opcItemDefs.Count];
                opcGroup.ValidateItems(opcItemDefs.ToArray(), true, out results);
                for (var i = results.Length - 1; i >= 0; i--)
                {
                    var result = results[i];
                    if (HRESULTS.Failed(result.Error))
                    {
                        LogEvent($"OpcItemResult error: {result.AccessRights}: {result.Error}, {group.Tags[i]}, removing tag from group!");
                        opcItemDefs.RemoveAt(i);
                    }
                }

                opcGroup.AddItems(opcItemDefs.ToArray(), out OpcItemResult[] opcItemResult);

                if (opcItemResult == null)
                {
                    LogEvent($"Error add items to group {group.Id} - null value returned by AddItems, no error code supplied! Group ignored");
                    continue;
                }

                opcGroup.DataChanged += OpcGroup_DataChanged;
                opcGroup.CancelCompleted += OpcGroup_CancelCompleted;

                opcGroup.SetEnable(true);
                opcGroup.Active = true;
            }

            return true;
        }

        public static void Shutdown()
        {
            foreach (var opcGroup in opcGroups)
            {
                try
                {
                    if (opcGroup != null)
                    {
                        opcGroup.DataChanged -= OpcGroup_DataChanged;
                        opcGroup.CancelCompleted -= OpcGroup_CancelCompleted;
                        opcGroup.Remove(true);
                    }
                }
                catch (Exception ex)
                {
                    LogEvent($"Exception caught when shutting down group:{ex.Message}");
                }
            }

            try
            {
                if (opcServer != null)
                {
                    opcServer.ShutdownRequested -= opcServer_ShutdownRequested;
                    opcServer.Disconnect();
                }
            }
            catch (Exception ex)
            {
                LogEvent($"Exception caught when shutting down server:{ex.Message}");
            }
        }

        private static async void OpcGroup_DataChanged(object sender, DataChangeEventArgs e)
        {
            var group = samplingGroups[e.GroupHandleClient];
            var datamessage = new DataMessage();
            datamessage.Version = 2;
            datamessage.Group = group.Name;
            datamessage.Interval = group.SamplingTime;
            datamessage.Sequence = 0;
            datamessage.Count = e.ItemStates.Length > 10 ? 10 : e.ItemStates.Length;
            datamessage.Points = new DataPoint[datamessage.Count];

            // LogEvent($"DataChanged Group:{e.GroupHandleClient} TrID:{e.TransactionID} E:{e.MasterError} Q:{e.MasterQuality}");
            var i = 0;
            var total = e.ItemStates.Length;
            foreach (OpcItemState s in e.ItemStates)
            {
                if (HRESULTS.Succeeded(s.Error))
                {
                    // LogEvent($" {s.HandleClient}: {s.DataValue} (Q:{s.Quality} T:{DateTime.FromFileTimeUtc(s.TimeStamp)})");
                    var point = new DataPoint();
                    point.Time = DateTime.FromFileTimeUtc(s.TimeStamp);
                    point.Name = group.Tags[s.HandleClient];
                    point.Value = Convert.ToDouble(s.DataValue);
                    point.Quality = s.Quality;

                    if (version == "2")
                    {
                        datamessage.Points[i++] = point;

                        if (i >= datamessage.Count)
                        {
                            var payload = JsonConvert.SerializeObject(datamessage);
                            byte[] bytes = Encoding.UTF8.GetBytes(payload);
                            if (nc != null)
                            {
                                nc.Publish("process.actual", bytes);
                            }
                            else if (mqttClient != null)
                            {
                                var msg = new MqttApplicationMessageBuilder()
                                    .WithTopic("process/actual")
                                    .WithPayload(bytes)
                                    .Build();
                                var result = await mqttClient.PublishAsync(msg, CancellationToken.None);
                                Console.WriteLine($"MQTT publish result: {result.ReasonString}");
                            } else { 
                                diodeClient.Send(bytes, bytes.Length);
                            }
                            total -= i;
                            i = 0;

                            datamessage.Count = total > 10 ? 10 : total;
                            if (datamessage.Count > 0) datamessage.Points = new DataPoint[datamessage.Count];
                        }
                    }
                    else if (nc != null)
                    {
                        var payload = JsonConvert.SerializeObject(point);
                        byte[] bytes = Encoding.UTF8.GetBytes(payload);
                        nc.Publish("process.actual", bytes);
                    }
                    else if (mqttClient != null && mqttClient.IsConnected)
                    {
                        var payload = JsonConvert.SerializeObject(point);
                        byte[] bytes = Encoding.UTF8.GetBytes(payload);
                        var msg = new MqttApplicationMessageBuilder().WithTopic("process/actual").WithPayload(bytes).Build();
                        try
                        {
                            var result = await mqttClient.PublishAsync(msg, CancellationToken.None);
                            // Console.WriteLine($"MQTT published message: {payload}");
                        } catch (MQTTnet.Exceptions.MqttClientNotConnectedException)
                        {
                            Console.WriteLine($"MQTT lost connection to server, reconnecting in 5 secs ...");
                            Thread.Sleep(5000);
                            await ConnectMqtt(url);
                        }
                    } else
                    {
                        Console.WriteLine($"MQTT lost connection to server, reconnecting in 5 secs ...");
                        Thread.Sleep(5000);
                        await ConnectMqtt(url);
                    }
                }
                else
                    LogEvent($" {s.HandleClient}: ERROR = 0x{s.Error:x} !");
            }
        }

        private static void OpcGroup_CancelCompleted(object sender, CancelCompleteEventArgs e)
        {
            LogEvent($"CancelCompleted Group:{e.GroupHandleClient} TrID:{e.TransactionID}");
        }

        private static void opcServer_ShutdownRequested(object sender, ShutdownRequestEventArgs e)
        {
            LogEvent($"ShutdownRequested: Reason:{e.ShutdownReason}");
        }

        private static void SetTimer()
        {
            // Create a timer with a two second interval.
            aTimer = new System.Timers.Timer(2000);
            // Hook up the Elapsed event for the timer. 
            aTimer.Elapsed += OnTimedEvent;
            aTimer.AutoReset = true;
            aTimer.Enabled = true;
        }

        private static void OnTimedEvent(Object source, ElapsedEventArgs e)
        {
            try
            {
                var s = opcServer.GetStatus();
                if (s.eServerState != OPCSERVERSTATE.OPC_STATUS_RUNNING)
                {
                    LogError($"Server doesn't seem to be running: {s.eServerState.ToString()}. Restarting ...");
                    Shutdown();
                    Startup();
                }
            }
            catch (Exception ex)
            {
                LogError($"Exception caught when trying to get server status: {ex.Message}. Restarting ...");
                Shutdown();
                Startup();
            }
        }

        public static void LogEvent(string message)
        {
            var now = DateTime.UtcNow;
            Console.WriteLine("{0}: {1}", now.ToString(), message);
            if (EventLog != null) EventLog.WriteEntry($"{message}");
        }

        public static void LogError(string message)
        {
            var now = DateTime.UtcNow;
            Console.WriteLine("{0}: {1}", now.ToString(), message);
            if (EventLog != null) EventLog.WriteEntry($"{message}", EventLogEntryType.Error);

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
