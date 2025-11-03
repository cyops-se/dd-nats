using DdUsvc;
using Newtonsoft.Json;
using System;
using System.Collections.Generic;
using System.IO;
using System.Linq;
using System.Text;
using CsvHelper;
using CsvHelper.Configuration;
using System.Globalization;
using dd_opcua_lib;
using System.Threading;

namespace dd_opcua_test
{
    internal class Program
    {
        static void Main(string[] args)
        {
            Console.WriteLine("Starting dd-opcua-test...");

            var url = args != null && args.Length > 0 && !string.IsNullOrWhiteSpace(args[0])
                ? args[0]
                : "nats://127.0.0.1:4222";
            var client = new TestClient(url);

            Console.WriteLine("Testing browsing OPC UA servers and nodes (request/reply)...");
            client.TestBrowseRootAndNodes();

            Console.WriteLine("Requesting all tags unfiltered (request/reply) and creating csv files...");
            client.RequestGetAllTagsUnfiltered();

            Console.WriteLine("Testing group start/stop (request/reply)...");
            client.TestGroupStartStop();

            Console.WriteLine("Requesting group historical data (request/reply)...");
            client.RequestGroupHistory();

            Console.WriteLine("Requesting single tag historical data (request/reply)...");
            client.RequestTagHistory();

            client.WaitAndGenerateProtocol(TimeSpan.FromSeconds(30));

            Console.WriteLine("Test complete. The program keeps running for broadcasts until exit (protocol already generated).");
            Console.WriteLine("Press <Enter> to exit...");
            Console.ReadLine();
        }
    }

    public class TestClient
    {
        private readonly IMessageBroker _broker;

        private readonly string _instance;
        private readonly HashSet<string> _visitedBrowsePositions = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        private int _browseRequestsMade = 0;
        private const int _maxBrowseRequests = 50;

        private readonly TestProtocolResult _protocol = new TestProtocolResult
        {
            StartedUtc = DateTime.UtcNow
        };

        private readonly object _sampleLock = new object();

        public TestClient(string url)
        {
            _instance = "default";

            if (url.StartsWith("mqtt", StringComparison.OrdinalIgnoreCase))
            {
                Console.WriteLine("Connecting to MQTT broker at: " + url);
                _broker = new DdUsvcMqttBroker(url);
            }
            else
            {
                Console.WriteLine("Connecting to NATS broker at: " + url);
                _broker = new DdUsvcNatsBroker(url);
            }

            var err = _broker.Connect();
            if (err.Code == DdUsvcErrorCode.Error)
            {
                Console.WriteLine("Broker connect error: " + err.Reason);
                throw new InvalidOperationException(err.Reason ?? "Broker connect failed.");
            }

            Subscribe("process.opc.server.status", this.OnServerStatus);
            Subscribe("process.actual", this.OnDataPoint);
            Subscribe("process.taglist", this.OnPeriodicTagList);

            Console.WriteLine("Subscribed to broadcast topics:");
            Console.WriteLine("  process.opc.server.status");
            Console.WriteLine("  process.actual");
            Console.WriteLine("  process.taglist");
        }

        private void Subscribe(string topic, IMessageHandler callback) => _broker.Subscribe(topic, callback);
        private byte[] Request(string topic, byte[] data) => _broker.Request(topic, data);

        public void WaitAndGenerateProtocol(TimeSpan wait)
        {
            Console.WriteLine($"Waiting {wait.TotalSeconds:n0}s before generating test protocol...");
            var end = DateTime.UtcNow + wait;
            while (DateTime.UtcNow < end)
            {
                var remaining = (int)(end - DateTime.UtcNow).TotalSeconds;
                Console.Write($"\r  Remaining: {remaining,2}s   ");
                Thread.Sleep(500);
            }
            Console.WriteLine();
            GenerateProtocolFile();
        }

