package models

import (
	"time"
)

type Task struct {
	ID          string    `json:"id"`
	JobType     string    `json:"job_type"`     // "short" or "long"
	Payload     string    `json:"payload"`      // task-specific data
	Status      string    `json:"status"`       // "queued", "running", "success", "failed"
	SubmittedAt time.Time `json:"submitted_at"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	RetryCount  int       `json:"retry_count"`
	Result      string    `json:"result,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// TaskRequest represents the request body for submitting a task
type TaskRequest struct {
	ID      string `json:"id,omitempty"` // Optional: client can provide ID for idempotency
	JobType string `json:"job_type" binding:"required"`
	Payload string `json:"payload"`
}