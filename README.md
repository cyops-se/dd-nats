# dd-nats
NOTE! This repo superseeds previous dd- repos for export of ICS data through a data diode using NATS as core transport on either side of the diode.
Collectors on the ICS side (inside) publish their data to NATS subjects, the proxy transfers them through the diode to the proxy on the outside which publish the messages to a NATS server outside of the diode.

Both sides are designed as micro-service architectures with the NATS server

## inner
The term 'inner' refers to applications and activities that operates on the ICS networks on the inside of the diode. The collect information from different sources on the ICS networks.

### dd-nats-inner-proxy
Subscribes to NATS subject dd.forward.> and forwards subject and data as UDP unicast to specified IP address and port. It assumes dd-nats-outer-proxy listens to that port on that IP address.

### dd-nats-opcda
### dd-nats-modbus
### dd-nats-file-inner

## outer
'outer' refers to applications and activities operating on the outside of the data diode, typically in an IT DMZ or directly on the office network.

### dd-nats-outer-proxy
Listens for UDP unicast messages on specified port and publish the received data on the subject provided in the message.

### dd-nats-postgresdb
### dd-nats-rabbitmq
### dd-nats-file-outer
