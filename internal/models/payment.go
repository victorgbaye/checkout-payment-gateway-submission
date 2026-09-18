package models

type PostPaymentRequest struct {
	CardNumber  string `json:"card_number" binding:"required" minLength:"14" maxLength:"19" example:"2222405343248877"`
	ExpiryMonth int    `json:"expiry_month" binding:"required" minimum:"1" maximum:"12" example:"12"`
	ExpiryYear  int    `json:"expiry_year" binding:"required" minimum:"1" maximum:"9999" example:"2099"`
	Currency    string `json:"currency" binding:"required" enums:"GBP,USD,EUR" example:"GBP"`
	Amount      int    `json:"amount" binding:"required" minimum:"1" example:"100"`
	Cvv         string `json:"cvv" binding:"required" minLength:"3" maxLength:"4" example:"012"`
}

type PostPaymentResponse struct {
	Id                 string `json:"id"`
	PaymentStatus      string `json:"payment_status" enums:"Authorized,Declined" example:"Authorized"`
	CardNumberLastFour string `json:"card_number_last_four" minLength:"4" maxLength:"4" example:"8877"`
	ExpiryMonth        int    `json:"expiry_month"`
	ExpiryYear         int    `json:"expiry_year"`
	Currency           string `json:"currency"`
	Amount             int    `json:"amount"`
}

type GetPaymentResponse struct {
	Id                 string `json:"id"`
	PaymentStatus      string `json:"payment_status" enums:"Authorized,Declined" example:"Authorized"`
	CardNumberLastFour string `json:"card_number_last_four" minLength:"4" maxLength:"4" example:"8877"`
	ExpiryMonth        int    `json:"expiry_month"`
	ExpiryYear         int    `json:"expiry_year"`
	Currency           string `json:"currency"`
	Amount             int    `json:"amount"`
}

// ErrorResponse documents the JSON error shape returned by payment handlers.
type ErrorResponse struct {
	Error string `json:"error" binding:"required" example:"bank unavailable"`
}

// RejectedPaymentResponse documents input failures before bank processing.
type RejectedPaymentResponse struct {
	Error         string `json:"error" binding:"required" example:"invalid payment: amount must be positive"`
	PaymentStatus string `json:"payment_status" binding:"required" enums:"Rejected" example:"Rejected"`
}
