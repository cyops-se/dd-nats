package main

import (
	"log"

	"dd-nats/common/types"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase(ctx types.Context) {
	// filename := path.Join(ctx.Wdir, "test.db")
	// database, err := gorm.Open(sqlite.Open(filename), &gorm.Config{})
	dsn := "user=postgres password=hemligt dbname=dev host=192.168.0.174 port=5432"
	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Println("Failed to connect to local database", err)
		return
	}

	log.Println("Postgres database connected!")

	// The User model is special due to the 'password' field, and has
	// user specific routes
	database.AutoMigrate(&types.User{})

	// Generic CRUD data types
	configureTypes(database, types.Log{}, types.KeyValuePair{})
	// configureTypes(database, types.User{}, types.Settings{}, types.Recipient{})
	// configureTypes(database, types.DataPointMeta{}, types.Listener{}, types.Emitter{})

	DB = database
}

func InitContent() {
}

func configureTypes(database *gorm.DB, datatypes ...interface{}) {
	if database == nil {
		return
	}

	for _, datatype := range datatypes {
		stmt := &gorm.Statement{DB: database}
		stmt.Parse(datatype)
		name := stmt.Schema.Table
		types.RegisterType(name, datatype)
		database.AutoMigrate(datatype)
	}
}
