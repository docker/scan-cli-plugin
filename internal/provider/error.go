package provider

import "fmt"

type authenticationError struct {
}

func (a authenticationError) Error() string {
	return "authentication error"
}

// IsAuthenticationError check if the error type is an authentication error
func IsAuthenticationError(err error) bool {
	_, ok := err.(*authenticationError)
	return ok
}

type invalidTokenError struct {
	token string
}

func (i invalidTokenError) Error() string {
	return fmt.Sprintf("invalid authentication token %q", i.token)
}

// IsInvalidTokenError check if the error type is an invalid token error
func IsInvalidTokenError(err error) bool {
	_, ok := err.(*invalidTokenError)
	return ok
}
