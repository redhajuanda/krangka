package migrate

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/redhajuanda/komon/logger"
	"gitlab.sicepat.tech/pka/sds/configs"
)

// Generate is a struct that contains the configuration for the migration generation
type Generate struct {
	cfg        *configs.Config
	log        logger.Logger
	migrators  []Migrator
	repository string
	fileName   string
}

type GenerateParams struct {
	Repository string
	FileName   string
}

// NewGenerate creates a new migration generation instance
func NewGenerate(cfg *configs.Config, log logger.Logger, migrators []Migrator, params GenerateParams) *Generate {
	return &Generate{
		cfg:        cfg,
		log:        log,
		migrators:  migrators,
		repository: params.Repository,
		fileName:   params.FileName,
	}
}

// Execute executes the migration generation process.
// It creates a new migration file with the current timestamp and the specified file name.
// The migration file is created in the directory specified in the migration configuration for the given repository.
func (g *Generate) Execute(ctx context.Context) error {

	if g.repository == "" {
		g.log.Fatal("repository is required for migration")
	}

	for _, migrator := range g.migrators {

		if migrator.GetRepositoryName() != string(g.repository) {
			continue
		}

		g.log.Infof("generating new migration file %s for repository %s", g.fileName, g.repository)

		if len(g.fileName) == 0 {
			g.log.Fatal("file name is required for new migration")
		}

		// create a new migration file
		newMigrationFile := filepath.Join(migrator.GetMigrationDir(), fmt.Sprintf("%s-%s.sql", time.Now().Format("20060102150405"), g.fileName))

		file, err := os.Create(newMigrationFile)
		if err != nil {
			g.log.Fatalf("failed to create migration file: %v", err)
		}
		defer file.Close()

		// write the initial migration content
		content := "-- +migrate Up\n\n-- +migrate Down\n\n"
		if _, err := file.WriteString(content); err != nil {
			g.log.Fatalf("failed to write to migration file: %v", err)
		}

		g.log.Infof("new migration file created: %s", newMigrationFile)

	}

	return nil
}