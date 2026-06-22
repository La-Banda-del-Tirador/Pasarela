package main

import (
	"log"
	"net/http"
	"os"
)

// corsMiddleware permite llamadas desde el frontend (iframe, redirect, dev server).
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func main() {
	// ── Configuración desde variables de entorno ───────────────────────────────
	bnbURL := getenv("BNB_BASE_URL", "http://test.bnb.com.bo")
	accountID := mustEnv("BNB_ACCOUNT_ID")
	authorizationID := mustEnv("BNB_AUTHORIZATION_ID")
	port := getenv("PORT", "8080")

	// ── Inicializar dependencias ───────────────────────────────────────────────
	app := &App{
		bnb:   NewBNBClient(bnbURL, accountID, authorizationID),
		store: NewStore(),
	}

	// ── Rutas ─────────────────────────────────────────────────────────────────
	mux := http.NewServeMux()

	// Pagos
	mux.HandleFunc("/api/payments", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			app.CreatePayment(w, r)
		default:
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/payments/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			app.GetPayment(w, r)
		case http.MethodDelete:
			app.CancelPayment(w, r)
		default:
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
		}
	})

	// Webhook — el BNB llama a esta URL cuando un QR es pagado
	mux.HandleFunc("/webhook/notification", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "método no permitido", http.StatusMethodNotAllowed)
			return
		}
		app.ReceiveNotification(w, r)
	})

	// Health check
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Frontend React — busca ../frontend/dist (desde backend/) o frontend/dist (desde raíz).
	// En Docker, nginx sirve el frontend directamente; el backend no necesita esta ruta.
	// Sobreescribible con la variable FRONTEND_DIST.
	frontendDist := os.Getenv("FRONTEND_DIST")
	if frontendDist == "" {
		for _, p := range []string{"../frontend/dist", "frontend/dist"} {
			if _, err := os.Stat(p); err == nil {
				frontendDist = p
				break
			}
		}
	}
	if frontendDist != "" {
		mux.Handle("/assets/", http.FileServer(http.Dir(frontendDist)))
		mux.HandleFunc("/pay/", func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, frontendDist+"/index.html")
		})
		log.Printf("  GET    /pay/{id}                   → pantalla de pago (%s)", frontendDist)
	} else {
		log.Printf("  [frontend] no encontrado — usa Docker o FRONTEND_DIST env var")
	}

	log.Printf("Pasarela BNB iniciada en :%s", port)
	log.Printf("Webhook URL: http://TU-DOMINIO:%s/webhook/notification", port)
	log.Printf("  POST   /api/payments              → crear pago (genera QR + payUrl)")
	log.Printf("  GET    /api/payments/{id}          → estado del pago")
	log.Printf("  DELETE /api/payments/{id}          → cancelar pago")
	log.Printf("  POST   /webhook/notification       → notificacion del BNB")

	if err := http.ListenAndServe(":"+port, corsMiddleware(mux)); err != nil {
		log.Fatal(err)
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("variable de entorno requerida: %s", key)
	}
	return v
}
