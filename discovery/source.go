package discovery

type Source interface {
	Name() string
	Discover() ([]JDK, error)
	Enabled(*Config) bool
}
