package worker

import (
	"context"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/redhajuanda/komon/logger"
	"gitlab.sicepat.tech/pka/sds/configs"
)

// WorkerGenerateID is a worker that generates ULIDs into stdout
type WorkerGenerateID struct {
	cfg *configs.Config
	log logger.Logger
}

// NewWorkerGenerateID creates a new WorkerGenerateID instance
func NewWorkerGenerateID(cfg *configs.Config, log logger.Logger) *WorkerGenerateID {
	return &WorkerGenerateID{
		cfg: cfg,
		log: log,
	}
}

// Execute runs the WorkerGenerateID worker
func (w *WorkerGenerateID) Execute(ctx context.Context) error {

	w.log.WithContext(ctx).Info("starting generate id worker")
	for range 20 {
		fmt.Println(ulid.Make().String())
	}
	w.log.WithContext(ctx).Info("generate id worker completed")
	return nil

}