        public void TestGroupStartStop()
        {
            try
            {
                var groupsTopic = $"usvc.opc.{_instance}.groups.getall";
                var bytes = Request(groupsTopic, new byte[0]);
                if (bytes == null)
                {
                    Console.WriteLine("[GRPTEST] groups.getall timeout.");
                    _protocol.Errors.Add("groups.start/stop: initial groups.getall timeout/null.");
                    return;
                }

                var resp = JsonConvert.DeserializeObject<OpcGroupItemResponse>(Encoding.UTF8.GetString(bytes));
                if (resp == null || !resp.Success || resp.Items == null || resp.Items.Length == 0)
                {
                    Console.WriteLine("[GRPTEST] Invalid groups response.");
                    _protocol.Errors.Add("groups.start/stop: invalid initial groups response.");
                    return;
                }

                var target = resp.Items
                    .Where(g => g != null && g.State != OpcGroupState.GroupStateDisabled)
                    .OrderBy(g => g.Id)
                    .FirstOrDefault();

                if (target == null)
                {
                    Console.WriteLine("[GRPTEST] No suitable group for start/stop test.");
                    _protocol.Errors.Add("groups.start/stop: no suitable group.");
                    return;
                }

                _protocol.GroupStartStopTestPerformed = true;
                _protocol.GroupStartStopGroupId = target.Id;
                _protocol.GroupInitialState = target.State.ToString();

                var startTopic = $"usvc.opc.{_instance}.groups.start";
                var stopTopic = $"usvc.opc.{_instance}.groups.stop";
                bool needStopFirst = target.State == OpcGroupState.GroupStateRunning || target.State == OpcGroupState.GroupStateRunningWithWarning;
                bool needStartFirst = target.State == OpcGroupState.GroupStateStopped;

                if (needStopFirst)
                {
                    if (!SendGroupCommandAndVerify(stopTopic, target.Id, OpcGroupState.GroupStateStopped, out var afterStopState, out var stopMsg))
                    {
                        _protocol.GroupStartStopStatusMessage = "Stop verification failed: " + stopMsg;
                        _protocol.Errors.Add(_protocol.GroupStartStopStatusMessage);
                        return;
                    }
                    _protocol.GroupStateAfterStop = afterStopState;
                    _protocol.GroupStopVerified = true;

                    if (!SendGroupCommandAndVerify(startTopic, target.Id, OpcGroupState.GroupStateRunning, out var afterStartState, out var startMsg))
                    {
                        _protocol.GroupStartStopStatusMessage = "Start verification after stop failed: " + startMsg;
                        _protocol.Errors.Add(_protocol.GroupStartStopStatusMessage);
                        return;
                    }
                    _protocol.GroupStateAfterStart = afterStartState;
                    _protocol.GroupStartVerified = true;
                }
                else if (needStartFirst)
                {
                    if (!SendGroupCommandAndVerify(startTopic, target.Id, OpcGroupState.GroupStateRunning, out var afterStartState, out var startMsg))
                    {
                        _protocol.GroupStartStopStatusMessage = "Initial start verification failed: " + startMsg;
                        _protocol.Errors.Add(_protocol.GroupStartStopStatusMessage);
                        return;
                    }
                    _protocol.GroupStateAfterStart = afterStartState;
                    _protocol.GroupStartVerified = true;

                    if (!SendGroupCommandAndVerify(stopTopic, target.Id, OpcGroupState.GroupStateStopped, out var afterStopState, out var stopMsg))
                    {
                        _protocol.GroupStartStopStatusMessage = "Stop verification after start failed: " + stopMsg;
                        _protocol.Errors.Add(_protocol.GroupStartStopStatusMessage);
                        return;
                    }
                    _protocol.GroupStateAfterStop = afterStopState;
                    _protocol.GroupStopVerified = true;

                }
                else
                {
                    if (SendGroupCommandAndVerify(startTopic, target.Id, OpcGroupState.GroupStateRunning, out var afterStartState, out var startMsg))
                    {
                        _protocol.GroupStateAfterStart = afterStartState;
                        _protocol.GroupStartVerified = true;
                        if (SendGroupCommandAndVerify(stopTopic, target.Id, OpcGroupState.GroupStateStopped, out var afterStopState, out var stopMsg))
                        {
                            _protocol.GroupStateAfterStop = afterStopState;
                            _protocol.GroupStopVerified = true;
                        }
                    }
                }

                if (_protocol.GroupStartVerified && _protocol.GroupStopVerified)
                    _protocol.GroupStartStopStatusMessage = "OK";
                else if (string.IsNullOrEmpty(_protocol.GroupStartStopStatusMessage))
                    _protocol.GroupStartStopStatusMessage = "Partial success";
            }
            catch (Exception ex)
            {
                Console.WriteLine("[GRPTEST] Exception: " + ex.Message);
                _protocol.Errors.Add("groups.start/stop exception: " + ex.Message);
            }
        }

        private bool SendGroupCommandAndVerify(string topic, int groupId, OpcGroupState expectedState, out string observedStateStr, out string statusMessage)
        {
            observedStateStr = null;
            statusMessage = null;
            try
            {
                var payload = Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(new IntMessage { Value = groupId }));
                var respBytes = Request(topic, payload);
                if (respBytes == null)
                {
                    statusMessage = "Command timeout/null for topic " + topic;
                    return false;
                }
                var statusResp = JsonConvert.DeserializeObject<StatusResponse>(Encoding.UTF8.GetString(respBytes));
                if (statusResp == null)
                {
                    statusMessage = "StatusResponse parse failed for " + topic;
                    return false;
                }
                if (!statusResp.Success && !string.IsNullOrEmpty(statusResp.StatusMessage))
                {
                    if (statusResp.StatusMessage.IndexOf("already", StringComparison.OrdinalIgnoreCase) < 0)
                    {
                        statusMessage = "Command unsuccessful: " + statusResp.StatusMessage;
                        return false;
                    }
                }

                Thread.Sleep(5000);

                var groupsTopic = $"usvc.opc.{_instance}.groups.getall";
                var groupsBytes = Request(groupsTopic, new byte[0]);
                if (groupsBytes == null)
                {
                    statusMessage = "Post-command groups.getall timeout";
                    return false;
                }
                var groupsResp = JsonConvert.DeserializeObject<OpcGroupItemResponse>(Encoding.UTF8.GetString(groupsBytes));
                if (groupsResp == null || !groupsResp.Success || groupsResp.Items == null)
                {
                    statusMessage = "Post-command groups response invalid";
                    return false;
                }
                var g = groupsResp.Items.FirstOrDefault(x => x != null && x.Id == groupId);
                if (g == null)
                {
                    statusMessage = "Group missing after command";
                    return false;
                }
                observedStateStr = g.State.ToString();
                if (g.State == expectedState ||
                    (expectedState == OpcGroupState.GroupStateRunning &&
                     g.State == OpcGroupState.GroupStateRunningWithWarning))
                {
                    return true;
                }
                statusMessage = $"Expected {expectedState} got {g.State}";
                return false;
            }
            catch (Exception ex)
            {
                statusMessage = "Exception: " + ex.Message;
                return false;
            }
        }

