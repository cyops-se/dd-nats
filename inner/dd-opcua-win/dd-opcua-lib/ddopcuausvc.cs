using dd_opcua_lib;
using DdUsvc;
using Newtonsoft.Json;
using Opc.Ua;
using Opc.Ua.Client;
using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;
using CsvHelper;
using CsvHelper.Configuration;
using System.Globalization;
using System.Timers;

namespace DdOpcUaLib
{
    public class DdOpcUaUsvc : DdUsvc.DdUsvc
    {
        protected Dictionary<string, OpcUaConnection> _opcServers = new Dictionary<string, OpcUaConnection>();
        protected List<OpcGroupItem> _groups;
        protected List<OpcTagItem> _tags;
        private Timer aTimer;
        private Timer metaTimer;
        private string _instance;
        protected OpcUaSubscription _subscription;
        Dictionary<int, string> _serverIds;

        public DdOpcUaUsvc(string name, string[] args) : base(name, args)
        {
            Name = name;
            SetTimer();
            _instance = settings["instance-id"];
            this.Subscribe($"usvc.opc.{_instance}.tags.getallunfiltered", this.GetAllTagsUnfiltered);
            this.Subscribe($"usvc.opc.{_instance}.tags.getall", this.GetAllTags);
            this.Subscribe($"usvc.opc.{_instance}.groups.getall", this.GetAllGroups);
            this.Subscribe($"usvc.opc.{_instance}.groups.start", this.StartGroup);
            this.Subscribe($"usvc.opc.{_instance}.groups.stop", this.StopGroup);
            this.Subscribe($"usvc.opc.{_instance}.servers.getall", this.GetAllServers);
            this.Subscribe($"usvc.opc.{_instance}.servers.root", this.GetOpcServerRoot);
            this.Subscribe($"usvc.opc.{_instance}.servers.getbranch", this.GetOpcServerBranch);
            this.Subscribe($"usvc.opc.{_instance}.groups.ha", this.GetGroupHistoricalData);
            this.Subscribe($"usvc.opc.{_instance}.tags.ha", this.GetTagHistoricalData);
        }

        private DdUsvcError GetTagHistoricalData(string topic, string responsetopic, byte[] data)
        {
            var response = new TagHistoryResponse
            {
                Success = false,
                StatusMessage = "Uninitialized"
            };
            try
            {
                if (data == null || data.Length == 0)
                    throw new ArgumentException("Empty request payload.");

                TagHistoryRequest request;
                try
                {
                    request = JsonConvert.DeserializeObject<TagHistoryRequest>(Encoding.UTF8.GetString(data));
                }
                catch (Exception dx)
                {
                    throw new ArgumentException("Failed to parse request: " + dx.Message);
                }

                if (request == null)
                    throw new ArgumentException("Null request object.");

                if (string.IsNullOrWhiteSpace(request.TagName))
                    throw new ArgumentException("TagName must be provided.");

                if (request.Start >= request.End)
                    throw new ArgumentException("Start time must be before end time.");

                if (_groups == null || _groups.Count == 0)
                    throw new InvalidOperationException("No groups loaded.");

                OpcTagItem targetTag = null;
                OpcGroupItem owningGroup = null;

                if (request.GroupId.HasValue)
                {
                    if (request.GroupId.Value < 1 || request.GroupId.Value > _groups.Count)
                        throw new ArgumentOutOfRangeException("gid", "Group id out of range.");
                    owningGroup = _groups[request.GroupId.Value - 1];
                    if (owningGroup == null || owningGroup.tags == null)
                        throw new InvalidOperationException("Group reference or tag list null.");
                    targetTag = owningGroup.tags.FirstOrDefault(t => string.Equals(t.Name, request.TagName, StringComparison.OrdinalIgnoreCase));
                    if (targetTag == null)
                        throw new KeyNotFoundException("Tag not found in specified group.");
                }
                else
                {
                    foreach (var g in _groups)
                    {
                        if (g?.tags == null) continue;
                        targetTag = g.tags.FirstOrDefault(t => string.Equals(t.Name, request.TagName, StringComparison.OrdinalIgnoreCase));
                        if (targetTag != null)
                        {
                            owningGroup = g;
                            break;
                        }
                    }
                    if (targetTag == null)
                        throw new KeyNotFoundException("Tag not found in any group.");
                }

                if (owningGroup == null)
                    throw new InvalidOperationException("Owning group not resolved.");

                OpcUaConnection conn;
                if (!_opcServers.TryGetValue(owningGroup.ProgID, out conn) || conn == null || conn.OpcSession == null || !conn.OpcSession.Connected)
                    throw new InvalidOperationException("No connected OPC UA session for group/tag.");

                var reader = new OpcUaHistoricalValues();
                List<DataValue> raw;
                try
                {
                    raw = (List<DataValue>)reader.ReadHistoricalData(conn.OpcSession, targetTag.Name, request.Start, request.End);
                }
                catch (Exception hx)
                {
                    throw new InvalidOperationException("Historical read failed: " + hx.Message);
                }

                var points = raw.Select(dv => new DataPoint
                {
                    Time = dv.SourceTimestamp,
                    Name = targetTag.Name,
                    Value = TryToDouble(dv.Value, out var v) ? v : 0,
                    Quality = (int)dv.StatusCode.Code,
                    Instance = _instance
                }).ToArray();

                response.GroupId = owningGroup.Id;
                response.TagName = targetTag.Name;
                response.Start = request.Start;
                response.End = request.End;
                response.Points = points;
                response.Success = true;
                response.StatusMessage = $"OK ({points.Length} point(s))";

                this.Publish(responsetopic, response);
                lasterror.Code = DdUsvcErrorCode.OK;
                lasterror.Reason = "";
            }
            catch (Exception ex)
            {
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;
                response.Success = false;
                response.StatusMessage = "GetTagHistoricalData failed: " + ex.Message;
                this.Publish(responsetopic, response);
                LogError(response.StatusMessage);
            }
            return lasterror;
        }

