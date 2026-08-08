package command

import (
	"context"
	"io"

	log "github.com/sirupsen/logrus"
)

// Runtime contains the process-facing dependencies used while executing a
// command. Keeping them in the command context lets blocking helpers share the
// same diagnostic stream without package-global mutable state.
type Runtime struct {
	Logger       *log.Logger
	Err          io.Writer
	ShowProgress bool
}

type runtimeContextKey struct{}

// WithRuntime associates command output and diagnostics with an execution context.
func WithRuntime(ctx context.Context, runtime Runtime) context.Context {
	return context.WithValue(ctx, runtimeContextKey{}, runtime)
}

// RuntimeFromContext returns the runtime associated with ctx. When no runtime
// is present, diagnostics are discarded and progress is disabled.
func RuntimeFromContext(ctx context.Context) Runtime {
	if runtime, ok := ctx.Value(runtimeContextKey{}).(Runtime); ok {
		if runtime.Err == nil {
			runtime.Err = io.Discard
		}
		if runtime.Logger == nil {
			runtime.Logger = discardLogger()
		}
		return runtime
	}
	return Runtime{Logger: discardLogger(), Err: io.Discard}
}

func loggerFromContext(ctx context.Context) *log.Logger {
	return RuntimeFromContext(ctx).Logger
}

func discardLogger() *log.Logger {
	logger := log.New()
	logger.SetOutput(io.Discard)
	return logger
}
