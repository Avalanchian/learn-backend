package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/avalanchian/learn-backend/middleware/servermw"
)

func main() {
	port := flag.Int("port", 8080, "port to listen on")
	flag.Parse()

	var h http.HandlerFunc = func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/time":
			fmt.Fprintln(w, time.Now().Format(time.RFC3339))
		case "/panic":
			panic("Don't Panic.")
		default:
			http.NotFound(w, r)
		}
	}

	h = servermw.RecordResponse(h)
	h = servermw.Recovery(h)
	h = servermw.Log(h)
	h = servermw.Trace(h)

	server := http.Server{
		Addr:              fmt.Sprintf(":%d", *port),
		Handler:           h,
		ReadTimeout:       1 * time.Second,
		WriteTimeout:      1 * time.Second,
		ReadHeaderTimeout: 200 * time.Millisecond,
	}
	log.Printf("Listening on %s", server.Addr)
	if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}
