package migrate

import (
	"context"

	"github.com/redhajuanda/komon/logger"
	"github.com/redhajuanda/krangka/configs"

	migrate "github.com/rubenv/sql-migrate"
)

// Migrate is a struct that contains the configuration for the migration
type Migrate struct {
	cfg           *configs.Config
	log           logger.Logger
	migrators     []Migrator
	migrationType string
	max           int
	repository    string
}

// MigrateParams is a struct that contains the parameters for the migration
type MigrateParams struct {
	MigrationType string
	Max           int
	Repository    string
}

// NewMigrate creates a new migrate instance
func NewMigrate(cfg *configs.Config, log logger.Logger, migrator []Migrator, params MigrateParams) *Migrate {

	return &Migrate{
		cfg:           cfg,
		log:           log,
		migrators:     migrator,
		migrationType: params.MigrationType,
		max:           params.Max,
		repository:    params.Repository,
	}

}

// Execute executes the migration process based on the specified migration type.
// It applies the migrations to the database using the sql-migrate package.
func (m *Migrate) Execute(ctx context.Context) error {

	if m.repository == "" {
		m.log.Fatal("repository is required for migration")
	}

	for _, migrator := range m.migrators {

		if migrator.GetRepositoryName() != string(m.repository) {
			continue
		}

		m.log.Infof("starting migration %s for repository %s", m.migrationType, m.repository)

		var (
			// cfg        = loadMigrateConfig(m.log, m.repository)
			migrations = &migrate.FileMigrationSource{
				Dir: migrator.GetMigrationDir(),
			}
			direction migrate.MigrationDirection
		)

		m.log.Infof("using migration config, dialect: %s, dir: %s", migrator.GetDBDriver(), migrator.GetMigrationDir())

		switch m.migrationType {
		case "up":
			direction = migrate.Up
		case "down":
			direction = migrate.Down
		}

		count, err := migrate.ExecMax(migrator.GetDB(), migrator.GetDBDriver(), migrations, direction, int(m.max))
		if err != nil {
			m.log.Fatalf("migration failed: %v", err)
		}

		m.log.Infof("applied %d migrations", count)

	}

	return nil
}
