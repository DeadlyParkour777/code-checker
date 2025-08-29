package types

import (
	"time"
)

type ResultEvent struct {
	SubmissionID string `json:"submission_id"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
}

type Submission struct {
	ID        string
	ProblemID string
	UserID    string
	Language  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
