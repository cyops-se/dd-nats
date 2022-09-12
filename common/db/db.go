package db

import (
	"dd-nats/common/types"
	"log"
	"path"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase(ctx types.Context, name string) error {
	filename := path.Join(ctx.Wdir, name)
	database, err := gorm.Open(sqlite.Open(filename), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to database: %s", err.Error())
		return err
	}

	log.Printf("Application database connected")
	DB = database
	return nil
}

func ConfigureTypes(database *gorm.DB, datatypes ...interface{}) {
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
