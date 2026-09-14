package transaction

import (
	"time"
)

type Entry struct {
	ID            string    `json:"id"`
	TransactionID string    `json:"transaction_id"`
	AccountID     string    `json:"account_id"`
	Type          string    `json:"type"`
	Amount        int64     `json:"amount"`
	EntrySequence int64     `json:"entry_sequence"`
	CreatedAt     time.Time `json:"created_at"`
}

type Transaction struct {
	ID          string    `json:"id"`
	Reference   string    `json:"reference"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Metadata    any       `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
	Entries     []Entry   `json:"entries"`
}
