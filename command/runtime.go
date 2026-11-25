package command

import (
	"context"
	"io"
	"sync"

	log "github.com/sirupsen/logrus"
)

// Runtime contains the process-facing dependencies used while executing a
// command. Keeping them in the command context lets blocking helpers share the
// same diagnostic stream without package-global mutable state.
type Runtime struct {
	Logger       *log.Logger
	Err          io.Writer
	ShowProgress bool
	UserAgent    string

	// warned holds the warnings already logged for this command, so helpers
	// that list JDKs several times report each problem once.
	warned *sync.Map
}

type runtimeContextKey struct{}

// WithRuntime associates command output and diagnostics with an execution context.
func WithRuntime(ctx context.Context, runtime Runtime) context.Context {
	if runtime.warned == nil {
		runtime.warned = &sync.Map{}
	}
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

// warnOnce logs err as a warning unless the same warning was already logged
// for this command.
func warnOnce(ctx context.Context, err error) {
	runtime := RuntimeFromContext(ctx)
	if runtime.warned != nil {
		if _, seen := runtime.warned.LoadOrStore(err.Error(), struct{}{}); seen {
			return
		}
	}
	runtime.Logger.Warn(err)
}

func discardLogger() *log.Logger {
	logger := log.New()
	logger.SetOutput(io.Discard)
	return logger
}
