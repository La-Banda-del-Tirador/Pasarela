package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// BNBClient maneja la comunicación con la API del BNB.
// El token se cachea y se renueva automáticamente antes de expirar.
type BNBClient struct {
	baseURL         string
	accountID       string
	authorizationID string

	mu          sync.Mutex
	cachedToken string
	tokenExpiry time.Time

	httpClient *http.Client
}

func NewBNBClient(baseURL, accountID, authorizationID string) *BNBClient {
	return &BNBClient{
		baseURL:         baseURL,
		accountID:       accountID,
		authorizationID: authorizationID,
		httpClient:      &http.Client{Timeout: 30 * time.Second},
	}
}

// token retorna el JWT vigente, renovándolo si está por vencer.
func (c *BNBClient) token() (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Reutilizar token si le quedan más de 5 minutos de vida
	if c.cachedToken != "" && time.Now().Before(c.tokenExpiry.Add(-5*time.Minute)) {
		return c.cachedToken, nil
	}

	body, _ := json.Marshal(bnbTokenRequest{
		AccountID:       c.accountID,
		AuthorizationID: c.authorizationID,
	})

	resp, err := c.httpClient.Post(
		c.baseURL+"/ClientAuthentication.API/api/v1/auth/token",
		"application/json",
		bytes.NewBuffer(body),
	)
	if err != nil {
		return "", fmt.Errorf("bnb auth request: %w", err)
	}
	defer resp.Body.Close()

	var tr bnbTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", fmt.Errorf("bnb auth decode: %w", err)
	}
	if !tr.Success {
		return "", fmt.Errorf("bnb auth failed: %s", tr.Message)
	}

	c.cachedToken = tr.Message
	c.tokenExpiry = time.Now().Add(1 * time.Hour)
	return c.cachedToken, nil
}

// post ejecuta una llamada autenticada a la API del BNB.
func (c *BNBClient) post(path string, reqBody, respBody interface{}) error {
	tok, err := c.token()
	if err != nil {
		return err
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, c.baseURL+path, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+tok)
	req.Header.Set("cache-control", "no-cache")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("bnb request %s: %w", path, err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, respBody)
}

// GenerateQR genera un QR de cobro simple y devuelve la respuesta del BNB.
// Reintenta hasta 3 veces ante fallos intermitentes del sandbox.
func (c *BNBClient) GenerateQR(req bnbGenerateQRRequest) (*bnbGenerateQRResponse, error) {
	var (
		resp    bnbGenerateQRResponse
		lastErr error
	)
	for attempt := 1; attempt <= 3; attempt++ {
		resp = bnbGenerateQRResponse{}
		err := c.post("/QRSimple.API/api/v1/main/getQRWithImageAsync", req, &resp)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		if !resp.Success {
			lastErr = fmt.Errorf("bnb GenerateQR: %s", resp.Message)
			time.Sleep(time.Duration(attempt) * 500 * time.Millisecond)
			continue
		}
		return &resp, nil
	}
	return nil, lastErr
}

// GetQRStatus consulta el estado actual de un QR.
func (c *BNBClient) GetQRStatus(qrID int) (*bnbQRStatusResponse, error) {
	var resp bnbQRStatusResponse
	if err := c.post("/QRSimple.API/api/v1/main/getQRStatusAsync", bnbQRStatusRequest{QRId: qrID}, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// CancelQR anula un QR de uso único que aún no fue utilizado.
func (c *BNBClient) CancelQR(qrID int) (*bnbCancelQRResponse, error) {
	var resp bnbCancelQRResponse
	if err := c.post("/QRSimple.API/api/v1/main/CancelQRByIdAsync", bnbCancelQRRequest{QRId: qrID}, &resp); err != nil {
		return nil, err
	}
	if !resp.Success {
		return nil, fmt.Errorf("bnb CancelQR: %s", resp.Message)
	}
	return &resp, nil
}

// qrIDFromString convierte el string ID que devuelve el BNB a int.
func qrIDFromString(s string) (int, error) {
	id, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid qrId %q: %w", s, err)
	}
	return id, nil
}
