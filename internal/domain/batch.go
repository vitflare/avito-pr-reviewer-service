package domain

import "time"

type BatchDeactivateResult struct {
	DeactivatedUsers []string
	ReassignedPRs    []PRReassignment
	SkippedUsers     []string
	ProcessingTime   time.Duration
}

type ReassignmentTask struct {
	PrID             string
	AuthorID         string
	DeactivatedRevs  []string
	TeamName         string
	CurrentReviewers []string
}
