package main

import (
	"context"
	"dd-nats/common/types"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

const (
	maxPendingPoints  = 100000 // max number of points buffered while the database is unavailable
	maxRowsPerInsert  = 1000   // 4 parameters per row, well below the PostgreSQL limit of 65535
	flushInterval     = time.Second
	reconnectInterval = 5 * time.Second
	dbTimeout         = 30 * time.Second
)

type TimescaleEmitter struct {
	Host      string `json:"host"`
	Port      int    `json:"port"`
	User      string `json:"user"`
	Password  string `json:"password"`
	Database  string `json:"database"`
	SSLMode   string `json:"sslmode"`
	Batchsize int    `json:"batchsize"`

	messages    chan types.DataPoint
	stop        chan chan struct{}
	pending     []types.DataPoint
	lastConnect time.Time
	dropped     uint64

	idsLock sync.Mutex
	ids     map[string]int
}

var pool atomic.Pointer[pgxpool.Pool]

// TimescaleDBConn returns the current connection pool, or nil if not connected
func TimescaleDBConn() *pgxpool.Pool {
	return pool.Load()
}

func (emitter *TimescaleEmitter) InitEmitter() error {
	if emitter.Batchsize < 1 {
		emitter.Batchsize = 1
	}

	if emitter.SSLMode == "" {
		emitter.SSLMode = "disable"
	}

	emitter.ids = make(map[string]int)
	emitter.messages = make(chan types.DataPoint, 2000)
	emitter.stop = make(chan chan struct{})
	emitter.connectdb()

	go emitter.processMessages()
	return nil
}

// Stop flushes pending data points and stops the emitter
func (emitter *TimescaleEmitter) Stop() {
	done := make(chan struct{})
	select {
	case emitter.stop <- done:
		select {
		case <-done:
		case <-time.After(10 * time.Second):
		}
	case <-time.After(time.Second):
	}

	if p := pool.Swap(nil); p != nil {
		p.Close()
	}
}

func (emitter *TimescaleEmitter) ProcessDataPointHandler(topic string, responseTopic string, data []byte) error {
	var dp types.DataPoint
	if err := json.Unmarshal(data, &dp); err != nil {
		return fmt.Errorf("failed to unmarshal process data: %s", err.Error())
	}

	emitter.ProcessMessage(dp)
	return nil
}

func (emitter *TimescaleEmitter) ProcessMessage(dp types.DataPoint) {
	if emitter.messages != nil {
		emitter.messages <- dp
	}
}

// InvalidateIds clears the tag id cache, must be called when tags are removed
func (emitter *TimescaleEmitter) InvalidateIds() {
	emitter.idsLock.Lock()
	emitter.ids = make(map[string]int)
	emitter.idsLock.Unlock()
}

func (emitter *TimescaleEmitter) processMessages() {
	ticker := time.NewTicker(flushInterval)
	defer ticker.Stop()

	for {
		select {
		case dp := <-emitter.messages:
			if math.IsNaN(dp.Value) || math.IsInf(dp.Value, 0) {
				svc.Trace("TimescaleDB emitter", "Skipping non-finite value for %s", dp.Name)
				continue
			}

			emitter.pending = append(emitter.pending, dp)
			if len(emitter.pending) >= emitter.Batchsize {
				emitter.flush()
			}

		case <-ticker.C:
			if len(emitter.pending) > 0 {
				emitter.flush()
			}

		case done := <-emitter.stop:
			if len(emitter.pending) > 0 {
				emitter.flush()
			}
			close(done)
			return
		}
	}
}

// isConnectionError returns true for errors where the insert should be retried
// later (connection lost, server shutting down, ...) as opposed to errors
// caused by the data itself
func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	var pgerr *pgconn.PgError
	if errors.As(err, &pgerr) {
		return strings.HasPrefix(pgerr.Code, "08") || strings.HasPrefix(pgerr.Code, "57P")
	}

	// Anything else (network errors, timeouts, closed pool) is considered a connection problem
	return true
}

func (emitter *TimescaleEmitter) handleConnectionError(err error) {
	svc.Error("TimescaleDB emitter", "Database connection error, %d point(s) kept for retry: %s", len(emitter.pending), err.Error())
	if p := pool.Swap(nil); p != nil {
		p.Close()
	}
}

func (emitter *TimescaleEmitter) trimPending() {
	if excess := len(emitter.pending) - maxPendingPoints; excess > 0 {
		emitter.pending = append([]types.DataPoint(nil), emitter.pending[excess:]...)
		emitter.dropped += uint64(excess)
		svc.Error("TimescaleDB emitter", "Buffer full, %d oldest point(s) dropped (%d in total)", excess, emitter.dropped)
	}
}