        private DdUsvcError GetGroupHistoricalData(string topic, string responsetopic, byte[] data)
        {
            var response = new GroupHistoryResponse
            {
                Success = false,
                StatusMessage = "Uninitialized"
            };
            try
            {
                if (data == null || data.Length == 0)
                    throw new ArgumentException("Empty request payload.");

                GroupHistoryRequest request;
                try
                {
                    request = JsonConvert.DeserializeObject<GroupHistoryRequest>(Encoding.UTF8.GetString(data));
                }
                catch (Exception dx)
                {
                    throw new ArgumentException("Failed to parse request: " + dx.Message);
                }

                if (request.GroupId < 1 || _groups == null || request.GroupId > _groups.Count)
                    throw new ArgumentOutOfRangeException("GroupId", "Group id out of range.");

                if (request.Start >= request.End)
                    throw new ArgumentException("Start time must be before end time.");

                var group = _groups[request.GroupId - 1];
                if (group == null)
                    throw new InvalidOperationException("Group reference null.");

                if (group.tags == null || group.tags.Count == 0)
                    throw new InvalidOperationException("Group has no tags.");

                OpcUaConnection conn;
                if (!_opcServers.TryGetValue(group.ProgID, out conn) || conn == null || conn.OpcSession == null || !conn.OpcSession.Connected)
                    throw new InvalidOperationException("No connected OPC UA session for group.");

                var session = conn.OpcSession;
                var historicalReader = new OpcUaHistoricalValues();
                var seriesList = new List<GroupHistoryTagSeries>();

                for (int i = 0; i < group.tags.Count; i++)
                {
                    var tag = group.tags[i];
                    if (tag == null || string.IsNullOrWhiteSpace(tag.Name))
                        continue;

                    List<DataValue> raw;
                    try
                    {
                        raw = (List<DataValue>)historicalReader.ReadHistoricalData(session, tag.Name, request.Start, request.End);
                    }
                    catch (Exception hx)
                    {
                        LogError($"Historical read failed for tag '{tag.Name}': {hx.Message}");
                        continue;
                    }

                    seriesList.Add(new GroupHistoryTagSeries
                    {
                        Tag = tag.Name,
                        Points = raw.Select(dv => new DataPoint
                        {
                            Time = dv.SourceTimestamp,
                            Name = tag.Name,
                            Value = TryToDouble(dv.Value, out var v) ? v : 0,
                            Quality = (int)dv.StatusCode.Code,
                            Instance = _instance
                        }).ToArray()
                    });
                }

                response.GroupId = request.GroupId;
                response.Start = request.Start;
                response.End = request.End;
                response.Series = seriesList.ToArray();
                response.Success = true;
                response.StatusMessage = $"OK ({response.Series.Length} tag series)";
                this.Publish(responsetopic, response);
                lasterror.Code = DdUsvcErrorCode.OK;
                lasterror.Reason = "";
            }
            catch (Exception ex)
            {
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;
                response.Success = false;
                response.StatusMessage = "GetGroupHistoricalData failed: " + ex.Message;
                this.Publish(responsetopic, response);
                LogError(response.StatusMessage);
            }
            return lasterror;
        }

        private bool TryToDouble(object value, out double result)
        {
            if (value is double d) { result = d; return true; }
            if (value is float f) { result = f; return true; }
            if (value is int i) { result = i; return true; }
            if (value is long l) { result = l; return true; }
            if (value is uint ui) { result = ui; return true; }
            if (value is short s) { result = s; return true; }
            if (value is ushort us) { result = us; return true; }
            if (value is byte b) { result = b; return true; }
            if (value is sbyte sb) { result = sb; return true; }
            if (value is decimal dec) { result = (double)dec; return true; }
            if (value == null)
            {
                result = 0;
                return false;
            }
            return double.TryParse(value.ToString(), out result);
        }

