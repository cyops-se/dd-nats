using DdUsvc;
using Newtonsoft.Json;
using System;
using OPC.Common;
using OPC.Data;
using static DdOpcDaLib.DdOpcDaUsvc;
using System.Collections.Generic;
using System.IO;
using System.Text;
using System.Runtime.InteropServices;
using System.Security.Cryptography;
using System.Linq;
using System.Diagnostics;
using System.Xml.Linq;
using System.Threading;

namespace DdOpcDaLib
{
    public struct ServerStatus
    {
        [JsonProperty("progid")]
        public string ProgId { get; set; }
        [JsonProperty("currenttime")]
        public DateTime CurrentTime { get; set; }
        [JsonProperty("starttme")]
        public DateTime StartTime { get; set; }
        [JsonProperty("lastupdate")]
        public DateTime LastUpdate { get; set; }
        [JsonProperty("state")]
        public OPCSERVERSTATE State { get; set; }
        [JsonProperty("error")]
        public string Error { get; set; }
        [JsonProperty("bandwidth")]
        public int BandWidth { get; set; }
        [JsonProperty("groupcount")]
        public int GroupCount { get; set; }
        [JsonProperty("host")]
        public string HostName { get; set; }
        [JsonProperty("instance")]
        public string Instance { get; set; }
    }

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
        [JsonProperty("i")]
        public string Instance { get; set; }
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

    public class DdOpcDaUsvc : DdUsvc.DdUsvc
    {
        // Types
        public class IntMessage
        {
            [JsonProperty("value")]
            public int Value { get; set; }
        }

        public enum OpcGroupState
        {
            GroupStateUnknown = 0,
            GroupStateStopped = 1,
            GroupStateRunning = 2,
            GroupStateRunningWithWarning = 3,
            GroupStateDisabled = 4,
        }

        public class OpcTagItem
        {
            [JsonProperty("id")]
            public int Id { get; set; }
            [JsonProperty("name")]
            public string Name { get; set; }
            [JsonProperty("group")]
            public OpcGroupItem Group { get; set; }
            [JsonProperty("groupid")]
            public int GroupID { get; set; }
            [JsonProperty("time")]
            public DateTime Time { get; set; }
            [JsonProperty("value")]
            public double Value { get; set; }
            [JsonProperty("quality")]
            public int Quality { get; set; }
            [JsonProperty("instance")]
            public string Instance { get; set; }
            [JsonProperty("Error")]
            public int Error { get; set; }
        }

        public class OpcGroupItem
        {
            [JsonProperty("id")]
            public int Id { get; set; }
            [JsonProperty("name")]
            public string Name { get; set; }
            [JsonProperty("progid")]
            public string ProgID { get; set; }
            [JsonProperty("interval")]
            public int Interval { get; set; }
            [JsonProperty("runatstart")]
            public bool RunAtStart { get; set; }
            [JsonProperty("defaultgroup")]
            public bool DefaultGroup { get; set; }
            [JsonProperty("state")]
            public OpcGroupState State { get; set; }
            [JsonProperty("lastrun")]
            public DateTime LastRun { get; set; }
            [JsonIgnore]
            public List<OpcTagItem> tags { get; set; }
            [JsonIgnore]
            public OpcGroup opcGroup { get; set; }
            [JsonIgnore]
            public List<OpcItemDefinition> opcItemDefinitions { get; set; }
            [JsonIgnore]
            public List<OpcItemDefinition> invalidItemDefinitions { get; set; }
        }

        public class Tag
        {
            [JsonProperty("tag")]
            public string Item { get; set; }
        }

        public class Tags
        {
            [JsonProperty("tags")]
            public Tag[] Items { get; set; }
        }

        public class OpcItems
        {
            [JsonProperty("items")]
            public OpcTagItem[] Items { get; set; }
        }

        public class Groups
        {
            [JsonProperty("items")]
            public OpcGroupItem[] Items { get; set; }
        }

        public class OpcServerItem
        {
            [JsonProperty("id")]
            public int ID { get; set; }
            [JsonProperty("progid")]
            public string ProgID { get; set; }
            [JsonProperty("name")]
            public string Name { get; set; }
        }

        public class OpcServers
        {
            [JsonProperty("items")]
            public OpcServerItem[] Items { get; set; }
        }


        // Requests
        public class GetOPCBranches
        {
            [JsonProperty("sid")]
            public int ServerId { get; set; }

