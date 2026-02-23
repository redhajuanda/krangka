package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"reflect"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-co-op/gocron/v2"
)

type Closer struct {
	Name string
	Fn   func() error
}

// Bootstrap is the main bootstrapper for the application
// It is responsible for initializing the application and wiring the dependencies
// It also handles the graceful shutdown of the application
type Bootstrap struct {
	mu      sync.Mutex
	closers []Closer
	dep     *Dependency
}

// NewBootstrap creates a new Bootstrap instance and automatically binds closers
func New(dep *Dependency) *Bootstrap {

	b := &Bootstrap{dep: dep}
	b.registerClosers(b.getClosers())
	return b

}

// RunOptions is a struct that contains the options for the Run method
type RunOptions struct {
	StartTimeout time.Duration // StartTimeout is the duration to wait for the runner to start before timing out, if timeout is reached before the runner is started, the runner will be stopped and an error will be returned, default is 30 seconds
	StopTimeout  time.Duration // StopTimeout is the duration to wait for the runner to stop before timing out, if timeout is reached before the runner is stopped, the runner will be stopped and an error will be returned, default is 30 seconds
}

// Run blocks until it receives a signal (SIGINT/SIGTERM).
// It then runs cleanup operations (runner shutdown and resource cleanup) before exiting.
// This method mimics the behavior of fx.Run() - it only exits on signals, not when runner completes.
func (b *Bootstrap) Run(runner Runnable, opts ...RunOptions) error {

	var (
		startTimeout = 30 * time.Second
		stopTimeout  = 30 * time.Second
	)

	if len(opts) > 0 {
		if opts[0].StartTimeout != 0 {
			startTimeout = opts[0].StartTimeout
		}
		if opts[0].StopTimeout != 0 {
			stopTimeout = opts[0].StopTimeout
		}
	}

	// Setup signal handling
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Run the application in a goroutine and monitor for errors
	errCh := make(chan error, 1)
	go func() {
		startCtx, startCancel := context.WithTimeout(context.Background(), startTimeout)
		defer startCancel()

		select {
		case <-startCtx.Done():
			errCh <- startCtx.Err()
		default:
			if err := runner.OnStart(startCtx); err != nil {
				errCh <- err
			}
		}
	}()

	// Block until signal is received (ignore successful runner completion)
	var runErr error
	select {
	case <-ctx.Done():
		// Signal received - perform graceful shutdown
		b.dep.GetLogger().SkipSource().Info("Received shutdown signal, initiating graceful shutdown...")

	case err := <-errCh:
		// Runner failed to start - exit immediately without waiting for signal
		b.dep.GetLogger().SkipSource().Errorf("Runner failed to start: %v", err.Error())
		runErr = err
		// Don't wait for signal, proceed directly to cleanup
	}

	// Perform cleanup with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), stopTimeout)
	defer cancel()

	// Shutdown runner first
	if shutdownErr := runner.OnStop(shutdownCtx); shutdownErr != nil {
		b.dep.GetLogger().SkipSource().Errorf("Error during runner shutdown: %v", shutdownErr.Error())
		if runErr == nil {
			runErr = shutdownErr
		}
	}

	// Close all registered resources
	if closeErr := b.close(); closeErr != nil {
		b.dep.GetLogger().SkipSource().Errorf("Error during resource cleanup: %v", closeErr.Error())
		if runErr == nil {
			runErr = closeErr
		}
	}

	if runErr == nil {
		b.dep.GetLogger().SkipSource().Info("Application shutdown complete")
	}

	return runErr

}

// ScheduleOptions is a struct that contains the options for the schedule method
type ScheduleOptions struct {
	ShutdownTimeout time.Duration // ShutdownTimeout is the duration to wait for the running execution to complete before shutting down the scheduler, if timeout is reached before the execution is completed, the execution will be cancelled, default is 30 seconds
	SingletonMode   bool          // When SingletonMode is true, running execution will be limited to one instance at a time, this is useful to prevent multiple instances of the same job from running at the same time
}

