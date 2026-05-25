package contracts

import "time"

type CreateContractInput struct {
	TransactionID int32      `json:"transaction_id"`
	PeriodID      *int32     `json:"period_id"` // Frecuencia de pago (opcional para ventas)
	Currency      string     `json:"currency"`
	AgreedAmount  float64    `json:"agreed_amount"`
	StartDate     time.Time  `json:"start_date"`
	EndDate       *time.Time `json:"end_date"`
}

type CreateContractResult struct {
	ContractID   int32  `json:"contract_id"`
	ContractUUID string `json:"contract_uuid"`
	StorageKey   string `json:"storage_key"`
	PDFUrl       string `json:"pdf_url"`
}

type ContractDetail struct {
	ContractUUID  string     `json:"contract_uuid"`
	PropertyTitle string     `json:"property_title"`
	OwnerName     string     `json:"owner_name"`
	ClientName    string     `json:"client_name"`
	AgreedAmount  float64    `json:"agreed_amount"`
	Currency      string     `json:"currency"`
	StartDate     time.Time  `json:"start_date"`
	EndDate       *time.Time `json:"end_date"`
	Status        string     `json:"status"`
	PDFUrl        string     `json:"pdf_url"`
}

type ListContractsFilter struct {
	OwnerID         *int32     `json:"owner_id"`
	TransactionType *string    `json:"transaction_type"`
	StatusID        *int32     `json:"status_id"`
	StartDate       *time.Time `json:"start_date"`
	EndDate         *time.Time `json:"end_date"`
	Search          *string    `json:"search"`
	Page            int32      `json:"page"`
	Limit           int32      `json:"limit"`
}

type ContractListItem struct {
	ContractUUID    string    `json:"contract_uuid"`
	TransactionType string    `json:"transaction_type"`
	PropertyTitle   string    `json:"property_title"`
	AgreedAmount    float64   `json:"agreed_amount"`
	Currency        string    `json:"currency"`
	StartDate       time.Time `json:"start_date"`
	Status          string    `json:"status"`
	ClientName      string    `json:"client_name"`
	CreatedAt       time.Time `json:"created_at"`
}
