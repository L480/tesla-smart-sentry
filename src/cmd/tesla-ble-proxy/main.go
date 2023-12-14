package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"context"

	"github.com/L480/tesla-smart-sentry/internal/logger"
	"github.com/L480/tesla-smart-sentry/internal/tesla"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
)

var contextTimeout time.Duration = 15 * time.Second

func parseBody(r *http.Request, w http.ResponseWriter) (tesla.Config, error) {
	type certificate struct {
		Vin        string `json:"vin"`
		PrivateKey string `json:"private_key"`
	}
	var jsonData certificate
	body, _ := io.ReadAll(r.Body)
	json.Unmarshal(body, &jsonData)
	if jsonData.Vin == "" || jsonData.PrivateKey == "" {
		w.WriteHeader(http.StatusBadRequest)
		return tesla.Config{}, fmt.Errorf("json property missing")
	} else {
		privateKeyFile, _ := os.CreateTemp("/dev/shm", "key")
		privateKeyFile.Write([]byte(jsonData.PrivateKey))
		return tesla.Config{
			Vin:            jsonData.Vin,
			PrivateKeyFile: privateKeyFile.Name(),
			Ble:            true,
		}, nil
	}
}

func main() {
	logger.SetLevel(logger.LevelDebug)
	port := "8080"
	if len(os.Args) == 2 {
		port = os.Args[1]
	}
	router := mux.NewRouter()
	srv := &http.Server{
		Handler:      router,
		Addr:         "0.0.0.0:" + port,
		WriteTimeout: 30 * time.Second,
		ReadTimeout:  30 * time.Second,
	}

	router.HandleFunc("/sentry-mode/on", func(w http.ResponseWriter, r *http.Request) {
		t, _ := parseBody(r, w)
		defer os.Remove(t.PrivateKeyFile)
		ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
		defer cancel()
		if err := tesla.Execute(ctx, t, true, func(car *vehicle.Vehicle) error {
			return car.Wakeup(ctx)
		}); err != nil {
			logger.Error("Failed to wake up vehicle: %s", err)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		ctx, cancel = context.WithTimeout(context.Background(), contextTimeout)
		defer cancel()
		if err := tesla.Execute(ctx, t, false, func(car *vehicle.Vehicle) error {
			return car.SetSentryMode(ctx, true)
		}); err != nil {
			logger.Error("Failed to enable Sentry Mode: %s", err)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		logger.Info("Sentry Mode is enabled.")
	}).Methods("POST")

	router.HandleFunc("/sentry-mode/off", func(w http.ResponseWriter, r *http.Request) {
		t, _ := parseBody(r, w)
		defer os.Remove(t.PrivateKeyFile)
		ctx, cancel := context.WithTimeout(context.Background(), contextTimeout)
		defer cancel()
		if err := tesla.Execute(ctx, t, false, func(car *vehicle.Vehicle) error {
			return car.SetSentryMode(ctx, false)
		}); err != nil {
			logger.Error("Failed to disable Sentry Mode: %s", err)
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		logger.Info("Sentry Mode is disabled.")
	}).Methods("POST")

	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	router.Use(func(next http.Handler) http.Handler { return handlers.LoggingHandler(os.Stdout, next) })

	logger.Info("Waiting for requests on port %s.", port)
	log.Fatal(srv.ListenAndServe())
}
