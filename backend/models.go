package main

import "time"

// ── BNB API structs ───────────────────────────────────────────────────────────

type bnbTokenRequest struct {
	AccountID       string `json:"accountId"`
	AuthorizationID string `json:"authorizationId"`
}

type bnbTokenResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"` // JWT token when success=true
}

type bnbGenerateQRRequest struct {
	Currency             string  `json:"currency"`
	Gloss                string  `json:"gloss"`
	Amount               float64 `json:"amount"`
	SingleUse            bool    `json:"singleUse"`
	ExpirationDate       string  `json:"expirationDate"`
	AdditionalData       string  `json:"additionalData,omitempty"`
	DestinationAccountID string  `json:"destinationAccountId,omitempty"`
}

type bnbGenerateQRResponse struct {
	ID      string `json:"id"`
	QR      string `json:"qr"` // base64 PNG
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type bnbQRStatusRequest struct {
	QRId int `json:"qrId"`
}

type bnbQRStatusResponse struct {
	ID             int    `json:"id"`
	StatusID       int    `json:"statusId"` // 1=No Usado 2=Usado 3=Expirado 4=Error
	ExpirationDate string `json:"expirationDate"`
	VoucherID      string `json:"voucherId"`
	Success        bool   `json:"success"`
	Message        string `json:"message"`
}

type bnbCancelQRRequest struct {
	QRId int `json:"qrId"`
}

type bnbCancelQRResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// BNBNotification es el body que el BNB envía a nuestro webhook cuando un QR es pagado.
type BNBNotification struct {
	QRId                string  `json:"QRId"`
	Gloss               string  `json:"Gloss"`
	SourceBankID        int     `json:"sourceBankId"`
	OriginName          string  `json:"originName"`
	VoucherID           string  `json:"VoucherId"`
	TransactionDateTime string  `json:"TransactionDateTime"`
	AdditionalData      string  `json:"additionalData"`
	Amount              float64 `json:"amount"`
	CurrencyID          int     `json:"currencyId"`
}

// ── Nuestros modelos de pasarela ──────────────────────────────────────────────

// PaymentStatus representa el estado de un pago.
type PaymentStatus string

const (
	StatusPending   PaymentStatus = "pending"
	StatusPaid      PaymentStatus = "paid"
	StatusExpired   PaymentStatus = "expired"
	StatusCancelled PaymentStatus = "cancelled"
)

// Payment es la entidad central de la pasarela.
type Payment struct {
	ID          string        `json:"id"`          // nuestro UUID interno
	QRId        string        `json:"qrId"`        // ID del BNB
	QRImage     string        `json:"qrImage"`     // base64 PNG para mostrar
	Amount      float64       `json:"amount"`
	Currency    string        `json:"currency"`
	Description string        `json:"description"`
	Reference   string        `json:"reference"`   // ID de orden del comercio
	CallbackURL string        `json:"callbackUrl,omitempty"` // redirigir al pagar
	Status      PaymentStatus `json:"status"`
	CreatedAt   time.Time     `json:"createdAt"`
	ExpiresAt   string        `json:"expiresAt"`
	PaidAt      *time.Time    `json:"paidAt,omitempty"`
	VoucherID   string        `json:"voucherId,omitempty"`
	PayerName   string        `json:"payerName,omitempty"`
	SourceBank  int           `json:"sourceBank,omitempty"`
}

// ── Request / Response de nuestra API ────────────────────────────────────────

type CreatePaymentRequest struct {
	Amount      float64 `json:"amount"`
	Currency    string  `json:"currency"`             // "BOB" o "USD"
	Description string  `json:"description"`
	Reference   string  `json:"reference"`            // ID interno del comercio
	ExpiresAt   string  `json:"expiresAt"`            // "YYYY-MM-DD"
	CallbackURL string  `json:"callbackUrl,omitempty"` // redirigir al usuario tras pagar
}

type CreatePaymentResponse struct {
	PaymentID   string        `json:"paymentId"`
	PayURL      string        `json:"payUrl"`              // URL de la pantalla de pago
	QRId        string        `json:"qrId"`
	QRImage     string        `json:"qrImage"`             // base64, mostrar al usuario
	Amount      float64       `json:"amount"`
	Currency    string        `json:"currency"`
	Description string        `json:"description"`
	Status      PaymentStatus `json:"status"`
	ExpiresAt   string        `json:"expiresAt"`
	CallbackURL string        `json:"callbackUrl,omitempty"`
}

// PaymentStatusResponse incluye todos los datos que necesita el frontend.
type PaymentStatusResponse struct {
	PaymentID   string        `json:"paymentId"`
	QRId        string        `json:"qrId"`
	QRImage     string        `json:"qrImage,omitempty"`
	Status      PaymentStatus `json:"status"`
	Amount      float64       `json:"amount"`
	Currency    string        `json:"currency"`
	Description string        `json:"description"`
	ExpiresAt   string        `json:"expiresAt"`
	CallbackURL string        `json:"callbackUrl,omitempty"`
	PaidAt      *time.Time    `json:"paidAt,omitempty"`
	VoucherID   string        `json:"voucherId,omitempty"`
	PayerName   string        `json:"payerName,omitempty"`
	SourceBank  int           `json:"sourceBank,omitempty"`
}

type APIError struct {
	Error string `json:"error"`
}