        public void TestBrowseRootAndNodes()
        {
            try
            {
                var serversTopic = $"usvc.opc.{_instance}.servers.getall";
                var serversBytes = Request(serversTopic, new byte[0]);
                if (serversBytes == null)
                {
                    Console.WriteLine("[SERVERS] Request timeout / null response.");
                    _protocol.Errors.Add("Servers request timeout/null.");
                    return;
                }
                var serverResponse = JsonConvert.DeserializeObject<OpcServerItemResponse>(Encoding.UTF8.GetString(serversBytes));
                if (serverResponse == null || serverResponse.Items == null || !serverResponse.Success || serverResponse.Items.Length == 0)
                {
                    Console.WriteLine("[SERVERS] Invalid or empty server response.");
                    _protocol.Errors.Add("Invalid or empty server response.");
                    return;
                }

                _protocol.ServersRetrieved = true;
                _protocol.ServerCount = serverResponse.Items.Length;

                Console.WriteLine($"[SERVERS] Found {serverResponse.Items.Length} server(s).");
                foreach (var s in serverResponse.Items)
                    Console.WriteLine($"[SERVERS]  ID={s.ID} ProgID={s.ProgID} Name={s.Name}");

                var first = serverResponse.Items[0];
                Console.WriteLine($"[SERVERS] Using server ID {first.ID} ({first.ProgID})");

                var rootTopic = $"usvc.opc.{_instance}.servers.root";
                var rootPayload = Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(new IntMessage { Value = first.ID }));
                var rootBytes = Request(rootTopic, rootPayload);
                if (rootBytes == null)
                {
                    Console.WriteLine("[BROWSE] Root request timeout.");
                    _protocol.Errors.Add("Root request timeout.");
                    return;
                }

                var rootResponse = JsonConvert.DeserializeObject<BrowserPosition>(Encoding.UTF8.GetString(rootBytes));
                if (rootResponse == null || !rootResponse.Success)
                {
                    Console.WriteLine($"[BROWSE] Root failed: {rootResponse?.StatusMessage}");
                    _protocol.Errors.Add("Root browse failed: " + rootResponse?.StatusMessage);
                    return;
                }

                _protocol.RootRetrieved = true;
                _protocol.RootPosition = rootResponse.Position;
                _protocol.RootBranches = rootResponse.Branches != null ? rootResponse.Branches.Length : 0;
                _protocol.RootLeaves = rootResponse.Leaves != null ? rootResponse.Leaves.Length : 0;

                PrintBrowseNode("[BROWSE]", rootResponse);

                var stack = new Stack<string>();
                if (rootResponse.Branches != null)
                    for (int i = rootResponse.Branches.Length - 1; i >= 0; i--)
                        stack.Push(rootResponse.Branches[i]);

                _visitedBrowsePositions.Add(rootResponse.Position ?? "(null)");

                while (stack.Count > 0 && _browseRequestsMade < _maxBrowseRequests)
                {
                    var branchId = stack.Pop();
                    if (_visitedBrowsePositions.Contains(branchId))
                        continue;

                    _browseRequestsMade++;
                    _protocol.BranchRequestsMade = _browseRequestsMade;

                    var branchReq = new GetOPCBranches
                    {
                        ServerId = rootResponse.ServerId,
                        Branch = branchId
                    };

                    var branchTopic = $"usvc.opc.{_instance}.servers.getbranch";
                    Console.WriteLine($"[BROWSE] -> Requesting branch '{branchId}' (#{_browseRequestsMade})");
                    var branchBytes = Request(branchTopic, Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(branchReq)));
                    if (branchBytes == null)
                    {
                        Console.WriteLine($"[BRANCH] Timeout for branch {branchId}");
                        _protocol.BranchRequestsFailed++;
                        _protocol.Errors.Add("Branch timeout: " + branchId);
                        continue;
                    }
                    var branchResponse = JsonConvert.DeserializeObject<BrowserPosition>(Encoding.UTF8.GetString(branchBytes));
                    if (branchResponse == null || !branchResponse.Success)
                    {
                        Console.WriteLine($"[BRANCH] FAILED branch {branchId}: {branchResponse?.StatusMessage}");
                        _protocol.BranchRequestsFailed++;
                        _protocol.Errors.Add("Branch failed: " + branchId + " msg=" + branchResponse?.StatusMessage);
                        continue;
                    }

                    _protocol.BranchRequestsSucceeded++;
                    PrintBrowseNode("[BRANCH]", branchResponse);
                    _visitedBrowsePositions.Add(branchResponse.Position ?? "(null)");

                    if (branchResponse.Branches != null)
                    {
                        for (int i = branchResponse.Branches.Length - 1; i >= 0; i--)
                        {
                            var next = branchResponse.Branches[i];
                            if (!_visitedBrowsePositions.Contains(next))
                                stack.Push(next);
                        }
                    }
                }

