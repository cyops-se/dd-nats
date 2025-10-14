using CsvHelper.Configuration;
using DdUsvc;
using Newtonsoft.Json;
using System;
using System.Collections.Generic;

namespace dd_opcua_lib
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
        public string State { get; set; }
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
    public struct TagInfo
    {
        public string NodeId { get; set; }
        public string Name { get; set; }
        public string Description { get; set; }
        public string EngineeringUnit { get; set; }
        public double? MinValue { get; set; }
        public double? MaxValue { get; set; }
    }

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
        public OpcUaSubscription subscription { get; set; }
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

    public class GetOPCBranches
    {
        [JsonProperty("sid")]
        public int ServerId { get; set; }

        [JsonProperty("branch")]
        public string Branch { get; set; }
    }

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

    public class OpcTagNamesItemResponse : StatusResponse
    {
        [JsonProperty("items")]
        public string[] Items { get; set; }
    }

    public class OpcTagItemMetaResponse : StatusResponse
    {
        [JsonProperty("items")]
        public OpcTagMetaInfo[] Items { get; set; }
    }

    public class OpcTagMetaInfo
    {
        [JsonProperty("n")]
        public string Name { get; set; }
        [JsonProperty("d")]
        public string Description { get; set; }
        [JsonProperty("u")]
        public string EngineeringUnit { get; set; }
        [JsonProperty("min")]
        public double? Min { get; set; }
        [JsonProperty("max")]
        public double? Max { get; set; }
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

    public class GroupCsvRecord
    {
        public int Id { get; set; }
        public string Name { get; set; }
        public int Interval { get; set; }
        public string ProgID { get; set; }
        public int DefaultGroupRaw { get; set; }
        public int RunAtStartRaw { get; set; }
    }

    public class GroupCsvRecordMap : ClassMap<GroupCsvRecord>
    {
        public GroupCsvRecordMap()
        {
            Map(m => m.Id).Name("groupid");
            Map(m => m.Name).Name("groupname", "name");
            Map(m => m.Interval).Name("interval");
            Map(m => m.ProgID).Name("progid");
            Map(m => m.DefaultGroupRaw).Name("defaultgroup");
            Map(m => m.RunAtStartRaw).Name("runatstart");
        }
    }

    public class TagCsvRecord
    {
        public string Name { get; set; }
        public int GroupId { get; set; }
        public string GroupIdRaw { get; set; }
    }

    public class TagCsvRecordMap : ClassMap<TagCsvRecord>
    {
        public TagCsvRecordMap()
        {
            Map(m => m.Name).Name("name");
            Map(m => m.GroupIdRaw).Name("groupid");
            Map(m => m.GroupId).Name("groupid");
        }
    }

    public class GroupHistoryRequest
    {
        [JsonProperty("gid")]
        public int GroupId { get; set; }

        [JsonProperty("start")]
        public DateTime Start { get; set; }

        [JsonProperty("end")]
        public DateTime End { get; set; }
    }

    public class GroupHistoryTagSeries
    {
        [JsonProperty("tag")]
        public string Tag { get; set; }

        [JsonProperty("points")]
        public DataPoint[] Points { get; set; }
    }

    public class GroupHistoryResponse : StatusResponse
    {
        [JsonProperty("gid")]
        public int GroupId { get; set; }

        [JsonProperty("start")]
        public DateTime Start { get; set; }

        [JsonProperty("end")]
        public DateTime End { get; set; }

        [JsonProperty("series")]
        public GroupHistoryTagSeries[] Series { get; set; }
    }

    public class TagHistoryRequest
    {
        [JsonProperty("gid")]
        public int? GroupId { get; set; }

        [JsonProperty("name")]
        public string TagName { get; set; }

        [JsonProperty("start")]
        public DateTime Start { get; set; }

        [JsonProperty("end")]
        public DateTime End { get; set; }
    }

    public class TagHistoryResponse : StatusResponse
    {
        [JsonProperty("gid")]
        public int GroupId { get; set; }

        [JsonProperty("name")]
        public string TagName { get; set; }

        [JsonProperty("start")]
        public DateTime Start { get; set; }

        [JsonProperty("end")]
        public DateTime End { get; set; }

        [JsonProperty("points")]
        public DataPoint[] Points { get; set; }
    }
}
