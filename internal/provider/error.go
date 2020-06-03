package provider

type authenticationError struct {
}

func (authError authenticationError) Error() string {
	return "authentication error"
}

// IsAuthenticationError check if the error type is an authentication error
func IsAuthenticationError(err error) bool {
	_, ok := err.(*authenticationError)
	return ok
}
