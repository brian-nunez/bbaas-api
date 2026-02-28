package data

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"
)

type Config struct {
	Driver       string
	DSN          string
	MaxOpenConns int
	MaxIdleConns int
}

type Adapter interface {
	Name() string
	DriverName() string
	DefaultDSN() string
	NormalizeDSN(input string) string
}

type SQLiteAdapter struct{}

type PostgresAdapter struct{}

func (SQLiteAdapter) Name() string {
	return "sqlite"
}

func (SQLiteAdapter) DriverName() string {
	return "sqlite"
}

func (SQLiteAdapter) DefaultDSN() string {
	return "file:bbaas.db?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
}

func (SQLiteAdapter) NormalizeDSN(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return (SQLiteAdapter{}).DefaultDSN()
	}

	return trimmed
}

func (PostgresAdapter) Name() string {
	return "postgres"
}

func (PostgresAdapter) DriverName() string {
	return "postgres"
}

func (PostgresAdapter) DefaultDSN() string {
	return "postgres://postgres:postgres@localhost:5432/bbaas?sslmode=disable"
}

func (PostgresAdapter) NormalizeDSN(input string) string {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return (PostgresAdapter{}).DefaultDSN()
	}

	return trimmed
}

func DefaultConfig() Config {
	return Config{
		Driver:       "sqlite",
		DSN:          "",
		MaxOpenConns: 10,
		MaxIdleConns: 10,
	}
}

func ResolveAdapter(name string) (Adapter, error) {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "sqlite":
		return SQLiteAdapter{}, nil
	case "postgres", "postgresql":
		return PostgresAdapter{}, nil
	default:
		return nil, fmt.Errorf("unsupported DB driver %q (supported: sqlite, postgres)", name)
	}
}

func Open(config Config) (*sql.DB, Adapter, error) {
	adapter, err := ResolveAdapter(config.Driver)
	if err != nil {
		return nil, nil, err
	}

	driverName := adapter.DriverName()
	if !isDriverRegistered(driverName) {
		return nil, nil, fmt.Errorf("%s driver is not registered in this binary", driverName)
	}

	dsn := adapter.NormalizeDSN(config.DSN)
	db, err := sql.Open(driverName, dsn)
	if err != nil {
		return nil, nil, fmt.Errorf("open %s connection: %w", adapter.Name(), err)
	}

	if config.MaxOpenConns > 0 {
		db.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		db.SetMaxIdleConns(config.MaxIdleConns)
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("ping %s database: %w", adapter.Name(), err)
	}

	// SQLite is single-writer; keeping one pooled connection avoids lock contention
	// that can stall authenticated request paths under concurrent writes.
	if adapter.Name() == "sqlite" {
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
	}

	return db, adapter, nil
}

func isDriverRegistered(driverName string) bool {
	for _, registeredDriver := range sql.Drivers() {
		if registeredDriver == driverName {
			return true
		}
	}

	return false
}
