package main

import (
	"context"
	"database/sql"
	"dd-nats/common/types"
	"fmt"
	"log"
)

func getAllMetaFromDatabase() (types.DataPointMetas, error) {
	if TimescaleDBConn == nil {
		return nil, fmt.Errorf("no Timescale database connection available")
	}

	var items types.DataPointMetas
	if rows, err := TimescaleDBConn.Query(context.Background(), "select name,description,location,type,unit,min,max from measurements.tags"); err != nil {
		log.Printf("TIMESCALE failed to insert, err: %s,", err.Error())
	} else {
		defer rows.Close()
		for rows.Next() {
			var n, d, l, t, u sql.NullString
			var min, max sql.NullFloat64

			err = rows.Scan(&n, &d, &l, &t, &u, &min, &max)
			if err != nil {
				return nil, err
			}

			item := &types.DataPointMeta{Name: n.String, Description: d.String, Location: l.String, EngUnit: u.String, MinValue: min.Float64, MaxValue: max.Float64}
			items = append(items, *item)
		}
	}

	return items, nil
}

func updateAllMetaInDatabase(items types.DataPointMetas) error {
	if TimescaleDBConn == nil {
		return fmt.Errorf("no Timescale database connection available")
	}

	for _, dp := range items {
		var id int
		if emitter.rowExists("select name from measurements.tags where name=$1", dp.Name) == false {
			if err := TimescaleDBConn.QueryRow(context.Background(), "insert into measurements.tags (name,unit,min,max,description,location,type) values ($1,$2,$3,$4,$5) returning tag_id",
				dp.Name, dp.EngUnit, dp.MinValue, dp.MaxValue, dp.Description, dp.Location, dp.Type).Scan(&id); err != nil {
				return err
			}
		} else {
			_, err := TimescaleDBConn.Exec(context.Background(), "update measurements.tags set unit=$2,min=$3,max=$4,description=$5,type=$6,location=$7 where name=$1",
				dp.Name, dp.EngUnit, dp.MinValue, dp.MaxValue, dp.Description, dp.Type, dp.Location)

			if err != nil {
				return err
			}
		}
	}

	return nil
}

// func (emitter *TimescaleEmitter) syncMeta() {
// 	ticker := time.NewTicker(30 * time.Second)
// 	for {
// 		<-ticker.C
// 		var metaitems []types.DataPointMeta
// 		if err := db.DB.Find(&metaitems).Error; err != nil {
// 			fmt.Println("TIMESCALE failed to get meta items,", err.Error())
// 			continue
// 		}

// 		for _, dp := range metaitems {
// 			var id int
// 			if emitter.rowExists("select name from measurements.tags where name=$1", dp.Name) == false {
// 				if err := TimescaleDBConn.QueryRow(context.Background(), "insert into measurements.tags (name,unit,min,max,description) values ($1,$2,$3,$4,$5) returning tag_id",
// 					dp.Name, dp.EngUnit, dp.MinValue, dp.MaxValue, dp.Description).Scan(&id); err != nil {
// 					log.Printf("TIMESCALE failed to insert, err: %s,", err.Error())
// 				}
// 			} else {
// 				_, err := TimescaleDBConn.Exec(context.Background(), "update measurements.tags set unit=$2,min=$3,max=$4,description=$5 where name=$1",
// 					dp.Name, dp.EngUnit, dp.MinValue, dp.MaxValue, dp.Description)

// 				if err != nil {
// 					log.Printf("TIMESCALE failed to update, err: %s", err.Error())
// 				}
// 			}
// 		}
// 	}
// }
