package utils

import (
	"fmt"

	"github.com/redhajuanda/komon/tracer"
	"gitlab.sicepat.tech/pka/sds/configs"
)

// LocalDebug prints the error stack trace in local environment
func LocalDebug(cfg *configs.Config, err error) {

	// print verbose error stack trace in local environment
	// this is useful for debugging purposes
	if cfg.GetEnv().IsLocal() {
		if stackTracer, ok := err.(tracer.StackTracer); ok {
			fmt.Printf("\n[LOCAL DEBUGGING]\nerror: %s\n", err)
			for _, frame := range stackTracer.StackTrace() {
				fmt.Printf("%+v\n", frame)
			}
		}
	}

}