                if (_browseRequestsMade >= _maxBrowseRequests)
                    Console.WriteLine("[BROWSE] Reached max browse request limit.");
            }
            catch (Exception ex)
            {
                Console.WriteLine("TestBrowseRootAndNodes exception: " + ex.Message);
                _protocol.Errors.Add("Browse exception: " + ex.Message);
            }
        }

        public void RequestGetAllTagsUnfiltered()
        {
            try
            {
                var topic = $"usvc.opc.{_instance}.tags.getallunfiltered";
                var respBytes = Request(topic, new byte[0]);
                if (respBytes == null)
                {
                    Console.WriteLine("GetAllTagsUnfiltered request timed out.");
                    _protocol.Errors.Add("GetAllTagsUnfiltered timeout/null.");
                    return;
                }
                GetAllTagsUnfiltered(topic, null, respBytes);
            }
            catch (Exception ex)
            {
                Console.WriteLine("RequestGetAllTagsUnfiltered exception: " + ex.Message);
                _protocol.Errors.Add("Unfiltered tags exception: " + ex.Message);
            }
        }

        public void RequestGroupHistory()
        {
            try
            {
                var groupsTopic = $"usvc.opc.{_instance}.groups.getall";
                var groupsBytes = Request(groupsTopic, new byte[0]);
                if (groupsBytes == null)
                {
                    Console.WriteLine("[HISTORY] groups.getall timeout.");
                    _protocol.Errors.Add("Group history: groups.getall timeout/null.");
                    return;
                }

                var groupsResp = JsonConvert.DeserializeObject<OpcGroupItemResponse>(Encoding.UTF8.GetString(groupsBytes));
                if (groupsResp == null || !groupsResp.Success || groupsResp.Items == null || groupsResp.Items.Length == 0)
                {
                    Console.WriteLine("[HISTORY] Invalid groups response.");
                    _protocol.Errors.Add("Group history: invalid groups response.");
                    return;
                }

                var selected = groupsResp.Items.FirstOrDefault(g => g != null && g.State != OpcGroupState.GroupStateDisabled);
                if (selected == null)
                {
                    Console.WriteLine("[HISTORY] No suitable (enabled) group found.");
                    _protocol.Errors.Add("Group history: no suitable group.");
                    return;
                }

                var end = DateTime.UtcNow;
                var start = end - TimeSpan.FromMinutes(5);

                var req = new GroupHistoryRequest
                {
                    GroupId = selected.Id,
                    Start = start,
                    End = end
                };

                var histTopic = $"usvc.opc.{_instance}.groups.ha";
                Console.WriteLine($"[HISTORY] Requesting history for GroupId={req.GroupId} Range={req.Start:o}->{req.End:o}");
                var histBytes = Request(histTopic, Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(req)));
                if (histBytes == null)
                {
                    Console.WriteLine("[HISTORY] groups.ha timeout.");
                    _protocol.Errors.Add("Group history: request timeout/null.");
                    return;
                }

                var histResp = JsonConvert.DeserializeObject<GroupHistoryResponse>(Encoding.UTF8.GetString(histBytes));
                if (histResp == null)
                {
                    Console.WriteLine("[HISTORY] Response parse failed (null).");
                    _protocol.Errors.Add("Group history: null response after parse.");
                    return;
                }

                _protocol.HistoryStatusMessage = histResp.StatusMessage;

                if (!histResp.Success)
                {
                    Console.WriteLine("[HISTORY] Failed: " + histResp.StatusMessage);
                    _protocol.Errors.Add("Group history failed: " + histResp.StatusMessage);
                    return;
                }

                int seriesCount = histResp.Series != null ? histResp.Series.Length : 0;
                int totalPoints = 0;
                if (histResp.Series != null)
                {
                    for (int i = 0; i < histResp.Series.Length; i++)
                        totalPoints += (histResp.Series[i].Points != null) ? histResp.Series[i].Points.Length : 0;
                }

                _protocol.HistoryRetrieved = true;
                _protocol.HistoryGroupId = histResp.GroupId;
                _protocol.HistorySeriesCount = seriesCount;
                _protocol.HistoryTotalPoints = totalPoints;
                _protocol.HistoryStartUtc = histResp.Start;
                _protocol.HistoryEndUtc = histResp.End;

                Console.WriteLine($"[HISTORY] Success. Series={seriesCount} TotalPoints={totalPoints} Status='{histResp.StatusMessage}'");

                try
                {
                    var tsHist = DateTime.UtcNow.ToString("yyyyMMdd_HHmmss");
                    File.WriteAllText($"history_group_{histResp.GroupId}_{tsHist}.json",
                        Encoding.UTF8.GetString(histBytes), Encoding.UTF8);
                }
                catch (Exception fex)
                {
                    Console.WriteLine("[HISTORY] Failed writing history JSON file: " + fex.Message);
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine("[HISTORY] Exception: " + ex.Message);
                _protocol.Errors.Add("Group history exception: " + ex.Message);
            }
        }

        public void RequestTagHistory()
        {
            try
            {
                var tagsTopic = $"usvc.opc.{_instance}.tags.getall";
                var tagsBytes = Request(tagsTopic, new byte[0]);
                if (tagsBytes == null)
                {
                    Console.WriteLine("[TAGHA] tags.getall timeout.");
                    _protocol.Errors.Add("Tag history: tags.getall timeout/null.");
                    return;
                }

                var tagsResp = JsonConvert.DeserializeObject<OpcTagItemResponse>(Encoding.UTF8.GetString(tagsBytes));
                if (tagsResp == null || !tagsResp.Success || tagsResp.Items == null || tagsResp.Items.Length == 0)
                {
                    Console.WriteLine("[TAGHA] Invalid tags response.");
                    _protocol.Errors.Add("Tag history: invalid tags response.");
                    return;
                }

                var chosen = tagsResp.Items.FirstOrDefault(t => t != null);
                if (chosen == null)
                {
                    Console.WriteLine("[TAGHA] No tag available.");
                    _protocol.Errors.Add("Tag history: no tag available.");
                    return;
                }

                var end = DateTime.UtcNow;
                var start = end - TimeSpan.FromMinutes(5);

                var req = new TagHistoryRequest
                {
                    GroupId = chosen.GroupID,
                    TagName = chosen.Name,
                    Start = start,
                    End = end
                };

                var topic = $"usvc.opc.{_instance}.tags.ha";
                Console.WriteLine($"[TAGHA] Requesting tag history GroupId={req.GroupId} Tag={req.TagName} Range={req.Start:o}->{req.End:o}");
                var respBytes = Request(topic, Encoding.UTF8.GetBytes(JsonConvert.SerializeObject(req)));
                if (respBytes == null)
                {
                    Console.WriteLine("[TAGHA] tags.ha timeout.");
                    _protocol.Errors.Add("Tag history: request timeout/null.");
                    return;
                }

                var historyResp = JsonConvert.DeserializeObject<TagHistoryResponse>(Encoding.UTF8.GetString(respBytes));
                if (historyResp == null)
                {
                    Console.WriteLine("[TAGHA] Response parse failed (null).");
                    _protocol.Errors.Add("Tag history: null parsed response.");
                    return;
                }

                _protocol.TagHistoryStatusMessage = historyResp.StatusMessage;
                if (!historyResp.Success)
                {
                    Console.WriteLine("[TAGHA] Failed: " + historyResp.StatusMessage);
                    _protocol.Errors.Add("Tag history failed: " + historyResp.StatusMessage);
                    return;
                }

                _protocol.TagHistoryRetrieved = true;
                _protocol.TagHistoryName = historyResp.TagName;
                _protocol.TagHistoryGroupId = historyResp.GroupId;
                _protocol.TagHistoryPointCount = historyResp.Points != null ? historyResp.Points.Length : 0;
                _protocol.TagHistoryStartUtc = historyResp.Start;
                _protocol.TagHistoryEndUtc = historyResp.End;

                Console.WriteLine($"[TAGHA] Success. Points={_protocol.TagHistoryPointCount} Status='{historyResp.StatusMessage}'");

                try
                {
                    var tsTag = DateTime.UtcNow.ToString("yyyyMMdd_HHmmss");
                    File.WriteAllText($"history_tag_{historyResp.GroupId}_{SanitizeFileName(historyResp.TagName)}_{tsTag}.json",
                        Encoding.UTF8.GetString(respBytes), Encoding.UTF8);
                }
                catch (Exception fex)
                {
                    Console.WriteLine("[TAGHA] Failed writing tag history JSON file: " + fex.Message);
                }
            }
            catch (Exception ex)
            {
                Console.WriteLine("[TAGHA] Exception: " + ex.Message);
                _protocol.Errors.Add("Tag history exception: " + ex.Message);
            }
        }

        private string SanitizeFileName(string name)
        {
            if (string.IsNullOrEmpty(name)) return "tag";
            foreach (var c in Path.GetInvalidFileNameChars())
                name = name.Replace(c, '_');
            return name;
        }

        private void GenerateProtocolFile()
        {
            _protocol.FinishedUtc = DateTime.UtcNow;

            foreach (var p in _visitedBrowsePositions)
                _protocol.VisitedPositions.Add(p);

            var ts = DateTime.UtcNow.ToString("yyyyMMdd_HHmmss");
            if (_protocol.DataSamples.Count > 0)
            {
                var liveFile = "livedata_samples_" + ts + ".json";
                try
                {
                    var livePayload = new
                    {
                        startedUtc = _protocol.StartedUtc,
                        finishedUtc = _protocol.FinishedUtc,
                        sampleCount = _protocol.DataSamples.Count,
                        samples = _protocol.DataSamples.Select(s => new
                        {
                            t = s.Time,
                            n = s.Name,
                            v = s.Value,
                            q = s.Quality
                        }).ToArray()
                    };
                    File.WriteAllText(liveFile, JsonConvert.SerializeObject(livePayload, Formatting.Indented), Encoding.UTF8);
                    Console.WriteLine("Live data samples written to: " + liveFile);
                    _protocol.LiveDataSamplesFileName = liveFile;
                }
                catch (Exception ex)
                {
                    Console.WriteLine("Failed to write live data samples file: " + ex.Message);
                    _protocol.Errors.Add("Live samples write failed: " + ex.Message);
                    _protocol.LiveDataSamplesFileName = "(write failed)";
                }
            }
            else
            {
                _protocol.LiveDataSamplesFileName = "(no samples)";
            }

            var protoFile = "test_protocol_" + ts + ".txt";
            try
            {
                File.WriteAllText(protoFile, _protocol.ToText(), Encoding.UTF8);
                Console.WriteLine("Test protocol written to: " + protoFile);
            }
            catch (Exception ex)
            {
                Console.WriteLine("Failed to write test protocol: " + ex.Message);
            }
        }

        private void PrintBrowseNode(string prefix, BrowserPosition pos)
        {
            var branches = pos.Branches ?? new string[0];
            var leaves = pos.Leaves ?? new string[0];
            Console.WriteLine($"{prefix} ServerId={pos.ServerId} Position={pos.Position ?? "(null)"}  Branches={branches.Length}  Leaves={leaves.Length}");
            if (branches.Length > 0)
            {
                Console.WriteLine("  Branches:");
                for (int i = 0; i < branches.Length; i++)
                    Console.WriteLine("    [" + i + "] " + branches[i]);
            }
            if (leaves.Length > 0)
            {
                Console.WriteLine("  Leaves (time series candidates):");
                foreach (var leaf in leaves.Take(20))
                    Console.WriteLine("    " + leaf);
                if (leaves.Length > 20)
                    Console.WriteLine($"    ... ({leaves.Length - 20} more)");
            }
        }

        private DdUsvcError GetAllTagsUnfiltered(string topic, string responsetopic, byte[] data)
        {
            Console.WriteLine("Processing GetAllTagsUnfiltered response...");
            try
            {
                var parsedData = Encoding.UTF8.GetString(data);
                var tagNamesResponse = JsonConvert.DeserializeObject<OpcTagItemMetaResponse>(parsedData);
                if (tagNamesResponse == null || tagNamesResponse.Items == null)
                    throw new InvalidOperationException("Received empty tag metadata response.");

                var incomingTags = tagNamesResponse.Items;

                _protocol.UnfilteredTagsRetrieved = true;
                _protocol.UnfilteredTagCount = incomingTags.Length;

                var csvConfig = new CsvConfiguration(CultureInfo.InvariantCulture)
                {
                    Delimiter = ";",
                    Encoding = Encoding.UTF8,
                    HasHeaderRecord = true,
                    Quote = '"'
                };

                var groups = new List<GroupRecord>();
                var tagRecords = new List<TagRecord>();

                int groupSize = 10;
                int groupId = 1;
                int countInGroup = 0;
                string progId = "default";

                groups.Add(new GroupRecord
                {
                    GroupId = groupId,
                    Name = "auto_group_" + groupId,
                    SamplingTime = 1,
                    ProgId = progId,
                    Default = 1,
                    RunAtStart = 1
                });
                Console.WriteLine(incomingTags.Length + " tags");

                foreach (var tag in incomingTags)
                {
                    tagRecords.Add(new TagRecord
                    {
                        Name = tag.Name,
                        GroupId = groupId
                    });

                    countInGroup++;
                    if (countInGroup >= groupSize)
                    {
                        groupId++;
                        countInGroup = 0;
                        groups.Add(new GroupRecord
                        {
                            GroupId = groupId,
                            Name = "auto_group_" + groupId,
                            SamplingTime = 1,
                            ProgId = progId,
                            Default = 0,
                            RunAtStart = 1
                        });
                    }
                }

                using (var writer = new StreamWriter("groups_auto.csv", false, Encoding.UTF8))
                using (var csv = new CsvWriter(writer, csvConfig))
                {
                    csv.Context.RegisterClassMap<GroupRecordMap>();
                    csv.WriteHeader<GroupRecord>();
                    csv.NextRecord();
                    foreach (var g in groups)
                    {
                        csv.WriteRecord(g);
                        csv.NextRecord();
                    }
                }

                using (var writer = new StreamWriter("tags_auto.csv", false, Encoding.UTF8))
                using (var csv = new CsvWriter(writer, csvConfig))
                {
                    csv.Context.RegisterClassMap<TagRecordMap>();
                    csv.WriteHeader<TagRecord>();
                    csv.NextRecord();
                    foreach (var t in tagRecords)
                    {
                        csv.WriteRecord(t);
                        csv.NextRecord();
                    }
                }

                Console.WriteLine("Created default group and tag CSV files (CsvHelper).");
                return new DdUsvcError { Code = DdUsvcErrorCode.OK, Reason = "OK" };
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error processing GetAllTagsUnfiltered: {ex.Message}");
                _protocol.Errors.Add("Unfiltered tags processing error: " + ex.Message);
                return new DdUsvcError { Code = DdUsvcErrorCode.Error, Reason = ex.Message };
            }
        }

        private DdUsvcError OnServerStatus(string topic, string responsetopic, byte[] data)
        {
            try
            {
                var json = Encoding.UTF8.GetString(data);
                var status = JsonConvert.DeserializeObject<ServerStatus>(json);
                _protocol.BroadcastServerStatusMessages++;
                Console.WriteLine($"[STATUS] {status.ProgId} state={status.State} last={status.LastUpdate}");
            }
            catch (Exception ex)
            {
                Console.WriteLine("OnServerStatus parse error: " + ex.Message);
                _protocol.Errors.Add("ServerStatus parse: " + ex.Message);
            }
            return new DdUsvcError { Code = DdUsvcErrorCode.OK };
        }

        private DdUsvcError OnDataPoint(string topic, string responsetopic, byte[] data)
        {
            try
            {
                var json = Encoding.UTF8.GetString(data);
                var point = JsonConvert.DeserializeObject<DataPoint>(json);
                _protocol.BroadcastDataPoints++;

                lock (_sampleLock)
                {
                    _protocol.TryAddSample(new DataPointSample
                    {
                        Time = point.Time,
                        Name = point.Name,
                        Value = point.Value,
                        Quality = point.Quality
                    });
                }

                Console.WriteLine($"[DATA] {point.Time:o} {point.Name}={point.Value} q={point.Quality}");
            }
            catch (Exception ex)
            {
                Console.WriteLine("OnDataPoint parse error: " + ex.Message);
                _protocol.Errors.Add("DataPoint parse: " + ex.Message);
            }
            return new DdUsvcError { Code = DdUsvcErrorCode.OK };
        }

        private DdUsvcError OnPeriodicTagList(string topic, string responsetopic, byte[] data)
        {
            try
            {
                var json = Encoding.UTF8.GetString(data);
                var meta = JsonConvert.DeserializeObject<OpcTagMetaInfo[]>(json);
                _protocol.BroadcastPeriodicTagLists++;
                Console.WriteLine($"[TAGLIST] Hourly unfiltered tags count={(meta == null ? 0 : meta.Length)}");
            }
            catch (Exception ex)
            {
                Console.WriteLine("OnPeriodicTagList parse error: " + ex.Message);
                _protocol.Errors.Add("PeriodicTagList parse: " + ex.Message);
            }
            return new DdUsvcError { Code = DdUsvcErrorCode.OK };
        }
    }

    internal class DataPointSample
    {
        public DateTime Time { get; set; }
        public string Name { get; set; }
        public double Value { get; set; }
        public int Quality { get; set; }
    }

    internal class TestProtocolResult
    {
        private const int MaxSamples = 25;

        public DateTime StartedUtc { get; set; }
        public DateTime FinishedUtc { get; set; }

        public bool ServersRetrieved { get; set; }
        public int ServerCount { get; set; }

        public bool RootRetrieved { get; set; }
        public string RootPosition { get; set; }
        public int RootBranches { get; set; }
        public int RootLeaves { get; set; }

        public int BranchRequestsMade { get; set; }
        public int BranchRequestsSucceeded { get; set; }
        public int BranchRequestsFailed { get; set; }
        public HashSet<string> VisitedPositions { get; } = new HashSet<string>(StringComparer.OrdinalIgnoreCase);

        public bool UnfilteredTagsRetrieved { get; set; }
        public int UnfilteredTagCount { get; set; }

        public bool HistoryRetrieved { get; set; }
        public int HistoryGroupId { get; set; }
        public int HistorySeriesCount { get; set; }
        public int HistoryTotalPoints { get; set; }
        public DateTime HistoryStartUtc { get; set; }
        public DateTime HistoryEndUtc { get; set; }
        public string HistoryStatusMessage { get; set; }

        public bool TagHistoryRetrieved { get; set; }
        public string TagHistoryName { get; set; }
        public int TagHistoryGroupId { get; set; }
        public int TagHistoryPointCount { get; set; }
        public DateTime TagHistoryStartUtc { get; set; }
        public DateTime TagHistoryEndUtc { get; set; }
        public string TagHistoryStatusMessage { get; set; }

        public int BroadcastServerStatusMessages { get; set; }
        public int BroadcastDataPoints { get; set; }
        public int BroadcastPeriodicTagLists { get; set; }

        internal List<DataPointSample> DataSamples { get; } = new List<DataPointSample>();
        public int LiveDataSampleCount => DataSamples.Count;
        public string LiveDataSamplesFileName { get; set; }

        public bool GroupStartStopTestPerformed { get; set; }
        public int GroupStartStopGroupId { get; set; }
        public string GroupInitialState { get; set; }
        public string GroupStateAfterStop { get; set; }
        public string GroupStateAfterStart { get; set; }
        public bool GroupStopVerified { get; set; }
        public bool GroupStartVerified { get; set; }
        public string GroupStartStopStatusMessage { get; set; }

        public List<string> Errors { get; } = new List<string>();

        public void TryAddSample(DataPointSample sample)
        {
            if (sample == null) return;
            if (DataSamples.Count < MaxSamples)
                DataSamples.Add(sample);
        }

        public string ToText()
        {
            var sb = new StringBuilder();
            sb.AppendLine("==== dd-opcua-test protocol ====");
            sb.AppendLine("Started (UTC): " + StartedUtc.ToString("o"));
            sb.AppendLine("Finished (UTC): " + FinishedUtc.ToString("o"));
            sb.AppendLine("Duration: " + (FinishedUtc - StartedUtc));
            sb.AppendLine();
            sb.AppendLine("SERVERS:");
            sb.AppendLine($"  Retrieved: {ServersRetrieved}");
            sb.AppendLine($"  Count: {ServerCount}");
            sb.AppendLine();
            sb.AppendLine("ROOT:");
            sb.AppendLine($"  Retrieved: {RootRetrieved}");
            sb.AppendLine($"  Position: {RootPosition}");
            sb.AppendLine($"  Branches: {RootBranches}");
            sb.AppendLine($"  Leaves: {RootLeaves}");
            sb.AppendLine();
            sb.AppendLine("BROWSE (getbranch):");
            sb.AppendLine($"  Requests made: {BranchRequestsMade}");
            sb.AppendLine($"  Succeeded: {BranchRequestsSucceeded}");
            sb.AppendLine($"  Failed: {BranchRequestsFailed}");
            sb.AppendLine($"  Unique positions visited: {VisitedPositions.Count}");
            sb.AppendLine();
            sb.AppendLine("UNFILTERED TAGS:");
            sb.AppendLine($"  Retrieved: {UnfilteredTagsRetrieved}");
            sb.AppendLine($"  Tag count: {UnfilteredTagCount}");
            sb.AppendLine();
            sb.AppendLine("GROUP START/STOP TEST:");
            sb.AppendLine($"  Performed: {GroupStartStopTestPerformed}");
            if (GroupStartStopTestPerformed)
            {
                sb.AppendLine($"  GroupId: {GroupStartStopGroupId}");
                sb.AppendLine($"  Initial: {GroupInitialState}");
                sb.AppendLine($"  After stop: {GroupStateAfterStop}  (verified={GroupStopVerified})");
                sb.AppendLine($"  After start: {GroupStateAfterStart}  (verified={GroupStartVerified})");
                sb.AppendLine($"  Status: {GroupStartStopStatusMessage}");
            }
            sb.AppendLine();
            sb.AppendLine("HISTORICAL GROUP (groups.ha):");
            sb.AppendLine($"  Retrieved: {HistoryRetrieved}");
            if (HistoryRetrieved)
            {
                sb.AppendLine($"  GroupId: {HistoryGroupId}");
                sb.AppendLine($"  Time range: {HistoryStartUtc:o} -> {HistoryEndUtc:o}");
                sb.AppendLine($"  Series count: {HistorySeriesCount}");
                sb.AppendLine($"  Total points: {HistoryTotalPoints}");
            }
            sb.AppendLine($"  Status: {HistoryStatusMessage}");
            sb.AppendLine();
            sb.AppendLine("HISTORICAL TAG (tags.ha):");
            sb.AppendLine($"  Retrieved: {TagHistoryRetrieved}");
            if (TagHistoryRetrieved)
            {
                sb.AppendLine($"  GroupId: {TagHistoryGroupId}");
                sb.AppendLine($"  Tag: {TagHistoryName}");
                sb.AppendLine($"  Time range: {TagHistoryStartUtc:o} -> {TagHistoryEndUtc:o}");
                sb.AppendLine($"  Points: {TagHistoryPointCount}");
            }
            sb.AppendLine($"  Status: {TagHistoryStatusMessage}");
            sb.AppendLine();
            sb.AppendLine("LIVE DATA SAMPLES:");
            sb.AppendLine($"  Captured (first N): {LiveDataSampleCount}");
            sb.AppendLine($"  JSON file: {LiveDataSamplesFileName ?? "(not written)"}");
            sb.AppendLine();
            sb.AppendLine("BROADCAST COUNTS:");
            sb.AppendLine($"  Server status msgs: {BroadcastServerStatusMessages}");
            sb.AppendLine($"  Data points: {BroadcastDataPoints}");
            sb.AppendLine($"  Periodic tag lists: {BroadcastPeriodicTagLists}");
            sb.AppendLine();
            sb.AppendLine("ERRORS:");
            if (Errors.Count == 0)
                sb.AppendLine("  (none)");
            else
                foreach (var e in Errors)
                    sb.AppendLine("  - " + e);
            sb.AppendLine("================================");
            return sb.ToString();
        }
    }

    internal class GroupRecord
    {
        public int GroupId { get; set; }
        public string Name { get; set; }
        public int SamplingTime { get; set; }
        public string ProgId { get; set; }
        public int Default { get; set; }
        public int RunAtStart { get; set; }
    }

    internal sealed class GroupRecordMap : ClassMap<GroupRecord>
    {
        public GroupRecordMap()
        {
            Map(m => m.GroupId).Name("groupid");
            Map(m => m.Name).Name("name");
            Map(m => m.SamplingTime).Name("samplingtime");
            Map(m => m.ProgId).Name("progid");
            Map(m => m.Default).Name("default");
            Map(m => m.RunAtStart).Name("runatstart");
        }
    }

    internal class TagRecord
    {
        public string Name { get; set; }
        public int GroupId { get; set; }
    }

    internal sealed class TagRecordMap : ClassMap<TagRecord>
    {
        public TagRecordMap()
        {
            Map(m => m.Name).Name("name");
            Map(m => m.GroupId).Name("groupid");
        }
    }
}