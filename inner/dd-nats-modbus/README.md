# dd-nats-modbus
Basic Modbus TCP master implementation managed through CSV files with the following format (first line is a header):
```
tagname;description;signaltype;slaveipaddress;datatype;datalength;engunit;byteorder;functioncode;modbusaddress;range;rangeplc;
[tag];[description];Serial Analog;[slave ip address];U-Int;16-bit;m;AB CD;3;40107;0-1;3965-20000
[tag];[description];Serial Analog;[slave ip address];U-Int;16-bit;m³/h;AB CD;3;40104;0-60;3965-20000
[tag];[description];Serial Analog;[slave ip address];U-Int;16-bit;mv;AB CD;3;40105;0-2,5;3965-20000
[tag];[description];Serial Analog;[slave ip address];U-Int;16-bit;m;AB CD;3;40106;0-5;3965-20000
```

    // col 0: tag name (mandatory, unique tag name)
    // col 1: tag description (informational, displayed in UI)
    // col 2: signal type (informational only)
    // col 3: ip address (mandatory, modbus slave ip address)
    // col 4: data type (mandatory, currently only supports uint)
    // col 5: data length (mandatory, currently only 16-bit supported)
    // col 6: engineering unit (informational, e.g. m3/h)
    // col 7: byte order (currently not supported, e.g. AB CD)
    // col 8: function code (mandatory, currently only 3 and 4 supported)
    // col 9: modbus address (mandatory, e.g. 40107)
    // col 10: actual range (mandatory, e.g. 0-60, string where min and max are separated by a -)
    // col 11: PLC range (mandatory, 3965-20000, string where raw min and max are separated by a -)