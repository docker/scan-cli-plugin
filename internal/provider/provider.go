package provider

// Provider abstracts a scan provider
type Provider interface {
	Authenticate(token string) error
	Scan(image string) error
	Version() (string, error)
}
