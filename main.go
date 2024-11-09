package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type ResultVolume struct {
	SpecificVolumeLiquid float64 `json:"specific_volume_liquid"`
	SpecificVolumeVapor  float64 `json:"specific_volume_vapor"`
}

func main() {
	r := mux.NewRouter()
	rc := routesConfig(r)

	r.HandleFunc("/phase-change-diagram", func(w http.ResponseWriter, r *http.Request) {

		queryPressure := r.URL.Query().Get("pressure")
		pressure, err := strconv.ParseFloat(queryPressure, 64)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		result, errVol := getVolumes(pressure)
		if errVol != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(&result)
	}).Methods("GET")

	s := &http.Server{
		Addr:         ":3000",
		Handler:      rc,
		IdleTimeout:  120 * time.Second,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	go func() {
		fmt.Println("Starting server on port 3000")

		err := s.ListenAndServe()
		if err != nil {
			fmt.Printf("Error starting server: %s\n", err)
			os.Exit(1)
		}
	}()

	// trap sigterm or interupt and gracefully shutdown the server
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, os.Kill)

	// Block until a signal is received.
	sig := <-c
	fmt.Println("Got signal:", sig)

	// gracefully shutdown the server, waiting max 30 seconds for current operations to complete
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	s.Shutdown(ctx)
}

func routesConfig(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*") // Allow any origin
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Content-Type", "application/json")

		// Check if the request is for CORS preflight
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Pass down the request to the next middleware (or final handler)
		next.ServeHTTP(w, r)
	})
}

func getVolumes(pressure float64) (*ResultVolume, error) {
	if pressure > 10 || pressure < 0.05 {
		return nil, errors.New("incorrect pressure")
	}

	volLiquid := (2450*pressure + 10325) / 9950000
	volVapor := (299999825 - 29996500*pressure) / 9950000

	return &ResultVolume{
		SpecificVolumeLiquid: volLiquid,
		SpecificVolumeVapor:  volVapor,
	}, nil
}