        private DdUsvcError GetOpcServerBranch(string topic, string responsetopic, byte[] data)
        {
            try
            {
                var request = JsonConvert.DeserializeObject<GetOPCBranches>(Encoding.UTF8.GetString(data));
                var sid = request.ServerId;

                if (_serverIds == null || _serverIds.Count == 0)
                    throw new KeyNotFoundException("_serverIds map is empty (no servers discovered yet).");

                string serverKey;
                if (!_serverIds.TryGetValue(sid, out serverKey))
                    throw new KeyNotFoundException("ServerId not found in map: " + sid);

                OpcUaConnection server;
                if (!_opcServers.TryGetValue(serverKey, out server) || server == null)
                    throw new InvalidOperationException("OPC server connection missing for key: " + serverKey);

                if (server.OpcSession == null || !server.OpcSession.Connected)
                    throw new InvalidOperationException("OPC server session not connected for key: " + serverKey);

                OpcUaTagBrowser browser = new OpcUaTagBrowser();
                BrowserPosition nodes = browser.BrowseBranch(server.OpcSession, request.Branch);
                nodes.ServerId = sid;
                this.Publish(responsetopic, nodes);
            }
            catch (Exception e)
            {
                var response = new BrowserPosition();
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = e.Message;
                response.Success = false;
                response.StatusMessage = $"GetOpcServerBranch failed: {e.Message}";
                LogError(response.StatusMessage);
                this.Publish(responsetopic, response);
            }
            return this.lasterror;
        }

        private DdUsvcError GetOpcServerRoot(string topic, string responsetopic, byte[] data)
        {
            try
            {
                var request = JsonConvert.DeserializeObject<IntMessage>(Encoding.UTF8.GetString(data));
                var sid = request.Value;

                if (_serverIds == null || _serverIds.Count == 0)
                    throw new KeyNotFoundException("_serverIds map is empty (no servers discovered yet).");

                string serverKey;
                if (!_serverIds.TryGetValue(sid, out serverKey))
                    throw new KeyNotFoundException("ServerId not found in map: " + sid);

                OpcUaConnection server;
                if (!_opcServers.TryGetValue(serverKey, out server) || server == null)
                    throw new InvalidOperationException("OPC server connection missing for key: " + serverKey);

                if (server.OpcSession == null || !server.OpcSession.Connected)
                    throw new InvalidOperationException("OPC server session not connected for key: " + serverKey);

                OpcUaTagBrowser browser = new OpcUaTagBrowser();
                BrowserPosition nodes = browser.BrowseRootNode(server.OpcSession);
                nodes.ServerId = sid;
                var payload = JsonConvert.SerializeObject(nodes);
                byte[] bytes = Encoding.UTF8.GetBytes(payload);
                broker.Publish(responsetopic, bytes);
                return new DdUsvcError { Code = DdUsvcErrorCode.OK, Reason = "" };
            }
            catch (Exception e)
            {
                var response = new BrowserPosition();
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = e.Message;
                response.Success = false;
                response.StatusMessage = $"GetOpcServerRoot failed: {e.Message}";
                LogError(response.StatusMessage);
                this.Publish(responsetopic, response);
            }
            return this.lasterror;
        }

        private DdUsvcError GetAllServers(string topic, string responsetopic, byte[] data)
        {
            var response = new OpcServerItemResponse();
    
            try
            {
                List<OpcServerItem> serverItems = FindOpcServers();

                response.Items = serverItems.ToArray();
                response.Success = true;
                response.StatusMessage = "Successfully retrieved OPC UA servers";

                this.Publish(responsetopic, response);
                return new DdUsvcError { Code = DdUsvcErrorCode.OK, Reason = "" };
            }
            catch (Exception ex)
            {
                response.Success = false;
                response.StatusMessage = $"Failed to retrieve OPC UA servers: {ex.Message}";

                this.Publish(responsetopic, response);
                return new DdUsvcError { Code = DdUsvcErrorCode.Error, Reason = ex.Message };
            }
        }

        private string ResolveEndpointUrl(ApplicationDescription app, string fallbackDiscoveryUrl)
        {
            if (app == null) return null;
            if (app.DiscoveryUrls != null)
            {
                foreach (var url in app.DiscoveryUrls)
                {
                    if (!string.IsNullOrWhiteSpace(url) && url.StartsWith("opc.tcp://", StringComparison.OrdinalIgnoreCase))
                        return url.Trim();
                }
            }
            if (!string.IsNullOrWhiteSpace(fallbackDiscoveryUrl) && fallbackDiscoveryUrl.StartsWith("opc.tcp://", StringComparison.OrdinalIgnoreCase))
                return fallbackDiscoveryUrl;

            return null;
        }

