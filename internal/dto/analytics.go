package dto

type AnalyticsResponse struct {
	IncomeThisMonth  string `json:"income_this_month"`
	ExpenseThisMonth string `json:"expense_this_month"`
	CreditLoad       string `json:"credit_load"`
}

type OperationStatisticsResponse struct {
	EntityType     string                     `json:"entity_type"`
	EntityID       int64                      `json:"entity_id"`
	Currency       string                     `json:"currency"`
	OperationCount int64                      `json:"operation_count"`
	IncomeCount    int64                      `json:"income_count"`
	ExpenseCount   int64                      `json:"expense_count"`
	TotalIncome    string                     `json:"total_income"`
	TotalExpense   string                     `json:"total_expense"`
	NetAmount      string                     `json:"net_amount"`
	ByType         []OperationTypeResponse    `json:"by_type"`
	ByStatus       []OperationStatusResponse  `json:"by_status"`
	Operations     []OperationHistoryResponse `json:"operations"`
}

type OperationTypeResponse struct {
	Type         string `json:"type"`
	Count        int64  `json:"count"`
	TotalIncome  string `json:"total_income"`
	TotalExpense string `json:"total_expense"`
	NetAmount    string `json:"net_amount"`
}

type OperationStatusResponse struct {
	Status      string `json:"status"`
	Count       int64  `json:"count"`
	TotalAmount string `json:"total_amount"`
}

type OperationHistoryResponse struct {
	ID            int64  `json:"id"`
	Direction     string `json:"direction"`
	Type          string `json:"type"`
	Status        string `json:"status"`
	Amount        string `json:"amount"`
	Currency      string `json:"currency"`
	Description   string `json:"description,omitempty"`
	FromAccountID *int64 `json:"from_account_id,omitempty"`
	ToAccountID   *int64 `json:"to_account_id,omitempty"`
	FromCardID    *int64 `json:"from_card_id,omitempty"`
	ToCardID      *int64 `json:"to_card_id,omitempty"`
	CreatedAt     string `json:"created_at"`
}
