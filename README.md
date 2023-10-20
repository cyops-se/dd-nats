# dd-nats
***NOTE! This repo represents the successor of the previous dd-opcda and dd-inserter repos intended for export of ICS data and files through a data diode.***

## Overview
The dd-nats repo contains a set of micro-services (usvcs) that together provide a solution for export of ICS (Industrial Control System) data and files over a data-diode. It currently contain collectors for OPC DA and Modbus TCP, but following the usvc architecture, it is easy to add new collectors for other sources. On the other side of the diode, the data is received by a set of usvcs that store the data in a database and/or forward it to other systems.

As message broker, the usvcs architecture on both sides of the diode use [NATS](https://nats.io/), a lightweight, single executable, high-performance messaging system. You are encouraged to read more about NATS unless you are already comfortable with the concept of message brokers. Both publish/subscribe and request/reply patterns are used by the applications in this repository.

![conceptual overview](./assets/dd-nats-overview-1.png)

Each usvc typically have a passive and an active part, where the passive is responding to requests from consumers and the active part process information in the background and publish the results to the NATS server, for example the OPC DA signal collector. The passive part is used to configure which tags to collect from which OPC DA server at what interval, and the active part collects and publish the data.

## micro service architecture (usvc)
All dd-* executables are designed as separate microservices (usvcs) with one service in each executable, using NATS as interface middleware. The common usvc framework implement a common interface for settings management and also emits a heartbeat to indicate the availability of the service. Usvc specific methods are registered with the framework which make them available for other usvcs to consume. A usvc can implement background workers that also emit messages that other services may subscribe to, for example the process data collected by ```dd-nats-opcda``` and ```dd-nats-modbus```.

Typically, a method is registered with a service name, identity and method name as subjects in the format: ```usvc.[servicename].[identity].[methodname]```. A method in the OPC DA collector that gets all configured groups would for example be regsitered as: ```usvc.opc.cs1.group.getall``` (where ```opc``` is the service name, ```cs1``` is the instance identity and ```group.getall``` is the method name). The instance identity is useful if you want to run several instances of the same service. If several instances run with the same identity, there will be a race condition between them when trying to manage servers, groups, tags and settings.

## Common user interface console (dd-ui)
The dd-ui application is a web application based on Vuetify 2 as frontend and NATS (via REST and Websockets) as backend. It implements all user interface views for the usvcs in the repo and displays the relevant user interface depending the heartbeats emitted from the usvcs connected to the same NATS instance as the dd-ui process. If no usvcs are running, an empty console is displayed.

![empty console](./assets/dd-ui/empty.png)

After starting the ```dd-logger``` usvc, the view representing the dd-logger usvc show up in the meny to the left.

![dd-logger example](./assets/dd-ui/dd-logger-example.png)

## inner
The term 'inner' refers to applications and activities that operates on the ICS networks on the inside of the diode. In other data diode contexts, the term "upstream" or "high end" is used insted to describe the sending side. There are several usvcs on the inner side, some collect information to be forwarded to the outer side and others are support services used for management, visibility and diagnostics.

### dd-nats-inner-proxy
Subscribes to NATS subjects specified in the inner proxy settings page which by default are: **```process.>, file.>, system.log.>, system.heartbeat```** and forwards subject and data as UDP unicast to specified IP address and port. It assumes dd-nats-outer-proxy listens to that port on that IP address. The '```>```' sign in the subject indicate a wildcard that accepts anything (including ```.```)

As it forwards NATS messages to the outer side where they are re-published as they were on the inside, it is possible to use this mechanism to forward any data from the inside to the outside by simlply publishing it as a NATS message.

### dd-nats-opcda
Collects specified process values (tags) from local OPC DA servers and publish them to NATS subject: **```process.actual```**, message: **```{"t": "[time]", "n": "[tagname]", "v": [value], "q": [quality]}```**. It implement methods for browsing local OPC DA servers and manage groups and tags. A tag is assigned to a group, where the group specifies sampling time and the tag represents the OPC DA tag name to be collected.

The user interface is used to manage tags to be collected either using the built-in browser, or by importing a CSV file with the tags. See [this section](./dd-nats-opcda/README.md) for more information on the ```dd-nats-opcda``` user interface.

### dd-nats-modbus
### dd-nats-cache
### dd-nats-file-inner
### dd-nats-logger
Subscribes

## outer
The term 'outer' refers to applications and activities operating on the outside of the data diode, typically in an IT DMZ or directly on the office network. In other data diode contexts, the term "downstream" or "low end" is used insted to describe the receiving side.

### dd-nats-outer-proxy
Listens for UDP unicast messages on specified port and publish the received data on the subject provided in the message.

### dd-nats-postgresdb
### dd-nats-rabbitmq
### dd-nats-file-outer
