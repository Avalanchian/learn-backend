package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/avalanchian/learn-backend/middleware/servermw"
)

func main() {
	port := flag.Int("port", 8080, "port to listen on")
	flag.Parse()

	h := BuildRouter()

	server := http.Server{
		Addr:         fmt.Sprintf(":%d", *port),
		Handler:      h,
		ReadTimeout:  1 * time.Second,
		WriteTimeout: 1 * time.Second,
	}

	log.Printf("listening on %s", server.Addr)
	server.ListenAndServe()
}

func applyMiddleware(h http.HandlerFunc) http.HandlerFunc {
	h = servermw.RecordResponse(h)
	h = servermw.Recovery(h)
	h = servermw.Log(h)
	h = servermw.Trace(h)

	return h
}

func BuildRouter() *http.ServeMux {
	router := http.NewServeMux()
	router.HandleFunc("GET /{$}", applyMiddleware(indexHandler))
	router.HandleFunc("GET /panic", applyMiddleware(panicHandler))
	router.HandleFunc("POST /greet/json", applyMiddleware(greetHandler))
	router.HandleFunc("GET /time", applyMiddleware(timeHandler))
	router.HandleFunc("GET /echo/{a}/{b}/{c}", applyMiddleware(echoHandler))

	return router
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

	_ = WriteJSON(w, struct {
		Greeting string `json:"greeting"`
		Category string `json:"category"`
	}{
		fmt.Sprintf("Hello, %s %s!", req.First, req.Last),
		category,
	})
}

func timeHandler(w http.ResponseWriter, r *http.Request) {
	format := r.URL.Query().Get("format")
	if format == "" {
		format = time.RFC3339
	}

	tz := r.URL.Query().Get("tz")
	var loc *time.Location = time.Local

	if tz != "" {
		var err error
		loc, err = time.LoadLocation(tz)
		if err != nil {
			WriteError(w, fmt.Errorf("invalid timezone %q: %w", tz, err), http.StatusBadRequest)
			return
		}
	}

	_ = WriteJSON(w, struct {
		Time string `json:"time"`
	}{
		time.Now().In(loc).Format(format),
	})
}

func echoHandler(w http.ResponseWriter, r *http.Request) {
	pathVars := map[string]string{
		"a": r.PathValue("a"),
		"b": r.PathValue("b"),
		"c": r.PathValue("c"),
	}

	switch strings.ToLower(r.URL.Query().Get("case")) {
	case "lower":
		for k, v := range pathVars {
			pathVars[k] = strings.ToLower(v)
		}
	case "upper":
		for k, v := range pathVars {
			pathVars[k] = strings.ToUpper(v)
		}
	}

	_ = WriteJSON(w, pathVars)
}

func WriteJSON[T any](w http.ResponseWriter, out T) error {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(out)
	return err
}

func WriteError(w http.ResponseWriter, msg error, status int) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	fmt.Fprint(w, msg)
}
