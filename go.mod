module dd-nats

go 1.16

require (
	github.com/cyops-se/opc v0.3.7
	github.com/go-ole/go-ole v1.2.6
	github.com/gofiber/fiber/v2 v2.32.0
	github.com/gofiber/websocket/v2 v2.0.21
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/influxdata/influxdb1-client v0.0.0-20220302092344-a9ab5670611c
	github.com/jackc/pgconn v1.12.1
	github.com/jackc/pgx/v4 v4.16.1
	github.com/nats-io/nats-server/v2 v2.8.1 // indirect
	github.com/nats-io/nats.go v1.14.0
	github.com/pkg/errors v0.9.1 // indirect
	github.com/simonvetter/modbus v1.5.1
	github.com/sirius1024/go-amqp-reconnect v1.0.0
	github.com/streadway/amqp v1.0.0
	github.com/stretchr/testify v1.8.0 // indirect
	golang.org/x/sys v0.0.0-20220227234510-4e6760a101f9
	google.golang.org/protobuf v1.28.0 // indirect
	gorm.io/driver/sqlite v1.3.2
	gorm.io/gorm v1.23.7
)

replace github.com/cyops-se/opc => c:\Development\src\cyops-se\opc