        private List<OpcServerItem> FindOpcServers()
        {
            List<OpcServerItem> serverItems = new List<OpcServerItem>();
            int id = 1;

            foreach (var server in _opcServers)
            {
                if (_serverIds == null)
                {
                    _serverIds = new Dictionary<int, string>();
                }
                if (!_serverIds.ContainsKey(id))
                {
                    _serverIds.Add(id, server.Key);
                }
                serverItems.Add(new OpcServerItem
                {
                    ID = id++,
                    ProgID = server.Key,
                    Name = server.Value?.OpcSession?.Endpoint?.EndpointUrl ?? server.Key
                });
            }

            try
            {
                //string defaultDiscoveryUrl = "opc.tcp://localhost:4840";
                //Console.WriteLine("Using default discovery URL: " + defaultDiscoveryUrl);
                var config = OpcUaConnection.GetApplicationConfiguration();
                //try
                //{
                //    using (var discoveryClient = DiscoveryClientSafeCreate(config, new Uri(defaultDiscoveryUrl)))
                //    {
                //        if (discoveryClient != null)
                //            BrowseAndConnect(discoveryClient, defaultDiscoveryUrl, ref id, serverItems);
                //    }
                //}
                //catch (Exception ex)
                //{
                //    LogError($"Error discovering servers from default URL: {ex.Message}, continuing with additional discovery URLs...");
                //}

                string additionalUrls = settings.ContainsKey("discovery-urls") ? settings["discovery-urls"] : null;
                if (!string.IsNullOrWhiteSpace(additionalUrls))
                {
                    foreach (var raw in additionalUrls.Split(','))
                    {
                        var discoveryUrl = raw.Trim();
                        if (string.IsNullOrEmpty(discoveryUrl)) continue;

                        try
                        {
                            using (var discoveryClient = DiscoveryClientSafeCreate(config, new Uri(discoveryUrl)))
                            {
                                if (discoveryClient != null)
                                    BrowseAndConnect(discoveryClient, discoveryUrl, ref id, serverItems);
                            }
                        }
                        catch (Exception exUrl)
                        {
                            LogError($"Error discovering servers from {discoveryUrl}: {exUrl.Message}");
                        }
                    }
                }
                LogEvent(_opcServers.Count + " OPC UA servers connected on init.");
            }
            catch (Exception discEx)
            {
                LogError($"Error during server discovery (top-level): {discEx.Message}");
            }


            if (_serverIds == null)
                _serverIds = new Dictionary<int, string>();
            else
                _serverIds.Clear();

            foreach (var item in serverItems)
            {
                if (!_serverIds.ContainsKey(item.ID))
                {
                    _serverIds[item.ID] = item.ProgID;
                }
            }

            return serverItems;
        }

        private DiscoveryClient DiscoveryClientSafeCreate(ApplicationConfiguration config, Uri uri)
        {
            try
            {
                return Opc.Ua.DiscoveryClient.Create(config, uri);
            }
            catch (TypeInitializationException tie)
            {
                LogError("TypeInitializationException creating DiscoveryClient for: " + uri);
            }
            catch (Exception ex)
            {
                LogError("DiscoveryClient.Create failed for " + uri + ": " + ex.Message);
            }
            return null;
        }

        private void BrowseAndConnect(DiscoveryClient discoveryClient, string discoveryUrl, ref int id, List<OpcServerItem> serverItems)
        {
            var servers = discoveryClient.FindServers(null);
            if (servers == null) return;

            foreach (var app in servers)
            {
                string endpointUrl = ResolveEndpointUrl(app, discoveryUrl);
                if (string.IsNullOrEmpty(endpointUrl))
                {
                    LogError($"No usable opc.tcp endpoint for application: {app.ApplicationUri} via {discoveryUrl}");
                    continue;
                }
                if (serverItems.Any(s => s.ProgID == endpointUrl)) continue;

                Console.WriteLine("Found server: {0} - {1} (endpoint: {2})",
                    app.ApplicationName?.Text, app.ApplicationUri, endpointUrl);

                serverItems.Add(new OpcServerItem
                {
                    ID = id++,
                    ProgID = endpointUrl,
                    Name = app.ApplicationName?.Text ?? endpointUrl
                });

                try
                {
                    var connection = new OpcUaConnection();
                    if (connection.ConnectToServer(endpointUrl))
                    {
                        _opcServers[endpointUrl] = connection;
                    }
                    else
                    {
                        LogError($"Failed to connect to OPC UA endpoint: {endpointUrl}");
                    }
                }
                catch (Exception cex)
                {
                    LogError($"Exception connecting to endpoint {endpointUrl}: {cex.Message}");
                }
            }
        }

        public void Initialize()
        {
            FindOpcServers();
            LoadGroups("groups.csv");
            LoadTags("tags.csv");
            CreateInitialSubscriptions();
            LogEvent("Initialized");
        }
        internal DdUsvcError StopGroup(string topic, string responsetopic, byte[] data)
        {
            var response = new StatusResponse();
            try
            {
                var request = JsonConvert.DeserializeObject<IntMessage>(Encoding.UTF8.GetString(data));
                if (request.Value >= 1 && request.Value <= _groups.Count)
                {
                    var group = _groups[request.Value - 1];
                    if (group.State == OpcGroupState.GroupStateStopped)
                    {
                        response.StatusMessage = $"Group already stopped, group: {group.Name} (id: {group.Id})";
                    }
                    else if (group.State != OpcGroupState.GroupStateDisabled)
                    {
                        group.State = OpcGroupState.GroupStateStopped;
                        response.Success = true;
                        if (group.subscription != null)
                            group.subscription.StopSubscription(group);
                        LogEvent($"Group stopped: {group.Id}, state: {group.State}");
                    }
                }
                this.Publish(responsetopic, response);
            }
            catch (Exception ex)
            {
                response.Success = false;
                response.StatusMessage = ex.Message;
                this.Publish(responsetopic, response);
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;
            }
            return this.lasterror;
        }

