package model

import "time"

type Transaction struct {
	ID          string    `json:"id"`
	Reference   string    `json:"reference"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Metadata    any       `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
}