            [JsonProperty("branch")]
            public string Branch {  get; set; }
        }


        // Responses
        public class BrowserPosition : StatusResponse
        {
            [JsonProperty("sid")]
            public int ServerId { get; set; }

            [JsonProperty("position")]
            public string Position { get; set; }

            [JsonProperty("branches")]
            public string[] Branches { get; set; }

            [JsonProperty("leaves")]
            public string[] Leaves { get; set; }
}

        public class OpcTagItemResponse : StatusResponse
        {
            [JsonProperty("items")]
            public OpcTagItem[] Items { get; set; }
        }

        public class OpcGroupItemResponse : StatusResponse
        {
            [JsonProperty("items")]
            public OpcGroupItem[] Items { get; set; }
        }

        public class OpcServerItemResponse : StatusResponse
        {
            [JsonProperty("items")]
            public OpcServerItem[] Items { get; set; }
        }

        protected Dictionary<string, OpcServer> _opcServers = new Dictionary<string, OpcServer>();
        protected List<OpcGroupItem> _groups;
        protected List<OpcTagItem> _tags;
        private System.Timers.Timer aTimer;
        private string _instance;

        public DdOpcDaUsvc(string name, string[] args) : base(name, args)
        {
            Name = name;
            SetTimer();
            _instance = settings["instance-id"];

            this.Subscribe($"usvc.opc.{_instance}.tags.getall", this.GetAllTags);
            this.Subscribe($"usvc.opc.{_instance}.groups.getall", this.GetAllGroups);
            this.Subscribe($"usvc.opc.{_instance}.groups.start", this.StartGroup);
            this.Subscribe($"usvc.opc.{_instance}.groups.stop", this.StopGroup);
            this.Subscribe($"usvc.opc.{_instance}.servers.getall", this.GetAllServers);
            this.Subscribe($"usvc.opc.{_instance}.servers.root", this.GetOpcServerRoot);
            this.Subscribe($"usvc.opc.{_instance}.servers.getbranch", this.GetOpcServerBranch);
        }

        internal OpcServer ConnectServer(string progid)
        {
            if (_opcServers.ContainsKey(progid)) return _opcServers[progid];

            // Try every 15 secs for a minute before giving up
            for (var i = 0; i < 4; i++)
            {
                var opcServer = new OpcServer();
                try
                {
                    LogEvent($"Connecting to OPC DA server with prog id: {progid}");
                    opcServer.Connect(progid);
                    _opcServers[progid] = opcServer;
                    return opcServer;
                }
                catch (Exception ex)
                {
                    LogEvent($"Failed to connect OPC DA server: {progid}, error: {ex.Message}. Retry attempts left: {4-i}");
                    Thread.Sleep(15000);
                }
            }

            return null;
        }

        public void Initialize()
        {
            LoadGroups("groups.csv");
            LoadTags("tags.csv");
            PrepareOpc();
        }

        public void Restart()
        {
            Shutdown();
            Initialize();
            Startup();
        }

        public void Startup()
        {
            foreach (var group in _groups)
            {
                if (group == null || group.State == OpcGroupState.GroupStateDisabled) continue;
                try
                {
                    group.opcGroup.DataChanged += OpcGroup_DataChanged;
                    group.opcGroup.CancelCompleted += OpcGroup_CancelCompleted;
                    group.opcGroup.SetEnable(true);
                    if (group.RunAtStart)
                    {
                        group.opcGroup.Active = true;
                        group.State = OpcGroupState.GroupStateRunning;
                        LogEvent($"Starting group: {group.Name}");
                    }
                }
                catch (Exception ex)
                {
                    LogEvent($"Exception caught when starting up group: {ex.Message}");
                }
            }
        }