        internal DdUsvcError StartGroup(string topic, string responsetopic, byte[] data)
        {
            var response = new StatusResponse();
            try
            {
                var request = JsonConvert.DeserializeObject<IntMessage>(Encoding.UTF8.GetString(data));
                if (request.Value >= 1 && request.Value <= _groups.Count)
                {
                    var group = _groups[request.Value - 1];
                    if (group.State == OpcGroupState.GroupStateRunning || group.State == OpcGroupState.GroupStateRunningWithWarning)
                    {
                        response.StatusMessage = $"Group already running, group: {group.Name} (id: {group.Id})";
                    }
                    else if (group.State != OpcGroupState.GroupStateDisabled)
                    {
                        group.State = OpcGroupState.GroupStateRunning;
                        response.Success = true;
                        if (group.subscription == null)
                            group.subscription = _subscription;
                        if (group.subscription != null)
                            group.subscription.AddTagsToGroupSubscription(group);
                        LogEvent($"Group started: {group.Id}, state: {group.State}");
                    }
                }
                this.Publish(responsetopic, response);
            }
            catch (Exception ex)
            {
                response.Success = false;
                response.StatusMessage = ex.Message;
                this.Publish(responsetopic, response);
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;
            }
            return this.lasterror;
        }

        private DdUsvcError GetAllGroups(string topic, string responsetopic, byte[] data)
        {
            var response = new OpcGroupItemResponse { Success = true };
            try
            {
                response.Items = _groups.ToArray();

                this.Publish(responsetopic, response);
            }
            catch (Exception ex)
            {
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;

                response.Success = false;
                response.StatusMessage = ex.Message;

                LogError($"groups.getall responding to ... {responsetopic}, ex: {response.StatusMessage}");
                this.Publish(responsetopic, response);
            }

            return this.lasterror;
        }

        private DdUsvcError GetAllTags(string topic, string responsetopic, byte[] data)
        {
            var response = new OpcTagItemResponse { Success = true };
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
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;

                response.Success = false;
                response.StatusMessage = ex.Message;

                var err = this.Publish(responsetopic, response);
                if (err.Code == DdUsvcErrorCode.Error)
                {
                    LogError($"tags.getall responding FAILED ... {responsetopic}, err: {err.Reason}, ex: {response.StatusMessage}");
                }
            }

            return this.lasterror;
        }

        private DdUsvcError GetAllTagsUnfiltered(string topic, string responsetopic, byte[] data)
        {
            OpcTagItemMetaResponse response = new OpcTagItemMetaResponse { Success = true };

            try
            {
                List<OpcTagMetaInfo> items = GetAllTagsUnfilteredInternal();


                response.Items = items.ToArray();
                var err = this.Publish(responsetopic, response);
                if (err.Code == DdUsvcErrorCode.Error)
                {
                    LogError($"tags.getallunfiltered responding FAILED ... {responsetopic}, err: {err.Reason}");
                }
            }
            catch (Exception ex)
            {
                lasterror.Code = DdUsvcErrorCode.Error;
                lasterror.Reason = ex.Message;

                response.Success = false;
                response.StatusMessage = ex.Message;

                var err = this.Publish(responsetopic, response);
                if (err.Code == DdUsvcErrorCode.Error)
                {
                    LogError($"tags.getallunfiltered responding FAILED ... {responsetopic}, err: {err.Reason}, ex: {response.StatusMessage}");
                }
            }
            return this.lasterror;
        }

        private List<OpcTagMetaInfo> GetAllTagsUnfilteredInternal()
        {
            var items = new List<OpcTagMetaInfo>();
            LogEvent("Getting all tags unfiltered...");
            if (_opcServers == null || _opcServers.Count == 0)
            {
                LogError("No OPC UA servers available (_opcServers empty).");
            }
            else
            {
                OpcUaTagBrowser browser = new OpcUaTagBrowser();
                foreach (var server in _opcServers)
                {
                    if (server.Value == null)
                    {
                        LogError($"Server entry {server.Key} has null connection object.");
                        continue;
                    }
                    var session = server.Value.OpcSession;
                    if (session == null || !session.Connected)
                    {
                        LogError($"Skipping browse on server {server.Key}: session is null or not connected.");
                        continue;
                    }

                    LogEvent($"Browsing tags on server: {server.Key}");
                    List<TagInfo> tags = null;
                    try
                    {
                        // tags = new OpcUaTagBrowser().BrowseTags(session);
                        tags = browser.BrowseTags(session);
                    }
                    catch (Exception bx)
                    {
                        LogError($"Browsing failed on {server.Key}: {bx.Message}");
                        continue;
                    }

                    if (tags != null)
                    {
                        for (int i = 0; i < tags.Count; i++)
                        {
                            var tag = tags[i];
                            items.Add(new OpcTagMetaInfo
                            {
                                Name = tag.Name,
                                Description = tag.Description,
                                EngineeringUnit = tag.EngineeringUnit,
                                Min = tag.MinValue,
                                Max = tag.MaxValue
                            });
                        }
                    }
                }
            }

            return items;
        }

