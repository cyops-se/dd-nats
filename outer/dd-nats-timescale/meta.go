package main

import (
	"context"
	"database/sql"
	"dd-nats/common/types"
	"fmt"
)

func getAllMetaFromDatabase() (types.DataPointMetas, error) {
	conn := TimescaleDBConn()
	if conn == nil {
		return nil, fmt.Errorf("no Timescale database connection available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	rows, err := conn.Query(ctx, "select name,description,location,type,unit,min,max from measurements.tags")
	if err != nil {
		return nil, fmt.Errorf("failed to query tags: %s", err.Error())
	}
	defer rows.Close()

	items := types.DataPointMetas{}
	for rows.Next() {
		var n, d, l, t, u sql.NullString
		var min, max sql.NullFloat64

		if err = rows.Scan(&n, &d, &l, &t, &u, &min, &max); err != nil {
			return nil, err
		}

		items = append(items, types.DataPointMeta{Name: n.String, Description: d.String, Location: l.String, Type: t.String, EngUnit: u.String, MinValue: min.Float64, MaxValue: max.Float64})
	}

	return items, rows.Err()
}

func updateAllMetaInDatabase(items types.DataPointMetas) error {
	conn := TimescaleDBConn()
	if conn == nil {
		return fmt.Errorf("no Timescale database connection available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	for _, dp := range items {
		exists, err := rowExists("select name from measurements.tags where name=$1", dp.Name)
		if err != nil {
			return err
		}

		if !exists {
			var id int
			if err := conn.QueryRow(ctx, "insert into measurements.tags (name,unit,min,max,description,location,type) values ($1,$2,$3,$4,$5,$6,$7) returning tag_id",
				dp.Name, dp.EngUnit, dp.MinValue, dp.MaxValue, dp.Description, dp.Location, dp.Type).Scan(&id); err != nil {
				return err
			}
		} else {
			if _, err := conn.Exec(ctx, "update measurements.tags set unit=$2,min=$3,max=$4,description=$5,type=$6,location=$7 where name=$1",
				dp.Name, dp.EngUnit, dp.MinValue, dp.MaxValue, dp.Description, dp.Type, dp.Location); err != nil {
				return err
			}
		}
	}

	svc.Event("timescale.metaupdated", nil)
	return nil
}

func deleteMetaInDatabase(items types.DataPointMetas) error {
	conn := TimescaleDBConn()
	if conn == nil {
		return fmt.Errorf("no Timescale database connection available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	for _, dp := range items {
		if _, err := conn.Exec(ctx, "delete from measurements.tags where name=$1", dp.Name); err != nil {
			return err
		}
	}

	// Deleted tags get new ids when they appear again
	emitter.InvalidateIds()

	svc.Event("timescale.metaupdated", nil)
	return nil
}
