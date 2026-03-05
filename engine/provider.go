package engine

// Provider is the interface that all search backends must implement.
type Provider interface {
	// Search executes a search query and returns results for the given page.
	Search(query string, page int) ([]Result, error)
	// Name returns the provider's display name.
	Name() string
}