// Schedule schedules a runner to run periodically, using github.com/go-co-op/gocron/v2
// It also handles graceful shutdown, when receives a signal (SIGINT/SIGTERM), it will stop the upcoming execution and wait for the running execution to complete until given ShutdownTimeout.
func (b *Bootstrap) Schedule(pattern string, execute Executable, opts ...ScheduleOptions) error {

	// Get options
	shutdownTimeout, singletonMode := b.parseScheduleOptions(opts)

	// Setup signal handling
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	// Create gocron v2 scheduler
	scheduler, err := gocron.NewScheduler()
	if err != nil {
		return fmt.Errorf("failed to create scheduler: %w", err)
	}

	// Track scheduler errors
	errCh := make(chan error, 1)
	var (
		executeMu     sync.Mutex
		executeCancel context.CancelFunc
		executeDone   chan struct{}
		isRunning     bool
	)

	// Build job options
	jobOpts := []gocron.JobOption{}
	if singletonMode {
		jobOpts = append(jobOpts, gocron.WithSingletonMode(gocron.LimitModeWait))
	}

	// Schedule the runner to execute periodically
	task := b.createScheduledTask(execute, pattern, errCh, &executeMu, &executeCancel, &executeDone, &isRunning)
	_, err = scheduler.NewJob(
		gocron.CronJob(pattern, true),
		gocron.NewTask(task),
		jobOpts...,
	)

	if err != nil {
		return fmt.Errorf("failed to schedule runner: %w", err)
	}

	// Start the scheduler asynchronously
	scheduler.Start()
	b.dep.GetLogger().SkipSource().Info("Scheduler started, pattern: %s", pattern)

	// Block until signal is received
	var runErr error
	select {
	case <-ctx.Done():
		// Signal received - perform graceful shutdown
		b.dep.GetLogger().SkipSource().Info("Received shutdown signal, stopping scheduler...")

	case err := <-errCh:
		// Runner returned an error - log it but still wait for signal
		b.dep.GetLogger().SkipSource().Error("Scheduled execution error occurred", "error", err)
		runErr = err
		// Continue blocking until signal
		<-ctx.Done()
		b.dep.GetLogger().SkipSource().Info("Received shutdown signal after error, stopping scheduler...")
	}

	// Check if job is currently running BEFORE stopping scheduler
	// (scheduler.Shutdown() blocks until jobs complete)
	executeMu.Lock()
	jobIsRunning := isRunning
	done := executeDone
	cancel := executeCancel
	executeMu.Unlock()

	// Wait for running execution to complete if necessary
	b.waitForExecutionComplete(jobIsRunning, done, cancel, shutdownTimeout)

	// Stop the scheduler (waits for any remaining jobs to complete)
	b.dep.GetLogger().SkipSource().Info("Stopping scheduler...")
	if err := scheduler.Shutdown(); err != nil {
		b.dep.GetLogger().SkipSource().Error("Error stopping scheduler", "error", err)
		if runErr == nil {
			runErr = err
		}
	}

	// Close all registered resources
	if closeErr := b.close(); closeErr != nil {
		b.dep.GetLogger().SkipSource().Error("Error during resource cleanup", "error", closeErr)
		if runErr == nil {
			runErr = closeErr
		}
	}

	if runErr == nil {
		b.dep.GetLogger().SkipSource().Info("Scheduler shutdown complete")
	}

	return runErr

}

// createScheduledTask creates a task function for the scheduler
func (b *Bootstrap) createScheduledTask(execute Executable, pattern string, errCh chan error, executeMu *sync.Mutex, executeCancel *context.CancelFunc, executeDone *chan struct{}, isRunning *bool) func() {
	return func() {
		// Create cancellable context for this execution
		executeMu.Lock()
		*isRunning = true
		runCtx, cancel := context.WithCancel(context.Background())
		*executeCancel = cancel
		*executeDone = make(chan struct{})
		executeMu.Unlock()

		// Execute the task
		if err := execute.Execute(runCtx); err != nil {
			b.dep.GetLogger().SkipSource().Errorf("Scheduled execution error, err: %v", err)
			select {
			case errCh <- err:
			default:
				// Channel full, error already sent
			}
		} else {
			b.dep.GetLogger().SkipSource().Infof("Scheduled execution completed successfully, pattern: %s", pattern)
		}

		// Mark execution as done
		executeMu.Lock()
		*isRunning = false
		if *executeDone != nil {
			close(*executeDone)
			*executeDone = nil
		}
		*executeCancel = nil
		executeMu.Unlock()
	}
}

