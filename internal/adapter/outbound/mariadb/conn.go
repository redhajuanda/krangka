package mariadb

import (
	"database/sql"
	"embed"
	"fmt"
	"time"

	"github.com/redhajuanda/komon/logger"

	"github.com/redhajuanda/qwery"

	_ "github.com/go-sql-driver/mysql" // MySQL driver
)

var (
	DBDriver = "mysql"
	//go:embed queries/*
	queryFiles embed.FS
)

// Qwery is a wrapper around the Qwery client
type Qwery struct {
	*qwery.Client
}

// ParamQwery is a parameter for the Qwery client
type ParamQwery struct {
	Username        string
	Password        string
	Host            string
	Port            string
	DBName          string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// NewQwery creates a new connection to the Qwery SDK
func NewQwery(param ParamQwery, log logger.Logger) *Qwery {

	c, err := qwery.Init(log, qwery.Option{
		DB:          newMariaDB(param, log),
		QueryFiles:  queryFiles,
		DriverName:  DBDriver,
		Placeholder: qwery.Question,
	})
	if err != nil {
		log.Fatalf("failed to initialize qwery client: %v", err)
	}

	return &Qwery{c}

}

// Close closes the MariaDB connection
func (m *Qwery) Close() error {
	return m.Client.DB().Close()
}

// newMariaDB creates a new connection to the MariaDB database
func newMariaDB(param ParamQwery, log logger.Logger) *sql.DB {

	var (
		connString string = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=True", param.Username, param.Password, param.Host, param.Port, param.DBName)
	)

	db, err := sql.Open(DBDriver, connString)
	if err != nil {
		log.Fatalf("failed to connect to MariaDB: %v", err)
	}

	// Configure connection pool settings
	db.SetMaxOpenConns(param.MaxOpenConns)       // Maximum number of open connections
	db.SetMaxIdleConns(param.MaxIdleConns)       // Maximum number of idle connections
	db.SetConnMaxLifetime(param.ConnMaxLifetime) // Maximum connection lifetime
	db.SetConnMaxIdleTime(param.ConnMaxIdleTime) // Maximum idle time for connections

	if err := db.Ping(); err != nil {
		log.Fatalf("failed to ping MariaDB: %v", err)
	}
	return db

}