        public void Shutdown()
        {
            foreach (var group in _groups)
            {
                if (group == null || group.State == OpcGroupState.GroupStateDisabled) continue;
                try
                {
                    var opcGroup = group.opcGroup;
                    if (opcGroup != null && group.State != OpcGroupState.GroupStateDisabled)
                    {
                        opcGroup.DataChanged -= OpcGroup_DataChanged;
                        opcGroup.CancelCompleted -= OpcGroup_CancelCompleted;
                        opcGroup.Remove(true);
                        group.opcGroup = null;
                        group.State = OpcGroupState.GroupStateStopped;
                        LogEvent($"Group stopped due to shutdown request: {group.Name}, {group.Id}");
                    } 
                }
                catch (Exception ex)
                {
                    LogError($"Exception caught when shutting down group:{ex.Message}");
                }

                try
                {
                    if (_opcServers.ContainsKey(group.ProgID))
                    {
                        var opcServer = _opcServers[group.ProgID];
                        if (opcServer != null)
                        {
                            opcServer.ShutdownRequested -= opcServer_ShutdownRequested;
                            opcServer.Disconnect();
                            _opcServers.Remove(group.ProgID);
                            LogEvent($"OPCDA server stopped due to shutdown request: {group.ProgID}, {group.Id}");
                        }
                    }
                }
                catch (Exception ex)
                {
                    LogError($"Exception caught when shutting down server:{ex.Message}");
                }
            }

            _opcServers = null;
        }

        internal void LoadGroups(string filename)
        {
            try
            {
                _opcServers = new Dictionary<string, OpcServer>();
                _groups = new List<OpcGroupItem>();
                string csvData = File.ReadAllText("groups.csv");
                foreach (string row in csvData.Split('\n'))
                {
                    if (!string.IsNullOrEmpty(row))
                    {
                        if (row.StartsWith("groupid;")) continue;
                        var fields = row.Split(';');
                        var group = new OpcGroupItem();
                        group.Id = int.Parse(fields[0]); // 1 based numbering
                        group.Name = fields[1];
                        group.Interval = int.Parse(fields[2]);
                        group.ProgID = fields[3];
                        group.DefaultGroup = int.Parse(fields[4]) == 1 ? true : false;
                        group.RunAtStart = int.Parse(fields[5]) == 1 ? true : false;
                        group.State = OpcGroupState.GroupStateStopped;
                        group.tags = new List<OpcTagItem>();

                        var opcServer = ConnectServer(group.ProgID);
                        if (opcServer != null)
                        {
                            group.opcGroup = opcServer.AddGroup($"dd-opcda-group-{group.Id}", false, group.Interval * 1000);
                            group.opcGroup.PercentDeadband = 0.0001f;
                            group.opcGroup.RefreshState();
                            Console.WriteLine($"Group id {group.Id} states, percentdeadband: {group.opcGroup.PercentDeadband}, interval: {group.opcGroup.UpdateRate}");
                            group.opcItemDefinitions = new List<OpcItemDefinition>();
                            group.invalidItemDefinitions = new List<OpcItemDefinition>();
                        } else
                        {
                            group.State = OpcGroupState.GroupStateDisabled;
                            LogError($"Failed to connect to OPCDA server: {group.ProgID}, for group: {group.Name}. Ignored!");
                        }

                        _groups.Add(group);
                    }
                }
            }
            catch (Exception e)
            {
                LogEvent($"Failed to load groups: {e.Message}");
                Console.WriteLine(e.ToString());
            }
        }

        // Reads all tags in a CSV file and stores them in a ordered list (_tags) for reference and OPC item definitions
        // list in the associated sampling group. Items in the group are validated later and may be remove from the
        // definitions list while the ordered list keep them to maintain the order integrity
        internal void LoadTags(string filename)
        {
            try
            {
                int tagid = 0; // 0 based indexing
                _tags = new List<OpcTagItem>();
                string csvData = File.ReadAllText("tags.csv");
                foreach (string row in csvData.Split('\n'))
                {
                    if (!string.IsNullOrEmpty(row))
                    {
                        try
                        {
                            if (row.StartsWith("name;")) continue;
                            var fields = row.Split(';');
                            var groupid = int.Parse(fields[1]); // 1 based numbering of group id
                            if (groupid < 1 || groupid > _groups.Count)
                            {
                                LogEvent($"Tag row item refers to group id out of range (1 based). {groupid} < 1 || {groupid} > {_groups.Count}.");
                                LogEvent($"Row ignored: {row}");
                                continue;
                            }

                            var group = _groups[groupid - 1]; // 1 based numbering in 0 based array
                            if (group != null && group.State != OpcGroupState.GroupStateDisabled)
                            {
                                var tag = new OpcTagItem() { Id = tagid++, GroupID = groupid, Group = group, Name = fields[0] };
                                _tags.Add(tag);
                                group.tags.Add(tag);
                                group.opcItemDefinitions.Add(new OpcItemDefinition(tag.Name, true, tag.Id, VarEnum.VT_EMPTY));
                            }
                        }
                        catch (Exception ex)
                        {
                            LogEvent($"Failed to read tag line: {row}, {ex.Message}");
                            Console.WriteLine(ex.ToString());
                        }
                    }
                }
            }
            catch (Exception e)
            {
                LogEvent($"Failed to load tags: {e.Message}");
            }
        }