func (emitter *TimescaleEmitter) flush() {
	defer emitter.trimPending()

	conn := TimescaleDBConn()
	if conn == nil {
		if time.Since(emitter.lastConnect) < reconnectInterval || emitter.connectdb() != nil {
			return
		}
		conn = TimescaleDBConn()
	}

	// Resolve tag ids, keep only points with a valid id
	rows := make([][]interface{}, 0, len(emitter.pending))
	for _, dp := range emitter.pending {
		id, err := emitter.resolveId(conn, dp.Name)
		if err != nil {
			if isConnectionError(err) {
				emitter.handleConnectionError(err)
				return
			}

			svc.Error("TimescaleDB emitter", "Failed to resolve tag id for %s, point dropped: %s", dp.Name, err.Error())
			continue
		}

		rows = append(rows, []interface{}{id, dp.Time, dp.Value, dp.Quality})
	}

	for len(rows) > 0 {
		n := len(rows)
		if n > maxRowsPerInsert {
			n = maxRowsPerInsert
		}

		if err := insertRows(conn, rows[:n]); err != nil {
			if isConnectionError(err) {
				// Keep all pending points for retry, rows that were already
				// inserted are ignored by 'on conflict do nothing'
				emitter.handleConnectionError(err)
				return
			}

			svc.Error("TimescaleDB emitter", "Failed to insert %d point(s), dropped: %s", n, err.Error())
		}

		rows = rows[n:]
	}

	emitter.pending = emitter.pending[:0]
}

func insertRows(conn *pgxpool.Pool, rows [][]interface{}) error {
	var sb strings.Builder
	sb.WriteString("insert into measurements.raw_measurements (tag, time, value, quality) values ")

	args := make([]interface{}, 0, len(rows)*4)
	for i, row := range rows {
		if i > 0 {
			sb.WriteString(",")
		}
		n := i * 4
		fmt.Fprintf(&sb, "($%d,$%d,$%d,$%d)", n+1, n+2, n+3, n+4)
		args = append(args, row...)
	}
	sb.WriteString(" on conflict do nothing")

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	_, err := conn.Exec(ctx, sb.String(), args...)
	return err
}

// resolveId uses 'ids' as a local datapoint name cache to resolve id. If not
// in the cache, get it from the database. If not in the database, insert a new
// meta item and get the new id. Only successfully resolved ids are cached.
func (emitter *TimescaleEmitter) resolveId(conn *pgxpool.Pool, name string) (int, error) {
	emitter.idsLock.Lock()
	id, ok := emitter.ids[name]
	emitter.idsLock.Unlock()

	if ok {
		return id, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	err := conn.QueryRow(ctx, "select tag_id from measurements.tags where name=$1", name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		err = conn.QueryRow(ctx, "insert into measurements.tags (name) values ($1) returning tag_id", name).Scan(&id)
	}

	if err != nil {
		return 0, err
	}

	emitter.idsLock.Lock()
	emitter.ids[name] = id
	emitter.idsLock.Unlock()

	return id, nil
}

func (emitter *TimescaleEmitter) connectionString() string {
	u := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(emitter.User, emitter.Password),
		Host:   net.JoinHostPort(emitter.Host, strconv.Itoa(emitter.Port)),
		Path:   "/" + emitter.Database,
	}

	q := url.Values{}
	q.Set("sslmode", emitter.SSLMode)
	u.RawQuery = q.Encode()
	return u.String()
}

func (emitter *TimescaleEmitter) connectdb() error {
	emitter.lastConnect = time.Now()

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	newpool, err := pgxpool.Connect(ctx, emitter.connectionString())
	if err != nil {
		svc.Error("TimescaleDB emitter", "Failed to connect to the database %s:%d, err: %s", emitter.Host, emitter.Port, err.Error())
		return err
	}

	if old := pool.Swap(newpool); old != nil {
		old.Close()
	}

	svc.Info("TimescaleDB emitter", "Database server connected: %s", emitter.Host)
	return nil
}

func rowExists(query string, args ...interface{}) (bool, error) {
	conn := TimescaleDBConn()
	if conn == nil {
		return false, fmt.Errorf("no Timescale database connection available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbTimeout)
	defer cancel()

	var exists bool
	query = fmt.Sprintf("SELECT exists (%s)", query)
	err := conn.QueryRow(ctx, query, args...).Scan(&exists)
	return exists, err
}
