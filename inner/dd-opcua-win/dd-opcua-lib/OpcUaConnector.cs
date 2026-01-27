using dd_opcua_lib;
using DdOpcUaLib;
using Microsoft.Extensions.Logging;
using Opc.Ua;
using Opc.Ua.Client;
using Opc.Ua.Configuration;
using System;
using System.Collections.Generic;
using System.Linq;
using System.Text;
using System.Threading.Tasks;
using static System.Net.Mime.MediaTypeNames;

namespace dd_opcua_lib
{

    public class OpcUaConnectionStatus
    {
        public bool Connected { get; set; }
        public string EndpointUrl { get; set; }
        public DateTime LastContactTime { get; set; }
    }

    public class OpcUaConnection : IDisposable
    {
        public Session OpcSession { get; set; } = null;
        private static ApplicationInstance Application { get; set; } = null;
        private static ApplicationConfiguration Config { get; set; } = null;
        private static string ClientName = "aCurve OPC UA (HA) Client";

        public static ApplicationConfiguration GetApplicationConfiguration(string clientName = "aCurve OPC UA (HA) Client")
        {
            if (Config != null)
            {
                return Config;
            }
            ClientName = clientName;
            string certStorePath = Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData) + @"\OpcUaConnector\pki\";

            var config = new ApplicationConfiguration()
            {
                ApplicationName = clientName,
                ApplicationType = ApplicationType.Client,
                SecurityConfiguration = new SecurityConfiguration
                {
                    AutoAcceptUntrustedCertificates = true,
                    AddAppCertToTrustedStore = true,
                    // ApplicationCertificate = new CertificateIdentifier { StoreType = "Directory", StorePath = certStorePath + "own", SubjectName = "CN=dd-nats Opc Ua connector, C=SE, O=cyops, DC=localhost" },
                    ApplicationCertificates = ApplicationConfigurationBuilder.CreateDefaultApplicationCertificates( "CN=dd-nats Opc Ua connector, C=SE, O=cyops, DC=localhost", CertificateStoreType.Directory, certStorePath + @"own"),
                    TrustedPeerCertificates = new CertificateTrustList { StoreType = "Directory", StorePath = certStorePath + "trusted" },
                    TrustedIssuerCertificates = new CertificateTrustList { StoreType = "Directory", StorePath = certStorePath + "issuers" },
                    RejectedCertificateStore = new CertificateTrustList { StoreType = "Directory", StorePath = certStorePath + "rejected" }
                },
                TransportConfigurations = new TransportConfigurationCollection(),
                TransportQuotas = new TransportQuotas { OperationTimeout = 15000 },
                ClientConfiguration = new ClientConfiguration { DefaultSessionTimeout = 60000 }
            };
            config.Validate(ApplicationType.Client).GetAwaiter().GetResult();
            Config = config;

            Application = new ApplicationInstance(config);
            Application.CheckApplicationInstanceCertificatesAsync(false).AsTask().Wait();
            return config;
        }

        public bool ConnectToServer(string serverUrl)
        {
            string originalInput = serverUrl;
            try
            {
                if (!string.IsNullOrWhiteSpace(serverUrl) &&
                    !serverUrl.StartsWith("urn:", StringComparison.OrdinalIgnoreCase) &&
                    serverUrl.IndexOf("://", StringComparison.Ordinal) < 0)
                {
                    if (serverUrl.IndexOf(':') >= 0)
                        serverUrl = "opc.tcp://" + serverUrl.Trim();
                    else
                        serverUrl = "opc.tcp://" + serverUrl.Trim() + ":4840";
                }

                Uri uri;
                if (!Uri.TryCreate(serverUrl, UriKind.Absolute, out uri) ||
                    !string.Equals(uri.Scheme, "opc.tcp", StringComparison.OrdinalIgnoreCase))
                {
                    Console.WriteLine(
                        $"Error connecting to OPC UA server: Invalid endpoint '{originalInput}'. " +
                        $"Normalized='{serverUrl}'. Expected format: opc.tcp://host:port[/path]");
                    OpcSession = null;
                    return false;
                }


                var config = GetApplicationConfiguration();

                EndpointDescription endpoint;
                try
                {
                    endpoint = CoreClientUtils.SelectEndpoint(config, serverUrl, false);
                }
                catch (Exception epSelEx)
                {
                    Console.WriteLine($"Error connecting to OPC UA server: Failed to select endpoint for '{serverUrl}'. {epSelEx.Message}");
                    OpcSession = null;
                    return false;
                }

                var session = Session.Create(
                    config,
                    new ConfiguredEndpoint(null, endpoint, EndpointConfiguration.Create(config)),
                    false,
                    ClientName,
                    60000,
                    null,
                    null
                ).GetAwaiter().GetResult();

                OpcSession = session;
                session.SessionClosing += Session_SessionClosing;
                return session != null && session.Connected;
            }
            catch (Exception ex)
            {
                Console.WriteLine($"Error connecting to OPC UA server: {ex.Message} (input='{originalInput}')");
                OpcSession = null;
                return false;
            }
        }