        internal void PrepareOpc()
        {
            // Initialize all groups
            foreach (var group in _groups)
            {
                if (group == null || group.State == OpcGroupState.GroupStateDisabled) continue;
                if (group.opcItemDefinitions.Count <= 0) continue;
                var results = new OpcItemResult[group.opcItemDefinitions.Count];
                group.opcGroup.ValidateItems(group.opcItemDefinitions.ToArray(), true, out results);
                for (var i = results.Length - 1; i >= 0; i--)
                {
                    var result = results[i];
                    if (HRESULTS.Failed(result.Error))
                    {
                        LogEvent($"OpcItemResult error: {result.AccessRights}: {result.Error}, {group.tags[i].Name}, i: {i}, removing tag from group {group.Name}!");
                        group.invalidItemDefinitions.Add(group.opcItemDefinitions[i]);
                        group.opcItemDefinitions.RemoveAt(i);
                    }
                }

                group.opcGroup.AddItems(group.opcItemDefinitions.ToArray(), out OpcItemResult[] opcItemResult);
            }
        }

        private void OpcGroup_CancelCompleted(object sender, CancelCompleteEventArgs e)
        {
            LogEvent($"CancelCompleted Group:{e.GroupHandleClient} TrID:{e.TransactionID}");
        }

        private void opcServer_ShutdownRequested(object sender, ShutdownRequestEventArgs e)
        {
            LogEvent($"ShutdownRequested: Reason:{e.ShutdownReason}");
            Shutdown();
        }


        // TODO: make this thread safe and handle exceptions gracefully
        private void OpcGroup_DataChanged(object sender, DataChangeEventArgs e)
        {
            try
            {
                var total = e.ItemStates.Length;
                foreach (OpcItemState s in e.ItemStates)
                {
                    if (s.HandleClient >= 0 && s.HandleClient < _tags.Count)
                    {
                        var tag = _tags[s.HandleClient]; // HandleClient set to tag.Id (0 based, matching the array)
                        tag.Error = s.Error;
                        if (HRESULTS.Succeeded(s.Error))
                        {
                            var point = new DataPoint();
                            point.Time = DateTime.FromFileTimeUtc(s.TimeStamp);
                            point.Name = tag.Name;
                            point.Value = Convert.ToDouble(s.DataValue);
                            point.Quality = s.Quality;
                            point.Instance = _instance;
                            var payload = JsonConvert.SerializeObject(point);
                            byte[] bytes = Encoding.UTF8.GetBytes(payload);
                            broker.Publish("process.actual", bytes);

                            tag.Time = point.Time;
                            tag.Value = point.Value;
                            tag.Quality = point.Quality;
                            tag.Instance = point.Instance;
                        }
                        else
                        {
                            LogError($"Error while processing data changed event, tag: {tag.Name}, returned error: {s.Error}");
                        }
                    }
                    else
                    {
                        LogError($"Error while processing data changed event, client handle index not in tag list: {s.HandleClient} ({_tags.Count})");
                    }
                }
            }
            catch (Exception ex)
            {
                LogError($"Exception while processing data changed event: {ex.ToString()}");
            }
        }

        private void SetTimer()
        {
            // Periodically send the last value every 10s for items not reporting new data
            // and check server status
            aTimer = new System.Timers.Timer(1000);
            aTimer.Elapsed += ATimer_Elapsed; ;
            aTimer.AutoReset = true;
            aTimer.Enabled = true;
        }

