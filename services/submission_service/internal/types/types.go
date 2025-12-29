package types

import (
	"time"
)

type Submission struct {
	ID        string    `json:"id"`
	ProblemID string    `json:"problem_id"`
	UserID    string    `json:"user_id"`
	Code      string    `json:"code"`
	Language  string    `json:"language"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type SubmissionEvent struct {
	SubmissionID string `json:"submission_id"`
	ProblemID    string `json:"problem_id"`
	Code         string `json:"code"`
	Language     string `json:"language"`
}

type UserLoginPayload struct {
	Username string
	Password string
}

type UserRegisterPayload struct {
	Username string
	Password string
}

type User struct {
	ID        string
	Username  string
	Password  string
	Role      string
	CreatedAt time.Time
}
