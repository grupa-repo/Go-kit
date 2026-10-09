package sqlpool_test

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"github.com/grupa-repo/go-kit/sqlpool"
)

// fakeDriver stands in for a real database. The DSN "down" refuses to connect.
type fakeDriver struct{}

var errDown = errors.New("connection refused")

func (fakeDriver) Open(dsn string) (driver.Conn, error) {
	if dsn == "down" {
		return nil, errDown
	}
	return fakeConn{}, nil
}

type fakeConn struct{}

func (fakeConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (fakeConn) Close() error                        { return nil }
func (fakeConn) Begin() (driver.Tx, error)           { return nil, errors.New("not implemented") }

func init() { sql.Register("sqlpool-fake", fakeDriver{}) }

func TestOpenAppliesLimits(t *testing.T) {
	db, err := sqlpool.Open(context.Background(), "sqlpool-fake", "up", sqlpool.Limits{MaxOpen: 6, MaxIdle: 3})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer db.Close()

	if got := db.Stats().MaxOpenConnections; got != 6 {
		t.Errorf("MaxOpenConnections = %d, want 6", got)
	}
}

func TestOpenRejectsAnUnlimitedPool(t *testing.T) {
	for name, l := range map[string]sqlpool.Limits{
		"zero value":      {},
		"negative":        {MaxOpen: -1},
		"idle above open": {MaxOpen: 2, MaxIdle: 3},
		"negative idle":   {MaxOpen: 2, MaxIdle: -1},
	} {
		t.Run(name, func(t *testing.T) {
			db, err := sqlpool.Open(context.Background(), "sqlpool-fake", "up", l)
			if err == nil {
				db.Close()
				t.Error("Open accepted limits it should refuse")
			}
		})
	}
}

func TestOpenReportsAnUnreachableDatabase(t *testing.T) {
	db, err := sqlpool.Open(context.Background(), "sqlpool-fake", "down", sqlpool.Limits{MaxOpen: 4})
	if !errors.Is(err, errDown) {
		t.Errorf("err = %v, want the driver's connection error", err)
	}
	if db != nil {
		t.Error("a failed Open should return a nil pool")
	}
}

func TestOpenReportsAnUnknownDriver(t *testing.T) {
	if _, err := sqlpool.Open(context.Background(), "no-such-driver", "up", sqlpool.Limits{MaxOpen: 4}); err == nil {
		t.Error("Open with an unregistered driver returned nil")
	}
}

func TestOpenHonoursACancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := sqlpool.Open(ctx, "sqlpool-fake", "up", sqlpool.Limits{MaxOpen: 4}); !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v, want context.Canceled", err)
	}
}