        private void ATimer_Elapsed(object sender, System.Timers.ElapsedEventArgs e)
        {
            foreach (OpcTagItem tag in _tags)
            {
                // Only process error free tags in an active group
                if (tag.Group.opcGroup != null && tag.Group.opcGroup.Active && HRESULTS.Succeeded(tag.Error))
                {
                    var now = DateTime.UtcNow;
                    var diff = now.Subtract(tag.Time);
                    // LogEvent($"diff: {diff.TotalSeconds}, trigger: {diff > TimeSpan.FromSeconds(10)}");
                    if (diff > TimeSpan.FromSeconds(10))
                    {
                        var point = new DataPoint();
                        point.Time = now;
                        point.Name = tag.Name;
                        point.Value = Convert.ToDouble(tag.Value);
                        point.Quality = 68; // Uncertain [Last usable] tag.Quality;
                        point.Instance = _instance;
                        var payload = JsonConvert.SerializeObject(point);
                        byte[] bytes = Encoding.UTF8.GetBytes(payload);
                        broker.Publish("process.actual", bytes);
                        tag.Time = now;
                    }
                }
            }

            try
            {
                foreach (var kv in _opcServers)
                {
                    var opcServer = (OpcServer)kv.Value;
                    if (opcServer != null)
                    {
                        var status = opcServer.GetStatus();
                        var msg = new ServerStatus();
                        msg.ProgId = kv.Key;
                        msg.State = status.eServerState;
                        msg.LastUpdate = new DateTime(status.ftLastUpdateTime);
                        msg.CurrentTime = new DateTime(status.ftCurrentTime);
                        msg.StartTime = new DateTime(status.ftStartTime);
                        msg.BandWidth = status.dwBandWidth;
                        msg.GroupCount = status.dwGroupCount;
                        msg.Instance = _instance;
                        msg.HostName = System.Environment.MachineName;

                        var payload = JsonConvert.SerializeObject(msg);
                        byte[] bytes = Encoding.UTF8.GetBytes(payload);
                        broker.Publish("process.opc.server.status", bytes);
                    }
                    else
                    {
                        LogEvent($"OPCDA server instance is NULL (unexpectedly)");
                    }
                }
            }
            catch (Exception ex)
            {
                LogError($"Exception caught when checking OPCDA servers: {ex.Message}");
                Restart();
            }
        }

