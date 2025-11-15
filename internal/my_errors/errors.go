package my_errors

import "errors"

// Sentinel my_errors для бизнес-логики
var (
	// User my_errors
	ErrUserNotFound              = errors.New("user not found")
	ErrCannotDeactivateAdmin     = errors.New("cannot deactivate admin users")
	ErrCannotDeactivateAdminTeam = errors.New("cannot deactivate admin team")
	ErrUserIsNotActive           = errors.New("user is not active")

	// Team my_errors
	ErrTeamAlreadyExists = errors.New("team already exists")
	ErrTeamNotFound      = errors.New("team not found")

	// PR my_errors
	ErrPRNotFound      = errors.New("pull request not found")
	ErrPRAlreadyMerged = errors.New("pull request already merged")
	ErrPRAlreadyExists = errors.New("pull request already exists")
	ErrAuthorNotFound  = errors.New("author not found")

	// Reviewer my_errors
	ErrNoActiveReviewerWasFound = errors.New("no active replacement candidate in team")
	ErrReviewerIsNotAssigned    = errors.New("reviewer is not assigned to this PR")

	// Auth my_errors
	ErrInvalidToken  = errors.New("invalid token")
	ErrTokenMismatch = errors.New("token mismatch")

	// Validation my_errors
	ErrInvalidInput = errors.New("invalid input")
	ErrEmptyField   = errors.New("required field is empty")
)
