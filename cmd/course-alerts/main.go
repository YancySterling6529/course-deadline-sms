package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/infrai-examples/course-deadline-sms/internal/alerts"
	"github.com/infrai-examples/course-deadline-sms/internal/infrai"
)

func main() {
	key := os.Getenv("INFRAI_API_KEY")
	if key == "" {
		log.Fatal("INFRAI_API_KEY is required")
	}
	dispatcher := alerts.Dispatcher{
		Sender: &infrai.SMSClient{APIKey: key, MaxRetries: 3},
		Now:    time.Now,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("POST /course-deadline-alerts", func(w http.ResponseWriter, r *http.Request) {
		var input alerts.Request
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&input); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid request body"})
			return
		}
		report, err := dispatcher.Dispatch(r.Context(), input)
		if err != nil {
			var apiErr *infrai.APIError
			message := "sms delivery request failed"
			if errors.As(err, &apiErr) {
				message = apiErr.Error()
			}
			writeJSON(w, infrai.ClientStatus(err), map[string]string{"error": message})
			return
		}
		writeJSON(w, http.StatusOK, report)
	})

	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":8080"
	}
	log.Printf("course alert service listening on %s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode response: %v", err)
	}
}
