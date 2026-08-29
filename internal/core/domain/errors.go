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

	// ErrEmptyDoctors indicates that no doctors were provided for queue wait calculation.
	ErrEmptyDoctors = errors.New("doctors list must not be empty")

	// ErrInvalidPosition indicates that the queue position is non-positive.
	ErrInvalidPosition = errors.New("queue position must be a positive integer (>= 1)")

	// ErrNilDoctor indicates that a doctor element in slice is nil.
	ErrNilDoctor = errors.New("doctor instance must not be nil")

	// ErrInvalidConsultationTime indicates that average consultation time is non-positive.
	ErrInvalidConsultationTime = errors.New("average consultation time must be greater than 0")

	// ErrActiveTicketExists indicates that the patient already has an active queue ticket.
	ErrActiveTicketExists = errors.New("active queue ticket already exists")

	// ErrTicketNotFound indicates that the requested queue ticket was not found.
	ErrTicketNotFound = errors.New("queue ticket not found")

	// ErrNoDoctorsAvailable indicates that no doctors are configured or available in the clinic.
	ErrNoDoctorsAvailable = errors.New("no doctors currently configured for this clinic")

	// ErrDoctorOffline indicates that the doctor must be online to call patients or perform actions.
	ErrDoctorOffline = errors.New("doctor must be online to call patients")

	// ErrActiveConsultationExists indicates that an active consultation session is already in progress.
	ErrActiveConsultationExists = errors.New("active consultation already in progress")

	// ErrNoActiveConsultation indicates that no active consultation session was found to finish.
	ErrNoActiveConsultation = errors.New("no active consultation found")

	// ErrQueueEmpty indicates that there are currently no waiting patients in the queue.
	ErrQueueEmpty = errors.New("queue is empty")

	// ErrDoctorNotFound indicates that the doctor entity was not found in the repository.
	ErrDoctorNotFound = errors.New("doctor not found")

	// ErrInvalidAction indicates that an invalid or empty audit action was specified.
	ErrInvalidAction = errors.New("invalid or empty audit action")

	// ErrInvalidPage indicates that page number is invalid.
	ErrInvalidPage = errors.New("page must be a positive integer (>= 1)")

	// ErrInvalidLimit indicates that limit number is invalid.
	ErrInvalidLimit = errors.New("limit must be a positive integer (>= 1)")
)
