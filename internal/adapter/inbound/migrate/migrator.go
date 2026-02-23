package migrate

import "database/sql"

// Migrator is a interface that contains the methods for the migrator
// It is implemented by the repository to perform database migration
type Migrator interface {
	GetRepositoryName() string // get repository name
	GetDB() *sql.DB            // get database connection
	GetDBDriver() string       // get database driver
	GetMigrationDir() string   // get migration directory
}