        private void SetTimer()
        {
            aTimer = new Timer(1000);
            aTimer.Elapsed += ATimer_Elapsed;
            aTimer.AutoReset = true;
            aTimer.Enabled = true;
            if (IsMetaTimerEnabled())
            {
                metaTimer = new Timer(3600000);
                metaTimer.Elapsed += MetaTimer_Elapsed;
                metaTimer.AutoReset = true;
                metaTimer.Enabled = true;
            }
            else
            {
                metaTimer = null;
                LogEvent("Meta timer disabled by setting 'tag-meta-timer-enabled=false'");
            }
        }

        private bool IsMetaTimerEnabled()
        {
            try
            {
                if (settings != null && settings.TryGetValue("tag-meta-timer-enabled", out string value))
                {
                    if (string.IsNullOrWhiteSpace(value)) return false;
                    if (string.Equals(value, "1")) return true;
                    if (string.Equals(value, "true")) return true;
                    if (string.Equals(value, "0")) return false;
                    if (string.Equals(value, "false")) return false;
                    return false;
                }
            }
            catch { }
            return false;
        }

        private void MetaTimer_Elapsed(object sender, ElapsedEventArgs e)
        {
            var tags = GetAllTagsUnfilteredInternal();
            var payload = JsonConvert.SerializeObject(tags);
            byte[] bytes = Encoding.UTF8.GetBytes(payload);
            broker.Publish($"process.taglist", bytes);
            LogEvent($"Published process.taglist response with {tags.Count} items");
        }

