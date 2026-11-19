package db

import (
	"dd-nats/common/types"
	"log"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB

func ConnectDatabase(ctx types.Context, name string) error {
	filename := filepath.Join(ctx.Wdir, name)
	database, err := gorm.Open(sqlite.Open(filename), &gorm.Config{})
	if err != nil {
		log.Printf("Failed to connect to database: %s", err.Error())
		return err
	}

	log.Printf("Application database connected: %s", filename)
	DB = database
	return nil
}

func ConfigureTypes(database *gorm.DB, datatypes ...interface{}) {
	if database == nil {
		return
	}

	for _, datatype := range datatypes {
		stmt := &gorm.Statement{DB: database}
		if err := stmt.Parse(datatype); err != nil {
			log.Printf("Failed to parse database type %T: %s", datatype, err.Error())
			continue
		}

		types.RegisterType(stmt.Schema.Table, datatype)
		if err := database.AutoMigrate(datatype); err != nil {
			log.Printf("Failed to migrate database type %T: %s", datatype, err.Error())
		}
	}
}
