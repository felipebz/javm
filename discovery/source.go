package discovery

import "context"

type Source interface {
	Name() string
	Discover(ctx context.Context) ([]JDK, error)
}
