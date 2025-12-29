package types

import "encoding/json"

type TestCase struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

type SubmissionEvent struct {
	SubmissionID string `json:"submission_id"`
	ProblemID    string `json:"problem_id"`
	Code         string `json:"code"`
	Language     string `json:"language"`
}

type ResultEvent struct {
	SubmissionID string `json:"submission_id"`
	Status       string `json:"status"`
	Message      string `json:"message,omitempty"`
}

func (r *ResultEvent) Marshal() []byte {
	data, _ := json.Marshal(r)
	return data
}
