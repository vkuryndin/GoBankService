package dto

type AnalyticsResponse struct {
	IncomeThisMonth  string `json:"income_this_month"`
	ExpenseThisMonth string `json:"expense_this_month"`
	CreditLoad       string `json:"credit_load"`
}
