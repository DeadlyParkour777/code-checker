package types

import "time"

type Problem struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type TestCase struct {
	ID        string `json:"id"`
	ProblemID string `json:"problem_id"`
	Input     string `json:"input"`
	Output    string `json:"output"`
}

type ProblemEvent struct {
	EventType string   `json:"event_type"`
	Problem   *Problem `json:"problem,omitempty"`
	ProblemID string   `json:"problem_id,omitempty"`
}