// waitForExecutionComplete waits for a running execution to complete during shutdown
func (b *Bootstrap) waitForExecutionComplete(jobIsRunning bool, done chan struct{}, cancel context.CancelFunc, shutdownTimeout time.Duration) {
	if !jobIsRunning {
		return
	}

	b.dep.GetLogger().SkipSource().Info("Waiting for running execution to complete...", "timeout", shutdownTimeout)

	// Wait for execution to finish or timeout
	timer := time.NewTimer(shutdownTimeout)
	defer timer.Stop()

	select {
	case <-done:
		b.dep.GetLogger().SkipSource().Info("Running execution completed successfully")
	case <-timer.C:
		b.handleExecutionTimeout(done, cancel)
	}
}

// handleExecutionTimeout handles execution timeout during shutdown
func (b *Bootstrap) handleExecutionTimeout(done chan struct{}, cancel context.CancelFunc) {
	b.dep.GetLogger().SkipSource().Warn("Shutdown timeout reached, cancelling running execution")
	if cancel != nil {
		cancel()
	}
	// Give a brief moment for cancellation to take effect
	select {
	case <-done:
		b.dep.GetLogger().SkipSource().Info("Running execution cancelled")
	case <-time.After(1 * time.Second):
		b.dep.GetLogger().SkipSource().Error("Running execution did not respond to cancellation")
	}
}

// parseScheduleOptions parses the schedule options and returns the shutdown timeout and singleton mode
func (b *Bootstrap) parseScheduleOptions(opts []ScheduleOptions) (shutdownTimeout time.Duration, singletonMode bool) {
	shutdownTimeout = 30 * time.Second // default is 30 seconds
	singletonMode = false
	if len(opts) > 0 {
		if opts[0].ShutdownTimeout != 0 {
			shutdownTimeout = opts[0].ShutdownTimeout
		}
		singletonMode = opts[0].SingletonMode
	}
	return shutdownTimeout, singletonMode
}

// Execute runs executable interface and returns without blocking anything
func (b *Bootstrap) Execute(ctx context.Context, execute Executable) error {

	return execute.Execute(ctx)

}

// getClosers gets all closers from the dependency
func (d *Bootstrap) getClosers() []Closer {

	closers := make([]Closer, 0)
	// Get the Dependency struct value
	depVal := reflect.ValueOf(d.dep).Elem()
	depTyp := depVal.Type()

	// Iterate through all fields in Dependency
	for i := 0; i < depVal.NumField(); i++ {
		ft := depTyp.Field(i)

		// Check if the type is a ResourceClosable
		typeName := ft.Type.String()
		if strings.Contains(typeName, "ResourceClosable") {
			// Get the field value
			f := depVal.Field(i)

			// Get the "register" field from ResourceClosable
			registerField := f.FieldByName("register")
			if registerField.IsValid() {
				// Make the field settable using unsafe operations
				registerField = reflect.NewAt(registerField.Type(), registerField.Addr().UnsafePointer()).Elem()

				// Capture the field name in a closure
				fieldName := ft.Name
				registerField.Set(reflect.ValueOf(func(fn func() error) {
					closers = append(closers, Closer{Name: fieldName, Fn: fn})
				}))

				// Handle late registration: if resource was already initialized before
				// register function was set, register it now
				// Get a pointer to the field to ensure we're calling on the actual field
				fieldPtr := reflect.NewAt(f.Type(), f.Addr().UnsafePointer())
				ensureRegisteredMethod := fieldPtr.MethodByName("EnsureRegistered")
				if ensureRegisteredMethod.IsValid() {
					// d.GetLogger().SkipSource().WithParam("name", fieldName).Info("calling ensure registered")
					ensureRegisteredMethod.Call(nil)
				}
			}
		}
	}

	return closers
}

// registerClosers registers all closers
func (b *Bootstrap) registerClosers(cls []Closer) {

	for _, cl := range cls {
		b.closers = append(b.closers, Closer{Name: cl.Name, Fn: cl.Fn})
	}

}

// close closes all registered closers in reverse order
func (b *Bootstrap) close() error {

	b.dep.GetLogger().SkipSource().Infof("Closing %d resources...", len(b.closers))
	b.mu.Lock()
	defer b.mu.Unlock()

	var errs []error
	for i := len(b.closers) - 1; i >= 0; i-- {
		if err := b.closers[i].Fn(); err != nil {
			errs = append(errs, err)
			b.dep.GetLogger().SkipSource().Errorf("%s error closing: %v", b.closers[i].Name, err)
		} else {
			b.dep.GetLogger().SkipSource().Infof("%s is closed successfully", b.closers[i].Name)
		}
	}
	return errors.Join(errs...)

}
