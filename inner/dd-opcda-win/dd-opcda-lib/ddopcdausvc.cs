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

namespace DdOpcDaLib
{
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

        // Responses
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

        public DdOpcDaUsvc(string name, string[] args) : base(name, args)
        {
            Name = name;
            loadGroups("groups.csv");
            loadTags("tags.csv");
            prepareOpc();

            this.Subscribe($"usvc.opc.{settings["instance-id"]}.tags.getall", this.getAllTags);
            this.Subscribe($"usvc.opc.{settings["instance-id"]}.groups.getall", this.getAllGroups);
            this.Subscribe($"usvc.opc.{settings["instance-id"]}.groups.start", this.startGroup);
            this.Subscribe($"usvc.opc.{settings["instance-id"]}.groups.stop", this.stopGroup);
            this.Subscribe($"usvc.opc.{settings["instance-id"]}.servers.getall", this.getAllServers);
        }

        internal OpcServer connectServer(string progid)
        {
            if (_opcServers.ContainsKey(progid)) return _opcServers[progid];

            var opcServer = new OpcServer();
            try
            {
                DdOpcDa.LogEvent($"Connecting to OPC DA server with prog id: {progid}");
                opcServer.Connect(progid);
            }
            catch (Exception ex)
            {
                DdOpcDa.LogEvent($"Failed to connect OPC DA server: {progid}, error: {ex.Message}");
                return null;
            }

            return opcServer;
        }

        public void Startup()
        {
            foreach (var group in _groups)
            {
                try
                {
                    group.opcGroup.DataChanged += OpcGroup_DataChanged;
                    group.opcGroup.CancelCompleted += OpcGroup_CancelCompleted;
                    group.opcGroup.SetEnable(true);
                    if (group.RunAtStart)
                    {
                        group.opcGroup.Active = true;
                        group.State = OpcGroupState.GroupStateRunning;
                        DdOpcDa.LogEvent($"Starting group: {group.Name}");
                    }
                }
                catch (Exception ex)
                {
                    DdOpcDa.LogEvent($"Exception caught when starting up group: {ex.Message}");
                }
            }
        }

