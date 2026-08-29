package domain

import "errors"

var (
	// ErrInvalidCredentials indicates that the provided username or password was incorrect.
	ErrInvalidCredentials = errors.New("invalid username or password")

	// ErrUsernameTaken indicates that a user attempted to register with an existing username.
	ErrUsernameTaken = errors.New("username is already taken")

	// ErrUserNotFound indicates that the requested user entity could not be found.
	ErrUserNotFound = errors.New("user not found")

	// ErrInvalidRole indicates that an unsupported or invalid role was provided.
	ErrInvalidRole = errors.New("invalid user role")

	// ErrDoctorProfileMissing indicates that a doctor user account has no associated doctor record.
	ErrDoctorProfileMissing = errors.New("doctor profile not associated with user")

	// ErrInvalidInput indicates that input parameters failed validation.
	ErrInvalidInput = errors.New("invalid input parameters")
)
