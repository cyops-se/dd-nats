package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"dd-nats/common/logger"
	"dd-nats/common/types"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

type TimescaleEmitter struct {
	Host      string               `json:"host"`
	Port      int                  `json:"port"`
	User      string               `json:"user"`
	Password  string               `json:"password"`
	Authident bool                 `json:"authident"`
	Database  string               `json:"database"`
	Batchsize int                  `json:"batchsize"`
	err       error                `json:"-" gorm:"-"`
	messages  chan types.DataPoint `json:"-" gorm:"-"`
	builder   strings.Builder      `json:"-" gorm:"-"`
	count     uint64               `json:"-" gorm:"-"`
}

var debug *bool
var batchSize *int

var TimescaleDBConn *pgxpool.Pool
var ids map[string]int

func (emitter *TimescaleEmitter) InitEmitter() error {
	emitter.connectdb()

	ids = make(map[string]int)
	emitter.initBatch()

	emitter.messages = make(chan types.DataPoint, 2000)
	go emitter.processMessages()
	go emitter.syncMeta()

	return nil
}

func (emitter *TimescaleEmitter) ProcessMessage(dp types.DataPoint) {
	if emitter.messages != nil {
		emitter.messages <- dp
	}
}

func (emitter *TimescaleEmitter) processMessages() {

	for {
		dp := <-emitter.messages

		if TimescaleDBConn == nil {
			if emitter.connectdb() != nil {
				continue
			}
		}

		var err error

		// use 'ids' as a local datapoint name cache to resolve id
		// if not in the cache, get it from the database
		// if not in the database, insert a new meta item and get the new id
		id, ok := ids[dp.Name]
		if !ok {
			if err = TimescaleDBConn.QueryRow(context.Background(), "select tag_id from measurements.tags where name=$1", dp.Name).Scan(&id); err != nil {
				log.Printf("TIMESCALEDB couldn't find %s in database, error: %s, type: %T, err value: %#v", dp.Name, err.Error(), err, err)
				if err == pgx.ErrNoRows {
					err = TimescaleDBConn.QueryRow(context.Background(), "insert into measurements.tags (name) values ($1) returning tag_id", dp.Name).Scan(&id)
				}
			}
			ids[dp.Name] = id
		}

		if err == nil {
			if emitter.appendPoint(id, &dp) {
				emitter.insertBatch()
				emitter.initBatch()
			}
		} else {
			log.Println("TIMESCALEDB insert process data failed, err:", err.Error())
		}
	}
}

// func (emitter *TimescaleEmitter) ProcessMeta(dp *types.DataPointMeta) {
// 	var id int
// 	if emitter.rowExists("select name from measurements.tags where name=$1", dp.Name) == false {
// 		if err := TimescaleDBConn.QueryRow(context.Background(), "insert into measurements.tags (name, description) values ($1, $2) returning tag_id", dp.Name, dp.Description).Scan(&id); err != nil {
// 			fmt.Println("TIMESCALE failed to insert,", err.Error())
// 		}
// 	} else {
// 		if _, err := TimescaleDBConn.Exec(context.Background(), "update measurements.tags set description = $2 where name = $1", dp.Name, dp.Description); err != nil {
// 			fmt.Println("TIMESCALE failed to update,", err.Error())
// 		}
// 	}
// }

func (emitter *TimescaleEmitter) connectdb() error {
	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		emitter.Host, emitter.Port, emitter.User, emitter.Password, emitter.Database)
	// dburl := fmt.Sprintf("postgres://%s:%s@%s:5432/%s", emitter.User, emitter.Password, emitter.Host, emitter.Database)
	TimescaleDBConn, emitter.err = pgxpool.Connect(context.Background(), psqlInfo)
	if emitter.err != nil {
		logger.Log("error", "TimescaleDB emitter", fmt.Sprintf("Failed to connect to the database, err: %s", emitter.err.Error()))
		return emitter.err
	}

	logger.Log("info", "TimescaleDB emitter", fmt.Sprintf("Database server connected: %s", emitter.Host))
	return emitter.err
}

func (emitter *TimescaleEmitter) rowExists(query string, args ...interface{}) bool {
	var exists bool
	query = fmt.Sprintf("SELECT exists (%s)", query)
	err := TimescaleDBConn.QueryRow(context.Background(), query, args...).Scan(&exists)
	if err != nil && err != sql.ErrNoRows {
		log.Fatalf("error checking if row exists '%s' %v", args, err)
	}
	return exists
}

func (emitter *TimescaleEmitter) initBatch() {
	emitter.count = 0
	emitter.builder.Reset()
	emitter.builder.Grow(4096)

	fmt.Fprintf(&emitter.builder, "insert into measurements.raw_measurements (tag, time, value, quality) values ")
}

func (emitter *TimescaleEmitter) appendPoint(id int, v *types.DataPoint) bool {
	if emitter.count > 0 {
		fmt.Fprintf(&emitter.builder, ",")
	}

	fmt.Fprintf(&emitter.builder, "(%d, '%s', %v, %d)", id, v.Time.Format(time.RFC3339Nano), v.Value, v.Quality)
	emitter.count++

	return emitter.count > uint64(emitter.Batchsize)
}

func (emitter *TimescaleEmitter) insertBatch() error {
	if emitter.count > 0 {
		fmt.Fprintf(&emitter.builder, ";")
		insert := emitter.builder.String()
		_, err := TimescaleDBConn.Exec(context.Background(), insert)
		if err != nil {
			switch err := err.(type) {
			default:
				logger.Log("error", "TimescaleDB emitter", fmt.Sprintf("failed to insert: %#v", err))
				// log.Println(insert)
				emitter.initBatch()
				return err
			case *pgconn.PgError:
				if err.Code == "57P01" {
					return emitter.connectdb()
				}

				logger.Log("error", "TimescaleDB emitter", fmt.Sprintf("failed to insert: %#v", err))
				// log.Println(insert)
				emitter.initBatch()
				return err
			}
		}
	}

	return nil
}

func (emitter *TimescaleEmitter) syncMeta() {
	ticker := time.NewTicker(30 * time.Second)
	for {
		<-ticker.C
		// var metaitems []types.DataPointMeta
		// if err := db.DB.Find(&metaitems).Error; err != nil {
		// 	fmt.Println("TIMESCALE failed to get meta items,", err.Error())
		// 	continue
		// }

		// for _, dp := range metaitems {
		// 	var id int
		// 	if emitter.rowExists("select name from measurements.tags where name=$1", dp.Name) == false {
		// 		if err := TimescaleDBConn.QueryRow(context.Background(), "insert into measurements.tags (name,unit,min,max,description) values ($1,$2,$3,$4,$5) returning tag_id",
		// 			dp.Name, dp.EngUnit, dp.MinValue, dp.MaxValue, dp.Description).Scan(&id); err != nil {
		// 			log.Printf("TIMESCALE failed to insert, err: %s,", err.Error())
		// 		}
		// 	} else {
		// 		_, err := TimescaleDBConn.Exec(context.Background(), "update measurements.tags set unit=$2,min=$3,max=$4,description=$5 where name=$1",
		// 			dp.Name, dp.EngUnit, dp.MinValue, dp.MaxValue, dp.Description)

		// 		if err != nil {
		// 			log.Printf("TIMESCALE failed to update, err: %s", err.Error())
		// 		}
		// 	}
		// }
	}
}
