package mariadb

import (
	"database/sql"

	"github.com/redhajuanda/komon/logger"
	"gitlab.sicepat.tech/pka/sds/configs"
)

// Migrator is a wrapper around the Migrator client
type Migrator struct {
	cfg   *configs.Config
	log   logger.Logger
	qwery *Qwery
}

// NewMigrator creates a new Migrator instance
func NewMigrator(cfg *configs.Config, log logger.Logger, qwery *Qwery) *Migrator {
	return &Migrator{
		cfg:   cfg,
		log:   log,
		qwery: qwery,
	}
}

// GetRepositoryName returns the name of the repository
func (m *Migrator) GetRepositoryName() string {
	return "mariadb"
}

// GetDB returns the database connection
func (m *Migrator) GetDB() *sql.DB {
	return m.qwery.DB()
}

// GetDBDriver returns the driver name
func (m *Migrator) GetDBDriver() string {
	return DBDriver
}

// GetMigrationDir returns the migration directory
func (m *Migrator) GetMigrationDir() string {
	return "./internal/adapter/outbound/mariadb/migrations/scripts"
}