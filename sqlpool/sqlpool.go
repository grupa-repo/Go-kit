// Package sqlpool opens a database/sql pool with explicit limits.
//
// database/sql defaults to an unlimited number of open connections and only
// two idle ones. That is the wrong shape twice over: a burst can open
// connections until the database refuses them, and between bursts the pool
// throws away connections it is about to need again. A database's connection
// cap usually belongs to the instance, not to one client, so each service has
// to state its share.
package sqlpool

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// Limits is one service's share of a database. MaxOpen is required.
type Limits struct {
	// MaxOpen caps connections in use plus idle. It must be positive: leaving
	// it unlimited is the mistake this package exists to prevent.
	MaxOpen int
	// MaxIdle caps idle connections kept for reuse. Zero means MaxOpen/2,
	// and at least 1.
	MaxIdle int
	// MaxLifetime closes a connection this long after it was opened, so a
	// failover or a credential rotation is picked up. Zero means 30 minutes.
	MaxLifetime time.Duration
	// MaxIdleTime closes a connection idle for this long. Zero means 5
	// minutes.
	MaxIdleTime time.Duration
}

func (l Limits) withDefaults() (Limits, error) {
	if l.MaxOpen <= 0 {
		return l, errors.New("sqlpool: Limits.MaxOpen must be positive")
	}
	if l.MaxIdle < 0 || l.MaxIdle > l.MaxOpen {
		return l, fmt.Errorf("sqlpool: Limits.MaxIdle must be between 0 and MaxOpen (%d)", l.MaxOpen)
	}
	if l.MaxIdle == 0 {
		l.MaxIdle = max(l.MaxOpen/2, 1)
	}
	if l.MaxLifetime == 0 {
		l.MaxLifetime = 30 * time.Minute
	}
	if l.MaxIdleTime == 0 {
		l.MaxIdleTime = 5 * time.Minute
	}
	return l, nil
}

// Open opens a pool for the named driver, applies limits, and pings the
// database once to prove the connection string works. The caller imports the
// driver; this package depends on none.
//
// ctx bounds the ping only. On any error the pool is closed and nil returned.
func Open(ctx context.Context, driver, dsn string, limits Limits) (*sql.DB, error) {
	l, err := limits.withDefaults()
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(l.MaxOpen)
	db.SetMaxIdleConns(l.MaxIdle)
	db.SetConnMaxLifetime(l.MaxLifetime)
	db.SetConnMaxIdleTime(l.MaxIdleTime)

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
