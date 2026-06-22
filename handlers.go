package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"
)

// App agrupa las dependencias compartidas por todos los handlers.
type App struct {
	bnb   *BNBClient
	store *Store
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, APIError{Error: msg})
}

// ── POST /api/payments ────────────────────────────────────────────────────────
// Crea un pago: genera el QR en el BNB y lo almacena.

func (a *App) CreatePayment(w http.ResponseWriter, r *http.Request) {
	var req CreatePaymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "body inválido: "+err.Error())
		return
	}

	// Validaciones básicas
	if req.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "amount debe ser mayor a 0")
		return
	}
	if req.Currency != "BOB" && req.Currency != "USD" {
		writeError(w, http.StatusBadRequest, "currency debe ser BOB o USD")
		return
	}
	if req.ExpiresAt == "" {
		req.ExpiresAt = time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	}
	if req.Description == "" {
		writeError(w, http.StatusBadRequest, "description es requerido")
		return
	}

	// Generar QR en el BNB usando reference como additionalData
	// para poder identificar el pedido cuando llegue el webhook.
	// destinationAccountId es opcional — la cuenta sandbox personal no lo requiere.
	bnbResp, err := a.bnb.GenerateQR(bnbGenerateQRRequest{
		Currency:       req.Currency,
		Gloss:          req.Description,
		Amount:         req.Amount,
		SingleUse:      true,
		ExpirationDate: req.ExpiresAt,
		AdditionalData: req.Reference,
	})
	if err != nil {
		log.Printf("ERROR GenerateQR: %v", err)
		writeError(w, http.StatusBadGateway, "error generando QR en BNB: "+err.Error())
		return
	}

	payment := &Payment{
		ID:          newID(),
		QRId:        bnbResp.ID,
		QRImage:     bnbResp.QR,
		Amount:      req.Amount,
		Currency:    req.Currency,
		Description: req.Description,
		Reference:   req.Reference,
		CallbackURL: req.CallbackURL,
		Status:      StatusPending,
		CreatedAt:   time.Now(),
		ExpiresAt:   req.ExpiresAt,
	}
	a.store.Save(payment)

	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	payURL := fmt.Sprintf("%s://%s/pay/%s", scheme, r.Host, payment.ID)

	log.Printf("Payment creado: id=%s qrId=%s amount=%.2f %s payUrl=%s", payment.ID, payment.QRId, payment.Amount, payment.Currency, payURL)

	writeJSON(w, http.StatusCreated, CreatePaymentResponse{
		PaymentID:   payment.ID,
		PayURL:      payURL,
		QRId:        payment.QRId,
		QRImage:     payment.QRImage,
		Amount:      payment.Amount,
		Currency:    payment.Currency,
		Description: payment.Description,
		Status:      payment.Status,
		ExpiresAt:   payment.ExpiresAt,
		CallbackURL: payment.CallbackURL,
	})
}

// ── GET /api/payments/{id} ────────────────────────────────────────────────────
// Consulta el estado de un pago. También sincroniza con el BNB si está pendiente.

func (a *App) GetPayment(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/payments/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "falta el id del pago")
		return
	}

	payment, ok := a.store.GetByID(id)
	if !ok {
		writeError(w, http.StatusNotFound, "pago no encontrado")
		return
	}

	// Si sigue pendiente, sincronizar con BNB para detectar pagos que
	// llegaron sin notificación (failsafe)
	if payment.Status == StatusPending {
		qrID, err := qrIDFromString(payment.QRId)
		if err == nil {
			if status, err := a.bnb.GetQRStatus(qrID); err == nil {
				switch status.StatusID {
				case 2: // Usado
					if payment.Status != StatusPaid {
						now := time.Now()
						payment.Status = StatusPaid
						payment.PaidAt = &now
						payment.VoucherID = status.VoucherID
						a.store.Update(payment)
					}
				case 3: // Expirado
					payment.Status = StatusExpired
					a.store.Update(payment)
				}
			}
		}
	}

	writeJSON(w, http.StatusOK, PaymentStatusResponse{
		PaymentID:   payment.ID,
		QRId:        payment.QRId,
		QRImage:     payment.QRImage,
		Status:      payment.Status,
		Amount:      payment.Amount,
		Currency:    payment.Currency,
		Description: payment.Description,
		ExpiresAt:   payment.ExpiresAt,
		CallbackURL: payment.CallbackURL,
		PaidAt:      payment.PaidAt,
		VoucherID:   payment.VoucherID,
		PayerName:   payment.PayerName,
		SourceBank:  payment.SourceBank,
	})
}

// ── DELETE /api/payments/{id} ─────────────────────────────────────────────────
// Cancela un QR pendiente.

func (a *App) CancelPayment(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/payments/")
	if id == "" {
		writeError(w, http.StatusBadRequest, "falta el id del pago")
		return
	}

	payment, ok := a.store.GetByID(id)
	if !ok {
		writeError(w, http.StatusNotFound, "pago no encontrado")
		return
	}
	if payment.Status != StatusPending {
		writeError(w, http.StatusConflict, "solo se pueden cancelar pagos en estado pending")
		return
	}

	qrID, err := qrIDFromString(payment.QRId)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	if _, err := a.bnb.CancelQR(qrID); err != nil {
		log.Printf("ERROR CancelQR %s: %v", payment.QRId, err)
		writeError(w, http.StatusBadGateway, "error cancelando QR en BNB: "+err.Error())
		return
	}

	payment.Status = StatusCancelled
	a.store.Update(payment)

	log.Printf("Payment cancelado: id=%s qrId=%s", payment.ID, payment.QRId)
	writeJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

// ── POST /webhook/notification ────────────────────────────────────────────────
// El BNB llama a este endpoint cuando un QR es pagado.
// Debe responder {"success": true, "message": "OK"} para que el BNB no reintente.

func (a *App) ReceiveNotification(w http.ResponseWriter, r *http.Request) {
	var notif BNBNotification
	if err := json.NewDecoder(r.Body).Decode(&notif); err != nil {
		log.Printf("WEBHOOK error decode: %v", err)
		writeError(w, http.StatusBadRequest, "body inválido")
		return
	}

	log.Printf("WEBHOOK recibido: QRId=%s amount=%.2f voucher=%s payer=%s",
		notif.QRId, notif.Amount, notif.VoucherID, notif.OriginName)

	payment, ok := a.store.GetByQRId(notif.QRId)
	if !ok {
		// QR desconocido — igual responder OK para que el BNB no reintente
		log.Printf("WEBHOOK QRId=%s no encontrado en store", notif.QRId)
		writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "OK"})
		return
	}

	if payment.Status == StatusPending {
		now := time.Now()
		payment.Status = StatusPaid
		payment.PaidAt = &now
		payment.VoucherID = notif.VoucherID
		payment.PayerName = notif.OriginName
		payment.SourceBank = notif.SourceBankID
		a.store.Update(payment)

		log.Printf("Payment PAGADO: id=%s qrId=%s voucher=%s payer=%s",
			payment.ID, payment.QRId, payment.VoucherID, payment.PayerName)
	}

	// Respuesta obligatoria para el BNB
	writeJSON(w, http.StatusOK, map[string]interface{}{"success": true, "message": "OK"})
}