        public void Shutdown()
        {
            foreach (var group in _groups)
            {
                try
                {
                    var opcGroup = group.opcGroup;
                    if (opcGroup != null)
                    {
                        opcGroup.DataChanged -= OpcGroup_DataChanged;
                        opcGroup.CancelCompleted -= OpcGroup_CancelCompleted;
                        opcGroup.Remove(true);
                        group.opcGroup = null;
                    }
                }
                catch (Exception ex)
                {
                    DdOpcDa.LogError($"Exception caught when shutting down group:{ex.Message}");
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
                        }
                    }
                }
                catch (Exception ex)
                {
                    DdOpcDa.LogEvent($"Exception caught when shutting down server:{ex.Message}");
                }
            }
        }

        internal void loadGroups(string filename)
        {
            try
            {
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

                        var opcServer = connectServer(group.ProgID);
                        group.opcGroup = opcServer.AddGroup($"dd-opcda-group-{group.Id}", false, group.Interval * 1000);
                        group.opcItemDefinitions = new List<OpcItemDefinition>();
                        group.invalidItemDefinitions = new List<OpcItemDefinition>();

                        _groups.Add(group);
                    }
                }
            }
            catch (Exception e)
            {
                DdOpcDa.LogEvent($"Failed to load groups: {e.Message}");
                Console.WriteLine(e.ToString());
            }
        }

        // Reads all tags in a CSV file and stores them in a ordered list (_tags) for reference and OPC item definitions
        // list in the associated sampling group. Items in the group are validated later and may be remove from the
        // definitions list while the ordered list keep them to maintain the order integrity
        internal void loadTags(string filename)
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
                            var groupid = int.Parse(fields[1]); // 1 based numbering
                            if (groupid < 1 || groupid > _groups.Count)
                            {
                                DdOpcDa.LogEvent($"Tag row item refers to group id out of range (1 based). Is 1 <= {groupid} <= {_groups.Count}. Row ignored");
                                continue;
                            }

                            var group = _groups[groupid - 1]; // 1 based numbering in 0 based array
                            var tag = new OpcTagItem() { Id = tagid++, GroupID = groupid, Name = fields[0] };
                            _tags.Add(tag);
                            group.tags.Add(tag);
                            group.opcItemDefinitions.Add(new OpcItemDefinition(tag.Name, true, tag.Id, VarEnum.VT_EMPTY));
                        }
                        catch (Exception ex)
                        {
                            DdOpcDa.LogEvent($"Failed to read tag line: {row}, {ex.Message}");
                            Console.WriteLine(ex.ToString());
                        }
                    }
                }
            }
            catch (Exception e)
            {
                DdOpcDa.LogEvent($"Failed to load tags: {e.Message}");
            }
        }

        internal void prepareOpc()
        {
            // Initialize all groups
            foreach (var group in _groups)
            {
                if (group.opcItemDefinitions.Count <= 0) continue;
                var results = new OpcItemResult[group.opcItemDefinitions.Count];
                group.opcGroup.ValidateItems(group.opcItemDefinitions.ToArray(), true, out results);
                for (var i = results.Length - 1; i >= 0; i--)
                {
                    var result = results[i];
                    if (HRESULTS.Failed(result.Error))
                    {
                        DdOpcDa.LogEvent($"OpcItemResult error: {result.AccessRights}: {result.Error}, {group.tags[i].Name}, i: {i}, removing tag from group {group.Name}!");
                        group.invalidItemDefinitions.Add(group.opcItemDefinitions[i]);
                        group.opcItemDefinitions.RemoveAt(i);
                    }
                }

                group.opcGroup.AddItems(group.opcItemDefinitions.ToArray(), out OpcItemResult[] opcItemResult);

                //group.opcGroup.DataChanged += OpcGroup_DataChanged;
                //group.opcGroup.CancelCompleted += OpcGroup_CancelCompleted;
                //group.opcGroup.SetEnable(true);
                //if (group.RunAtStart)
                //{
                //    group.opcGroup.Active = true;
                //    group.State = OpcGroupState.GroupStateRunning;
                //}
            }
        }

        private void OpcGroup_CancelCompleted(object sender, CancelCompleteEventArgs e)
        {
            DdOpcDa.LogEvent($"CancelCompleted Group:{e.GroupHandleClient} TrID:{e.TransactionID}");
        }

        private void opcServer_ShutdownRequested(object sender, ShutdownRequestEventArgs e)
        {
            DdOpcDa.LogEvent($"ShutdownRequested: Reason:{e.ShutdownReason}");
        }


        // TODO: make this thread safe and handle exceptions gracefully
        private void OpcGroup_DataChanged(object sender, DataChangeEventArgs e)
        {
            try
            {
                var total = e.ItemStates.Length;
                foreach (OpcItemState s in e.ItemStates)
                {
                    if (HRESULTS.Succeeded(s.Error))
                    {
                        if (s.HandleClient > 0 && s.HandleClient < _tags.Count)
                        {
                            var point = new DataPoint();
                            point.Time = DateTime.FromFileTimeUtc(s.TimeStamp);
                            point.Name = _tags[s.HandleClient].Name; // HandleClient set to tag.Id (0 based, matching the array)
                            point.Value = Convert.ToDouble(s.DataValue);
                            point.Quality = s.Quality;
                            var payload = JsonConvert.SerializeObject(point);
                            byte[] bytes = Encoding.UTF8.GetBytes(payload);
                            broker.Publish("process.actual", bytes);
                        }
                        else
                        {
                            DdOpcDa.LogError($"Error while processing data changed event, client handle index not in tag list: {s.HandleClient} ({_tags.Count})");
                        }
                    }
                }
            }
            catch (Exception ex)
            {
                DdOpcDa.LogError($"Exception while processing data changed event: {ex.ToString()}");
            }
        }

        internal DdUsvcError getAllTags(string topic, string responsetopic, byte[] data)
        {
            try
            {
                var response = new OpcTagItemResponse();
                response.Items = _tags.ToArray();
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

        internal DdUsvcError getAllGroups(string topic, string responsetopic, byte[] data)
        {
            try
            {
                var response = new OpcGroupItemResponse();
                response.Items = _groups.ToArray();
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

        internal DdUsvcError startGroup(string topic, string responsetopic, byte[] data)
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
                    else
                    {
                        group.opcGroup.Active = true;
                        group.State = OpcGroupState.GroupStateRunning;
                        response.Success = true;
                        DdOpcDa.LogEvent($"Group started: {group.Id}, state: {group.State}");
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

        internal DdUsvcError stopGroup(string topic, string responsetopic, byte[] data)
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
                    else
                    {
                        group.opcGroup.Active = false;
                        group.State = OpcGroupState.GroupStateStopped;
                        response.Success = true;
                        DdOpcDa.LogEvent($"Group stopped: {group.Id}, state: {group.State}");
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

        internal DdUsvcError getAllServers(string topic, string responsetopic, byte[] data)
        {
            try
            {
                var servers = OpcServerList.ListAll(OpcServerList.OpcDataAccess20);
                if (servers != null)
                {
                    Console.WriteLine($"Number of OPCDA20 servers found: {servers.Length}");
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
    }
}