        private void ATimer_Elapsed(object sender, ElapsedEventArgs e)
        {
            try
            {
                foreach (var kv in _opcServers)
                {
                    var opcServer = kv.Value;
                    if (opcServer != null)
                    {
                        var statusObj = opcServer.GetStatus();
                        if (statusObj == null)
                        {
                            LogError($"Status retrieval returned null for server {kv.Key} (session may be null).");
                            continue;
                        }
                        var msg = new ServerStatus();
                        msg.ProgId = kv.Key;
                        msg.State = statusObj.Connected.ToString();
                        msg.LastUpdate = statusObj.LastContactTime;
                        msg.BandWidth = 0;
                        msg.GroupCount = 0;
                        msg.Instance = _instance;
                        msg.HostName = System.Environment.MachineName;
                        var payload = JsonConvert.SerializeObject(msg);
                        byte[] bytes = Encoding.UTF8.GetBytes(payload);
                        broker.Publish("process.opc.server.status", bytes);
                    }
                    else
                    {
                        LogEvent("OPC UA server instance is NULL (unexpected).");
                    }
                }
            }
            catch (Exception ex)
            {
                LogError($"Exception caught when checking OPC UA servers: {ex.Message}");
                Restart();
            }
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
                if (group == null) continue;
                if (group.State == OpcGroupState.GroupStateDisabled) continue;
                if (group.RunAtStart && group.subscription != null &&
                    group.State == OpcGroupState.GroupStateStopped)
                {
                    group.State = OpcGroupState.GroupStateRunning;
                    LogEvent($"Starting group: {group.Name}");
                }
            }
            PrepareOpc();
        }

        public void DataChanged(OpcTagItem tagRef, MonitoredItem monitoredItem)
        {
            try
            {
                if (tagRef.Error != 0)
                {
                    LogError("Exception while processing data changed event.");
                    return;
                }
                var point = new DataPoint();
                point.Time = tagRef.Time;
                point.Name = tagRef.Name;
                point.Value = tagRef.Value;
                point.Quality = tagRef.Quality;
                point.Instance = _instance;
                var payload = JsonConvert.SerializeObject(point);
                byte[] bytes = Encoding.UTF8.GetBytes(payload);
                // LogEvent("Data changed: " + payload);
                broker.Publish("process.actual", bytes);
            }
            catch (Exception ex)
            {
                LogError($"Exception while processing data changed event: {ex.ToString()}");
            }
        }


        internal void LoadGroups(string filename)
        {
            try
            {
                _groups = new List<OpcGroupItem>();
                if (_subscription == null)
                    _subscription = new OpcUaSubscription(this);

                if (!File.Exists(filename))
                {
                    LogError("Groups file not found: " + filename + ", creating a single one second group ...");
                    var group = new OpcGroupItem { Id = 1, DefaultGroup = true, Interval = 1, RunAtStart = true, ProgID = _opcServers.FirstOrDefault().Key };
                    _groups.Add(group);

                    OpcUaConnection opcServer;
                    if (!_opcServers.TryGetValue(group.ProgID, out opcServer))
                    {
                        if (!opcServer.ConnectToServer(group.ProgID))
                        {
                            LogError($"Reconnect failed to {group.ProgID} for group {group.Name}. Disabling group.");
                            group.State = OpcGroupState.GroupStateDisabled;
                        }
                    }
                    return;
                }

                var config = new CsvConfiguration(CultureInfo.InvariantCulture)
                {
                    Delimiter = ";",
                    HasHeaderRecord = true,
                    Encoding = Encoding.UTF8,
                    IgnoreBlankLines = true,
                    MissingFieldFound = null,
                    HeaderValidated = null,
                    BadDataFound = ctx =>
                    {
                        try
                        {
                            LogError("Bad CSV data in groups file: " + ctx.RawRecord?.Trim());
                        }
                        catch { }
                    }
                };

                List<GroupCsvRecord> records;
                using (var reader = new StreamReader(filename, Encoding.UTF8))
                using (var csv = new CsvReader(reader, config))
                {
                    csv.Context.RegisterClassMap<GroupCsvRecordMap>();
                    try
                    {
                        records = csv.GetRecords<GroupCsvRecord>().ToList();
                    }
                    catch (Exception exParse)
                    {
                        LogError("Failed parsing groups CSV (" + filename + "): " + exParse.Message);
                        return;
                    }
                }

                if (records == null || records.Count == 0)
                {
                    LogEvent("No group rows found in file: " + filename);
                    return;
                }

                foreach (var r in records)
                {
                    if (r == null) continue;

                    if (r.Id == 0 && string.Equals(r.Name, "groupname", StringComparison.OrdinalIgnoreCase))
                        continue;

                    if (r.Id < 1)
                    {
                        LogError("Ignoring group with invalid id (<1). Name=" + r.Name);
                        continue;
                    }

                    var group = new OpcGroupItem
                    {
                        Id = r.Id,
                        Name = string.IsNullOrWhiteSpace(r.Name) ? ("group-" + r.Id) : r.Name.Trim(),
                        Interval = r.Interval <= 0 ? 1000 : r.Interval,
                        ProgID = string.IsNullOrWhiteSpace(r.ProgID) ? "default" : r.ProgID.Trim(),
                        DefaultGroup = r.DefaultGroupRaw == 1,
                        RunAtStart = r.RunAtStartRaw == 1,
                        State = OpcGroupState.GroupStateStopped,
                        tags = new List<OpcTagItem>()
                    };

                    if (group.ProgID == "default")
                    {
                        if (_opcServers.Count == 0)
                        {
                            LogError($"Group {group.Name} is configured to use 'default' OPC UA server, but no servers are available. Disabling group.");
                            group.State = OpcGroupState.GroupStateDisabled;
                            _groups.Add(group);
                            continue;
                        }
                        var defaultServer = _opcServers.FirstOrDefault().Key;
                        group.ProgID = defaultServer;
                    }

                    if (group.ProgID.StartsWith("urn:", StringComparison.OrdinalIgnoreCase))
                    {
                        LogError($"Group {group.Name} has an invalid OPC UA endpoint (URN). Skipping / disabling.");
                        group.State = OpcGroupState.GroupStateDisabled;
                        _groups.Add(group);
                        continue;
                    }

                    try
                    {
                        OpcUaConnection opcServer;
                        if (!_opcServers.TryGetValue(group.ProgID, out opcServer))
                        {
                            opcServer = new OpcUaConnection();
                            _opcServers[group.ProgID] = opcServer;
                            if (!opcServer.ConnectToServer(group.ProgID))
                            {
                                LogError($"Failed to connect to OPC UA endpoint: {group.ProgID} for group: {group.Name}. Disabling group.");
                                group.State = OpcGroupState.GroupStateDisabled;
                            }
                        }
                        else if (opcServer.OpcSession == null || !opcServer.OpcSession.Connected)
                        {
                            if (!opcServer.ConnectToServer(group.ProgID))
                            {
                                LogError($"Reconnect failed to {group.ProgID} for group {group.Name}. Disabling group.");
                                group.State = OpcGroupState.GroupStateDisabled;
                            }
                        }
                    }
                    catch (Exception cx)
                    {
                        group.State = OpcGroupState.GroupStateDisabled;
                        LogError($"Exception connecting endpoint {group.ProgID} for group {group.Name}: {cx.Message}");
                    }

                    _groups.Add(group);
                }
                _groups = _groups.OrderBy(g => g.Id).ToList();
                bool contiguous = true;
                for (int i = 0; i < _groups.Count; i++)
                {
                    if (_groups[i].Id != i + 1)
                    {
                        contiguous = false;
                        break;
                    }
                }
                if (!contiguous)
                {
                    LogError("Group IDs are not contiguous starting at 1. Tag loading by index may fail.");
                }

                LogEvent("Loaded " + _groups.Count + " group(s) from " + filename);
            }
            catch (Exception e)
            {
                LogEvent("Failed to load groups: " + e.Message);
            }
        }

        internal void LoadTags(string filename)
        {
            try
            {
                _tags = new List<OpcTagItem>();

                if (!File.Exists(filename))
                {
                    LogEvent($"Tags file not found: {filename}, loading all tags ...");

                    List<OpcTagMetaInfo> items = GetAllTagsUnfilteredInternal();
                    foreach (var item in items)
                    {
                        var group = _groups[0];
                        var tag = new OpcTagItem { Name = item.Name, Group = group, GroupID = 1 };
                        _tags.Add(tag);
                        if (group.tags == null) group.tags = new List<OpcTagItem>();
                        group.tags.Add(tag);
                    }
                    return;
                }

                var config = new CsvConfiguration(CultureInfo.InvariantCulture)
                {
                    Delimiter = ";",
                    HasHeaderRecord = true,
                    Encoding = Encoding.UTF8,
                    IgnoreBlankLines = true,
                    MissingFieldFound = null,
                    HeaderValidated = null,
                    BadDataFound = ctx =>
                    {
                        try
                        {
                            LogEvent($"Bad CSV data in tags file: {ctx.RawRecord?.Trim()}");
                        }
                        catch { }
                    }
                };

                int tagid = 0;
                int rawCount = 0;

                using (var reader = new StreamReader(filename, Encoding.UTF8))
                using (var csv = new CsvReader(reader, config))
                {
                    csv.Context.RegisterClassMap<TagCsvRecordMap>();
                    List<TagCsvRecord> records;
                    try
                    {
                        records = csv.GetRecords<TagCsvRecord>().ToList();
                    }
                    catch (Exception exParse)
                    {
                        LogEvent($"Failed parsing tags CSV ({filename}): {exParse.Message}");
                        return;
                    }

                    rawCount = records.Count;

                    foreach (var r in records)
                    {
                        if (r == null) continue;
                        if (string.Equals(r.Name, "name", StringComparison.OrdinalIgnoreCase) &&
                            string.Equals(r.GroupIdRaw, "groupid", StringComparison.OrdinalIgnoreCase))
                            continue;
                        if (r.GroupId < 1 || r.GroupId > _groups.Count)
                        {
                            LogEvent($"Tag row refers to group id out of range (1 based). {r.GroupId} < 1 || {r.GroupId} > {_groups.Count}. Row ignored: name={r.Name}");
                            continue;
                        }

                        var group = _groups[r.GroupId - 1];
                        if (group == null || group.State == OpcGroupState.GroupStateDisabled)
                            continue;

                        if (string.IsNullOrWhiteSpace(r.Name))
                        {
                            LogEvent($"Empty tag name in group {r.GroupId}, row ignored.");
                            continue;
                        }

                        var tag = new OpcTagItem
                        {
                            Id = tagid++,
                            GroupID = r.GroupId,
                            Group = group,
                            Name = r.Name.Trim()
                        };

                        _tags.Add(tag);
                        if (group.tags == null)
                            group.tags = new List<OpcTagItem>();
                        group.tags.Add(tag);

                        LogEvent("Loaded tag: " + tag.Name + " in group: " + group.Name);
                    }
                }

                LogEvent("Found " + _tags.Count + " tags (parsed " + rawCount + " rows).");
            }
            catch (Exception e)
            {
                LogEvent($"Failed to load tags: {e.Message}");
            }
        }

        internal void PrepareOpc()
        {
            foreach (var group in _groups)
            {
                if (group == null) continue;
                if (group.State == OpcGroupState.GroupStateDisabled) continue;
                if (group.tags == null || group.tags.Count == 0) continue;
                if (group.State == OpcGroupState.GroupStateRunning && group.subscription != null)
                {
                    group.subscription.AddTagsToGroupSubscription(group);
                }
            }
        }

        public void Shutdown()
        {
            foreach (var kv in _opcServers)
            {
                var opcServer = kv.Value;
                if (opcServer != null)
                {
                    try
                    {
                        opcServer.Dispose();
                    }
                    catch (Exception ex)
                    {
                        LogError($"Error disposing OPC UA server {kv.Key}: {ex.Message}");
                    }
                }
            }
        }

        private void CreateInitialSubscriptions()
        {
            if (_subscription == null)
                _subscription = new OpcUaSubscription(this);

            foreach (var group in _groups)
            {
                if (group == null) continue;
                if (!group.RunAtStart) continue;
                if (group.State == OpcGroupState.GroupStateDisabled) continue;
                if (group.tags == null || group.tags.Count == 0)
                {
                    LogError($"Group {group.Id} ({group.Name}) has no tags; skipping subscription creation.");
                    continue;
                }

                OpcUaConnection conn;
                if (!_opcServers.TryGetValue(group.ProgID, out conn) ||
                    conn == null || conn.OpcSession == null || !conn.OpcSession.Connected)
                {
                    LogError($"No connected session for group {group.Id} ({group.Name}); disabling group.");
                    group.State = OpcGroupState.GroupStateDisabled;
                    continue;
                }

                if (group.subscription == null)
                    group.subscription = _subscription;

                var sub = _subscription.GroupSubscription(conn.OpcSession, group);
                if (sub == null)
                {
                    LogError($"Failed to create subscription for group {group.Id} ({group.Name}); tags: {group.tags.Count}. Disabling group.");
                    group.State = OpcGroupState.GroupStateDisabled;
                }
            }
        }
    }
}