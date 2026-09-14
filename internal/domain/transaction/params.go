package transaction

type CreateTransactionParams struct {
	Reference   string
	Type        string
	Status      string
	Description string
	Metadata    string
}

type ListTransactionsParams struct {
	Page   int
	Limit  int
	Type   string
	Status string
}
