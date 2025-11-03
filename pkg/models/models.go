package models

import "time"

type Chunk struct {
	ID         string    `json:"id"`
	Repository string    `json:"repository"`
	Ref        string    `json:"ref"`
	Path       string    `json:"path"`
	Language   string    `json:"language"`
	Summary    string    `json:"summary"`
	Content    string    `json:"content"`
	LineStart  int       `json:"line_start"`
	LineEnd    int       `json:"line_end"`
	CreatedAt  time.Time `json:"created_at"`
}

type SearchResult struct {
	Chunk Chunk   `json:"chunk"`
	Score float64 `json:"score"`
}