        private void Session_SessionClosing(object sender, EventArgs e)
        {
            Console.WriteLine($"Sesssion closing event recevied.");
        }

        public void Dispose()
        {
            if (OpcSession != null && OpcSession.Connected)
            {
                OpcSession.Close();
                OpcSession.Dispose();
                OpcSession = null;
            }
        }

        public OpcUaConnectionStatus GetStatus()
        {
            if (OpcSession != null)
            {
                return new OpcUaConnectionStatus
                {
                    Connected = OpcSession.Connected,
                    EndpointUrl = OpcSession.Endpoint != null ? OpcSession.Endpoint.EndpointUrl : null,
                    LastContactTime = OpcSession.LastKeepAliveTime
                };
            }
            return null;
        }
    }

    public class OpcUaHistoricalValues
    {

        public IList<DataValue> ReadHistoricalData(Session session, string nodeId, DateTime startTime, DateTime endTime)
        {
            var historyReadDetails = new ReadRawModifiedDetails
            {
                IsReadModified = false,
                StartTime = startTime.ToUniversalTime(),
                EndTime = endTime.ToUniversalTime(),
                NumValuesPerNode = uint.MaxValue,
                ReturnBounds = true
            };

            var nodesToRead = new HistoryReadValueIdCollection
            {
                new HistoryReadValueId
                {
                    NodeId = new NodeId(nodeId),
                    IndexRange = null,
                    DataEncoding = null
                }
            };

            var results = new HistoryReadResultCollection();
            var diagnosticInfos = new DiagnosticInfoCollection();

            session.HistoryRead(
                null,
                new ExtensionObject(historyReadDetails),
                TimestampsToReturn.Both,
                false,
                nodesToRead,
                out results,
                out diagnosticInfos);

            var values = new List<DataValue>();

            if (results.Count > 0 && results[0].HistoryData != null)
            {
                var historyData = ExtensionObject.ToEncodeable(results[0].HistoryData) as HistoryData;
                if (historyData != null)
                {
                    foreach (var dv in historyData.DataValues)
                    {
                        values.Add(dv);
                    }
                }
            }
            return values;
        }
    }

    public class OpcUaSubscription
    {
        public OpcUaSubscription(DdOpcUaUsvc ddOpcUaUsvc)
        {
            Service = ddOpcUaUsvc;
        }
        private OpcUaSubscription() { }
        public float PercentDeadband { get; internal set; }
        public DdOpcUaUsvc Service { get; }
        private readonly Dictionary<int, Subscription> _groupSubscriptions = new Dictionary<int, Subscription>();
        private readonly Dictionary<int, Session> _groupSessions = new Dictionary<int, Session>();

        public void AddTagsToGroupSubscription(OpcGroupItem group)
        {
            Service.LogEvent($"AddTagsToGroupSubscription called for group {group?.Id}");
            Service.LogEvent($"Current tracked subscriptions: {_groupSubscriptions.Count}, sessions: {_groupSessions.Count}");
            Service.LogEvent($"Current tags in group: {group?.tags.Count}");
            Service.LogEvent($"Group state: {group?.State}");
            if (group == null) return;
            if (group.tags == null || group.tags.Count == 0) return;
            if(group.State == OpcGroupState.GroupStateDisabled)
            {
                return;
            }
            Subscription subscription;
            if (!_groupSubscriptions.TryGetValue(group.Id, out subscription))
            {
                return;
            }
            var existingNames = new HashSet<string>(
                subscription.MonitoredItems.Select(mi => mi.DisplayName),
                StringComparer.OrdinalIgnoreCase);
            var newItems = new List<MonitoredItem>();
            foreach (var tag in group.tags)
            {
                if (tag == null) continue;
                if (string.IsNullOrWhiteSpace(tag.Name)) continue;

                if (existingNames.Contains(tag.Name)) continue;

                NodeId nodeId;
                try
                {
                    nodeId = NodeId.Parse(tag.Name);
                }
                catch
                {
                    nodeId = new NodeId(tag.Name);
                }

                var mi = new MonitoredItem(subscription.DefaultItem)
                {
                    StartNodeId = nodeId,
                    DisplayName = tag.Name,
                    SamplingInterval = subscription.PublishingInterval,
                    QueueSize = 1,
                    DiscardOldest = true,
                    MonitoringMode = MonitoringMode.Reporting
                };

                var capturedTag = tag;
                mi.Notification += (monItem, args) => HandleNotification(capturedTag, (MonitoredItem)monItem, args);
                Service.LogEvent($"Adding monitored item '{mi.DisplayName}' to group '{subscription.DisplayName}'");

                newItems.Add(mi);
            }
            var groupTagNameSet = new HashSet<string>(
                group.tags.Where(t => t != null && !string.IsNullOrWhiteSpace(t.Name))
                          .Select(t => t.Name),
                StringComparer.OrdinalIgnoreCase);

            var removeItems = subscription.MonitoredItems
                                          .Where(mi => !groupTagNameSet.Contains(mi.DisplayName))
                                          .ToList();

            if (newItems.Count == 0 && removeItems.Count == 0)
                return;
            if (newItems.Count > 0)
            {
                subscription.AddItems(newItems);
            }
            if (removeItems.Count > 0)
            {
                subscription.RemoveItems(removeItems);
            }

            subscription.ApplyChanges();
        }

        public Subscription GroupSubscription(Session session, OpcGroupItem group)
        {
            if (session == null) throw new ArgumentNullException(nameof(session));
            if (!session.Connected) throw new InvalidOperationException("OPC UA session is not connected.");
            if (group == null) throw new ArgumentNullException(nameof(group));
            if (group.tags == null || group.tags.Count == 0) return null;
            group.subscription = this;

            var intervalMs = group.Interval <= 0 ? 1000 : group.Interval;

            const double TargetInactivityToleranceDays = 7.0;
            var targetInactivityMs = TargetInactivityToleranceDays * 24d * 60d * 60d * 1000d;
            var desiredLifetimeCountDouble = targetInactivityMs / intervalMs;
            if (desiredLifetimeCountDouble > uint.MaxValue - 1)
                desiredLifetimeCountDouble = uint.MaxValue - 1;

            var lifetimeCount = (uint)desiredLifetimeCountDouble;
            uint keepAliveCount = lifetimeCount / 100u;
            if (keepAliveCount < 20u) keepAliveCount = 20u;         
            if (keepAliveCount > 5000u) keepAliveCount = 5000u;
            if (lifetimeCount < keepAliveCount * 3u)
            {
                lifetimeCount = keepAliveCount * 3u;
            }

            var subscription = new Subscription(session.DefaultSubscription)
            {
                DisplayName = string.IsNullOrWhiteSpace(group.Name) ? $"Group_{group.Id}" : group.Name,
                PublishingInterval = intervalMs,
                PublishingEnabled = true,
                LifetimeCount = lifetimeCount,
                KeepAliveCount = keepAliveCount,
                MaxNotificationsPerPublish = 0,
                Priority = 0
            };

            Service.LogEvent($"Requesting subscription: Interval={intervalMs}ms KeepAliveCount={subscription.KeepAliveCount} LifetimeCount={subscription.LifetimeCount}");

            var monitoredItems = new List<MonitoredItem>();
            foreach (var tag in group.tags)
            {
                if (tag == null) continue;
                if (string.IsNullOrWhiteSpace(tag.Name)) continue;

                NodeId nodeId;
                try
                {
                    nodeId = NodeId.Parse(tag.Name);
                }
                catch (Exception ex)
                {
                    Service.LogError($"Error parsing NodeId '{tag.Name}' for tag '{tag.Name}': {ex.Message}");
                    continue;
                }

                var mi = new MonitoredItem(subscription.DefaultItem)
                {
                    StartNodeId = nodeId,
                    DisplayName = tag.Name,
                    SamplingInterval = intervalMs,
                    QueueSize = 1,
                    DiscardOldest = true,
                    MonitoringMode = MonitoringMode.Reporting
                };

                var capturedTag = tag;
                mi.Notification += (monItem, args) => HandleNotification(capturedTag, (MonitoredItem)monItem, args);
                Service.LogEvent($"Adding monitored item '{mi.DisplayName}' to group '{subscription.DisplayName}'");
                monitoredItems.Add(mi);
            }

            if (monitoredItems.Count == 0)
                return null;

            subscription.AddItems(monitoredItems);
            session.AddSubscription(subscription);
            subscription.Create();

            Service.LogEvent($"Revised subscription: Interval={subscription.CurrentPublishingInterval}ms KeepAliveCount={subscription.KeepAliveCount} LifetimeCount={subscription.LifetimeCount}");

            _groupSubscriptions[group.Id] = subscription;
            _groupSessions[group.Id] = session;
            group.State = OpcGroupState.GroupStateRunning;
            return subscription;
        }

        internal void RefreshState()
        {
            throw new NotImplementedException();
        }

        private void HandleNotification(OpcTagItem tagRef, MonitoredItem monitoredItem, MonitoredItemNotificationEventArgs args)
        {
            // Service.LogEvent($"Notification received for tag '{tagRef?.Name}' (NodeId: {tagRef?.Name})");
            if (tagRef == null) return;

            try
            {
                var notification = args.NotificationValue as MonitoredItemNotification;
                if (notification == null) return;

                var dv = notification.Value;
                if (dv == null) return;

                tagRef.Time = dv.SourceTimestamp != DateTime.MinValue ? dv.SourceTimestamp : dv.ServerTimestamp;
                tagRef.Quality = (int)dv.StatusCode.Code;

                double newVal;
                var val = dv.Value;

                if (val is double d) newVal = d;
                else if (val is float f) newVal = f;
                else if (val is int i) newVal = i;
                else if (val is long l) newVal = l;
                else if (val is uint ui) newVal = ui;
                else if (val is short s) newVal = s;
                else if (val is ushort us) newVal = us;
                else if (val is byte b) newVal = b;
                else if (val is sbyte sb) newVal = sb;
                else if (val is decimal dec) newVal = (double)dec;
                else
                {
                    if (val == null || !double.TryParse(val.ToString(), out newVal))
                        return; 
                }

                tagRef.Value = newVal;
                tagRef.Error = 0;
            }
            catch
            {
                tagRef.Error = 1;
            }
            Service.DataChanged(tagRef, monitoredItem);
        }

        public void StopSubscription(OpcGroupItem group)
        {
            if (group == null) return;

            Subscription subscription;
            Session session;

            if (!_groupSubscriptions.TryGetValue(group.Id, out subscription) ||
                !_groupSessions.TryGetValue(group.Id, out session) ||
                subscription == null)
            {
                return;
            }

            try
            {
                try
                {
                    if (session != null && session.Connected && subscription.Created)
                    {
                        subscription.SetPublishingMode(false);
                    }
                }
                catch { 
                }

                if (session != null && session.Subscriptions.Contains(subscription))
                {
                    session.RemoveSubscription(subscription);
                }

                try
                {
                    if (subscription.Created)
                    {
                        subscription.Delete(true);
                    }
                }
                catch (Exception exDel)
                {
                    Service?.LogError($"OPC UA delete subscription failed for group {group.Id}: {exDel.Message}");
                }
            }
            catch (Exception ex)
            {
                Service?.LogError($"OPC UA stop subscription error for group {group.Id}: {ex.Message}");
            }
            finally
            {
                _groupSubscriptions.Remove(group.Id);
                _groupSessions.Remove(group.Id);

                if (group.State != OpcGroupState.GroupStateDisabled)
                {
                    group.State = OpcGroupState.GroupStateStopped;
                }
            }
        }
    }

    public class OpcUaTagBrowser
    {

        public List<TagInfo> BrowseTags(Session session)
        {
            var tags = new List<TagInfo>();
            if (session == null)
                return tags;
            if (!session.Connected)
                return tags;
            try
            {
                // BrowseTagsRecursive(session, ObjectIds.ObjectsFolder, tags);
                Console.WriteLine($"Browsing tags recursively beginning at ns=2;s=Path");
                var nodeid = TryParseNodeId("ns=2;s=Path");
                BrowseTagsRecursive(session, nodeid, tags);
            }
            catch (Exception e)
            {
                
            }
            return tags;
        }

        private bool IsTimeSeriesVariable(Session session, NodeId nodeId, ReferenceDescription rd)
        {
            try
            {
                if (nodeId.NamespaceIndex == 0)
                    return false;

                var node = session.ReadNode(nodeId) as VariableNode;
                if (node == null)
                    return false;

                return node.Historizing;

                //if (node.TypeDefinitionId != null)
                //{
                //    var typeDefId = ExpandedNodeId.ToNodeId(node.TypeDefinitionId, session.NamespaceUris);
                //    if (typeDefId == VariableTypeIds.PropertyType)
                //        return false;
                //}

                //if (node.ValueRank != ValueRanks.Scalar && node.ValueRank != ValueRanks.Any && node.ValueRank != ValueRanks.ScalarOrOneDimension)
                //    return false;

                //var dt = node.DataType;
                //if (dt == null) return false;

                //var allowed = _allowedDataTypes;
                //if (!allowed.Contains(dt))
                //    return false;

                //return true;
            }
            catch
            {
                return false;
            }
        }

        private static readonly HashSet<NodeId> _allowedDataTypes = new HashSet<NodeId>
        {
            DataTypeIds.Boolean,
            DataTypeIds.SByte,
            DataTypeIds.Byte,
            DataTypeIds.Int16,
            DataTypeIds.UInt16,
            DataTypeIds.Int32,
            DataTypeIds.UInt32,
            DataTypeIds.Int64,
            DataTypeIds.UInt64,
            DataTypeIds.Float,
            DataTypeIds.Double,
            DataTypeIds.Decimal,
            DataTypeIds.String,
            DataTypeIds.DateTime
        };

        private static string CanonicalNodeIdFromReference(Session session, ReferenceDescription rd)
        {
            var nodeId = ExpandedNodeId.ToNodeId(rd.NodeId, session.NamespaceUris);
            if (nodeId == null) return null;
            if (nodeId.NamespaceIndex == 0)
                return "ns=0;" + FormatNodeIdIdentifier(nodeId);
            return nodeId.ToString();
        }

        private static string FormatNodeIdIdentifier(NodeId id)
        {
            

            switch (id.IdType)
            {
                case IdType.Numeric: return "i=" + id.Identifier;
                case IdType.String:  return "s=" + id.Identifier;
                case IdType.Guid:    return "g=" + id.Identifier;
                case IdType.Opaque:  return "b=" + Convert.ToBase64String((byte[])id.Identifier);
                default: return id.ToString();
            }
        }

        Dictionary<string, TagInfo> _ledger = new Dictionary<string, TagInfo>();
        private void BrowseTagsRecursive(Session session, NodeId nodeId, List<TagInfo> tags)
        {
            if (tags.Count > 50) return;
            if (session == null) return;
            Browser browser = null;
            try
            {
                browser = new Browser(session)
                {
                    BrowseDirection = BrowseDirection.Forward,
                    NodeClassMask = (int)NodeClass.Object | (int)NodeClass.Variable,
                    ReferenceTypeId = ReferenceTypeIds.HierarchicalReferences,
                    IncludeSubtypes = true
                };
            }
            catch (Exception ex) {
                Console.WriteLine($"Error while browsing tags recursively. Failed to create browser from session: {ex.Message}");
                return;
            }

            ReferenceDescriptionCollection refs = null;
            try { refs = browser.Browse(nodeId); } catch (Exception ex)
            {
                Console.WriteLine($"Error while browsing tags recursively. Failed to create browser from session: {ex.Message}");
                return;
            }

            if (refs == null)
            {
                Console.WriteLine($"Error while browsing tags recursively. refs == null");
                return;
            }

            for (int r = 0; r < refs.Count; r++)
            {
                var rd = refs[r];
                NodeId childId;
                try
                {
                    childId = ExpandedNodeId.ToNodeId(rd.NodeId, session.NamespaceUris);
                }
                catch (Exception ex)
                {
                    Console.WriteLine($"Error while browsing tags recursively. ToNodeId failed: {ex.Message}");
                    continue;
                }

                if ((rd.NodeClass & NodeClass.Variable) != 0 && IsTimeSeriesVariable(session, childId, rd))
                {
                    try
                    {
                        var cni = CanonicalNodeIdFromReference(session, rd);
                        if (!_ledger.ContainsKey(cni))
                        {
                            var tag = new TagInfo
                            {
                                NodeId = CanonicalNodeIdFromReference(session, rd),
                                //Name = CanonicalNodeIdFromReference(session, rd),
                                //Description = SafeGetDescription(session, childId),
                                //EngineeringUnit = SafeGetEngineeringUnit(session, childId),
                                //MinValue = SafeGetPropertyDouble(session, childId, "EURange", true),
                                //MaxValue = SafeGetPropertyDouble(session, childId, "EURange", false)
                            };
                            tag.Name = tag.NodeId;
                            tags.Add(tag);
                            _ledger.Add(cni, tag);
                            Console.WriteLine($"Tag added: {tag.NodeId}");
                        }
                        continue; // Don't bother with children to this node
                    }
                    catch (Exception ex)
                    {
                        Console.WriteLine($"Error while browsing tags recursively. Failed to create or add TagInfo object: {ex.Message}");
                        continue;
                    }
                }

                if ((rd.NodeClass & NodeClass.Object) != 0)
                {
                    BrowseTagsRecursive(session, childId, tags);
                }
            }
        }

        private string SafeGetDescription(Session session, NodeId nodeId)
        {
            try
            {
                var desc = session.ReadNode(nodeId).Description;
                return desc != null ? desc.Text : string.Empty;
            }
            catch { return string.Empty; }
        }

        private string SafeGetEngineeringUnit(Session session, NodeId nodeId)
        {
            try
            {
                var euNode = FindPropertyNode(session, nodeId, "EngineeringUnits");
                if (euNode != null)
                {
                    var euNodeId = ExpandedNodeId.ToNodeId(euNode.NodeId, session.NamespaceUris);
                    var value = session.ReadValue(euNodeId);
                    if (value.Value is ExtensionObject extObj && extObj.Body is EUInformation euInfo)
                        return euInfo.DisplayName.Text;
                }
            }
            catch { }
            return string.Empty;
        }

        private double? SafeGetPropertyDouble(Session session, NodeId nodeId, string propertyName, bool getLow)
        {
            try
            {
                var propNode = FindPropertyNode(session, nodeId, propertyName);
                if (propNode != null)
                {
                    var propNodeId = ExpandedNodeId.ToNodeId(propNode.NodeId, session.NamespaceUris);
                    var value = session.ReadValue(propNodeId);
                    if (value.Value is ExtensionObject extObj && extObj.Body is Opc.Ua.Range range)
                        return getLow ? range.Low : range.High;
                }
            }
            catch { }
            return null;
        }

        private ReferenceDescription FindPropertyNode(Session session, NodeId nodeId, string browseName)
        {
            if (session == null) return null;
            Browser browser = null;
            try
            {
                browser = new Browser(session)
                {
                    BrowseDirection = BrowseDirection.Forward,
                    NodeClassMask = (int)NodeClass.Variable,
                    ReferenceTypeId = ReferenceTypeIds.HasProperty,
                    IncludeSubtypes = true
                };
            }
            catch
            {
                return null;
            }

            ReferenceDescriptionCollection refs = null;
            try { refs = browser.Browse(nodeId); } catch { return null; }
            if (refs == null) return null;

            for (int i = 0; i < refs.Count; i++)
            {
                var rd = refs[i];
                if (rd.BrowseName.Name == browseName)
                    return rd;
            }
            return null;
        }

        public BrowserPosition BrowseRootNode(Session session)
        {
            Console.WriteLine("Browsing root node at ns=2;s=Path");
            return BrowseBranch(session, "ns=2;s=Path");
        }

        internal BrowserPosition BrowseBranch(Session opcSession, string branch)
        {
            Console.WriteLine($"Trying to browse branch {branch}");
            var response = new BrowserPosition
            {
                Success = false,
                StatusMessage = "Uninitialized",
                Position = ""
            };

            try
            {
                if (opcSession == null || !opcSession.Connected)
                {
                    Console.WriteLine($"Session not connected ... aborting");
                    response.StatusMessage = "Session not connected";
                    return response;
                }
                NodeId currentNode = ObjectIds.ObjectsFolder;
                string originalBranch = branch;

                if (!string.IsNullOrWhiteSpace(branch) ) //&&
                    //!branch.Equals("root", StringComparison.OrdinalIgnoreCase))
                {
                    NodeId explicitId = TryParseNodeId(branch);
                    if (explicitId != null)
                    {
                        currentNode = explicitId;
                    }
                    else
                    {
                        var segments = branch.Split(new[] { '/' }, StringSplitOptions.RemoveEmptyEntries);
                        for (int i = 0; i < segments.Length; i++)
                        {
                            var seg = segments[i].Trim();
                            if (seg.Length == 0) continue;

                            var child = FindChildByBrowseName(opcSession, currentNode, seg);
                            if (child == null)
                            {
                                response.StatusMessage = "Path segment not found: " + seg;
                                return response;
                            }
                            currentNode = child;
                        }
                    }
                }

                Browser browser = new Browser(opcSession)
                {
                    BrowseDirection = BrowseDirection.Forward,
                    NodeClassMask = (int)NodeClass.Object | (int)NodeClass.Variable,
                    ReferenceTypeId = ReferenceTypeIds.HierarchicalReferences,
                    IncludeSubtypes = true
                };

                ReferenceDescriptionCollection refs;
                try
                {
                    refs = browser.Browse(currentNode);
                }
                catch (Exception bx)
                {
                    response.StatusMessage = "Browse failed: " + bx.Message;
                    return response;
                }

                if (refs == null)
                {
                    response.StatusMessage = "No references";
                    response.Success = true;
                    response.Position = currentNode.ToString();
                    response.Branches = new string[0];
                    response.Leaves = new string[0];
                    return response;
                }

                var branches = new List<string>();
                var leaves = new List<string>();

                for (int i = 0; i < refs.Count; i++)
                {
                    var rd = refs[i];
                    NodeId childId;
                    try
                    {
                        childId = ExpandedNodeId.ToNodeId(rd.NodeId, opcSession.NamespaceUris);
                    }
                    catch
                    {
                        continue;
                    }

                    bool isObject = (rd.NodeClass & NodeClass.Object) != 0;
                    bool isVariable = (rd.NodeClass & NodeClass.Variable) != 0;

                    if (isObject)
                    {
                        var idStr = childId.ToString();
                        branches.Add(idStr);
                    }
                    else if (isVariable)
                    {
                        if (IsTimeSeriesVariable(opcSession, childId, rd))
                        {
                            var idStr = childId.ToString();
                            leaves.Add(idStr);
                        }
                    }
                }

                branches.Sort(StringComparer.OrdinalIgnoreCase);
                leaves.Sort(StringComparer.OrdinalIgnoreCase);

                response.Success = true;
                response.StatusMessage = "OK";
                response.Position = currentNode.ToString();
                response.Branches = branches.ToArray();
                response.Leaves = leaves.ToArray();
            }
            catch (Exception ex)
            {
                response.Success = false;
                response.StatusMessage = "Exception: " + ex.Message;
            }

            return response;
        }

        private static NodeId TryParseNodeId(string text)
        {
            if (string.IsNullOrWhiteSpace(text)) return null;
            try
            {
                return NodeId.Parse(text);
            }
            catch
            {
                return null;
            }
        }

        private NodeId FindChildByBrowseName(Session session, NodeId parent, string browseName)
        {
            if (session == null) return null;
            Browser browser;
            try
            {
                browser = new Browser(session)
                {
                    BrowseDirection = BrowseDirection.Forward,
                    NodeClassMask = (int)NodeClass.Object | (int)NodeClass.Variable,
                    ReferenceTypeId = ReferenceTypeIds.HierarchicalReferences,
                    IncludeSubtypes = true
                };
            }
            catch
            {
                return null;
            }

            ReferenceDescriptionCollection refs;
            try
            {
                refs = browser.Browse(parent);
            }
            catch
            {
                return null;
            }
            if (refs == null) return null;

            for (int i = 0; i < refs.Count; i++)
            {
                var rd = refs[i];
                var bn = rd.BrowseName.Name;
                var disp = rd.DisplayName?.Text;
                if (string.Equals(bn, browseName, StringComparison.OrdinalIgnoreCase) ||
                    string.Equals(disp, browseName, StringComparison.OrdinalIgnoreCase))
                {
                    try
                    {
                        return ExpandedNodeId.ToNodeId(rd.NodeId, session.NamespaceUris);
                    }
                    catch
                    {
                        return null;
                    }
                }
            }
            return null;
        }
    }
}