        internal DdUsvcError GetAllTags(string topic, string responsetopic, byte[] data)
        {
            var response = new OpcTagItemResponse();
            response.Success = true;
            try
            {
                response.Items = _tags.ToArray();

                var err = this.Publish(responsetopic, response);
                if (err.Code == DdUsvcErrorCode.Error)
                {
                    LogError($"tags.getall responding FAILED ... {responsetopic}, err: {err.Reason}");
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine(ex.Message);
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;

                response.Success = false;
                response.StatusMessage = ex.Message;

                var err = this.Publish(responsetopic, response);
                if (err.Code == DdUsvcErrorCode.Error) {
                    LogError($"tags.getall responding FAILED ... {responsetopic}, err: {err.Reason}, ex: {response.StatusMessage}");
                }
            }

            return this.lasterror;
        }

        internal DdUsvcError GetAllGroups(string topic, string responsetopic, byte[] data)
        {
            var response = new OpcGroupItemResponse();
            response.Success = true;
            try
            {
                response.Items = _groups.ToArray();

                this.Publish(responsetopic, response);
            }
            catch (Exception ex)
            {
                Console.WriteLine(ex.Message);
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;

                response.Success = false;
                response.StatusMessage = ex.Message;

                LogError($"groups.getall responding to ... {responsetopic}, ex: {response.StatusMessage}");
                this.Publish(responsetopic, response);
            }

            return this.lasterror;
        }

        internal DdUsvcError StartGroup(string topic, string responsetopic, byte[] data)
        {
            var response = new StatusResponse();
            try
            {
                var request = JsonConvert.DeserializeObject<IntMessage>(Encoding.UTF8.GetString(data));
                if (request.Value >= 1 || request.Value <= _groups.Count)
                {
                    var group = _groups[request.Value - 1];
                    if (group.State == OpcGroupState.GroupStateRunning || group.State == OpcGroupState.GroupStateRunningWithWarning)
                    {
                        response.StatusMessage = $"Group already running, group: {group.Name} (id: {group.Id})";
                    }
                    else if (group.State != OpcGroupState.GroupStateDisabled)
                    {
                        group.opcGroup.Active = true;
                        group.State = OpcGroupState.GroupStateRunning;
                        response.Success = true;
                        LogEvent($"Group started: {group.Id}, state: {group.State}");
                    }
                }

                this.Publish(responsetopic, response);
            }
            catch (Exception ex)
            {
                Console.WriteLine(ex.Message);
                response.Success = false;
                response.StatusMessage = ex.Message;

                this.Publish(responsetopic, response);

                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;
            }

            return this.lasterror;
        }

        internal DdUsvcError StopGroup(string topic, string responsetopic, byte[] data)
        {
            var response = new StatusResponse();
            try
            {
                var request = JsonConvert.DeserializeObject<IntMessage>(Encoding.UTF8.GetString(data));
                if (request.Value >= 1 || request.Value <= _groups.Count)
                {
                    var group = _groups[request.Value - 1];
                    if (group.State == OpcGroupState.GroupStateStopped)
                    {
                        response.StatusMessage = $"Group already stopped, group: {group.Name} (id: {group.Id})";
                    }
                    else if (group.State != OpcGroupState.GroupStateDisabled)
                    {
                        group.opcGroup.Active = false;
                        group.State = OpcGroupState.GroupStateStopped;
                        response.Success = true;
                        LogEvent($"Group stopped: {group.Id}, state: {group.State}");
                    }
                }

                this.Publish(responsetopic, response);
            }
            catch (Exception ex)
            {
                Console.WriteLine(ex.Message);
                response.Success = false;
                response.StatusMessage = ex.Message;

                this.Publish(responsetopic, response);

                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;
            }

            return this.lasterror;
        }

        internal DdUsvcError GetAllServers(string topic, string responsetopic, byte[] data)
        {
            try
            {
                var servers = OpcServerList.ListAll(OpcServerList.OpcDataAccess20);
                if (servers != null)
                {
                    var response = new OpcServerItemResponse();
                    response.Items = new OpcServerItem[servers.Length];
                    response.Success = true;

                    for (int i = 0; i < servers.Length; i++)
                    {
                        response.Items[i] = new OpcServerItem();
                        response.Items[i].ID = i;
                        response.Items[i].Name = servers[i].Name;
                        response.Items[i].ProgID = servers[i].ProgID;
                    }

                    this.Publish(responsetopic, response.Items);
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine(ex.ToString());
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;
            }

            return this.lasterror;
        }
        private DdUsvcError GetOpcServerRoot(string topic, string responsetopic, byte[] data)
        {
            var response = new BrowserPosition();
            try
            {
                var request = JsonConvert.DeserializeObject<IntMessage>(Encoding.UTF8.GetString(data));
                var sid = request.Value;

                var servers = OpcServerList.ListAll(OpcServerList.OpcDataAccess20);
                var server = servers[sid];
                var opcServer = ConnectServer(server.ProgID);
                opcServer.ChangeBrowsePosition(OPCBROWSEDIRECTION.OPC_BROWSE_TO, string.Empty);
                response.Branches = opcServer.BrowseItemIDs(OPCBROWSETYPE.OPC_BRANCH);
                response.Leaves = opcServer.BrowseItemIDs(OPCBROWSETYPE.OPC_LEAF);
                response.ServerId = sid;
                this.Publish(responsetopic, response);
            }
            catch (Exception ex)
            {
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;
                response.Success = false;
                response.StatusMessage = $"GetOpcServerRoot failed {ex.ToString()}";

                LogError(response.StatusMessage);
                this.Publish(responsetopic, response);
            }

            return this.lasterror;
        }

        private DdUsvcError GetOpcServerBranch(string topic, string responsetopic, byte[] data)
        {
            var response = new BrowserPosition();
            try
            {
                var request = JsonConvert.DeserializeObject<GetOPCBranches>(Encoding.UTF8.GetString(data));
                var servers = OpcServerList.ListAll(OpcServerList.OpcDataAccess20);
                var server = servers[request.ServerId];
                var opcServer = ConnectServer(server.ProgID);
                opcServer.ChangeBrowsePosition(OPCBROWSEDIRECTION.OPC_BROWSE_TO, request.Branch);
                response.Branches = opcServer.BrowseItemIDs(OPCBROWSETYPE.OPC_BRANCH);
                response.Leaves = opcServer.BrowseItemIDs(OPCBROWSETYPE.OPC_LEAF);
                response.ServerId = request.ServerId;
                this.Publish(responsetopic, response);
            }
            catch (Exception ex)
            {
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;
                response.Success = false;
                response.StatusMessage = $"GetOpcServerBranch failed {ex.ToString()}";

                LogError(response.StatusMessage);
                this.Publish(responsetopic, response);
            }

            return this.lasterror;
        }
    }
}
