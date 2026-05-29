package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

func BuildRouter() (*ServeMux, error) {
	router := NewServeMux()
	router.HandlerFunc("GET /", indexHandler)
	router.HandlerFunc("GET /panic", panicHandler)
	router.HandlerFunc("POST /greet/json", greetHandler)
}

func indexHandler(w http.ResponseWriter, _ *http.Request) {
	w.Write([]byte("Hello, World!\r\n\r\n"))
}

func panicHandler(_ http.ResponseWriter, _ *http.Request) {
	panic("oh no! A picnic!")
}

func greetHandler(w http.ResponseWriter, r *http.Request) {
	var req struct {
		First, Last string
		Age         int
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, err, http.StatusBadRequest)
		return
	}

	if req.Age < 0 {
		WriteError(w, errors.New("age must be >= 0"), http.StatusBadRequest)
	}

	var category string
	switch {
	case req.Age < 13:
		WriteError(w, errors.New("forbidden: come back when you're older"), http.StatusForbidden)
	case req.Age < 21:
		category = "teenager"
	case req.Age > 65:
		category = "senior"
	default:
		category = "adult"
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w.Body).Encode(struct {
		Greeting string `json:"greeting"`
		Category string `json:"category"`
	}{
		fmt.Sprintf("Hello, %s %s!", req.First, req.Last),
		category,
	})
}

func WriteError(w http.ResponseWriter, msg string, status int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprint(w, msg)
}
