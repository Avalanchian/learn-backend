package main

import (
	"context"
	"net/http"
	"os"
	"time"
)

func main() {
	server := http.Server{
		Addr:         ":8080",
		Handler:      http.HandlerFunc(helloWorld),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	go server.ListenAndServe()

	req, _ := http.NewRequestWithContext(context.TODO(), "GET", "http://localhost:8080", nil)
	res, err := new(http.Client).Do(req)
	_ = err
	defer res.Body.Close()

	res.Write(os.Stdout)
}

func helloWorld(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Hello World!\r\n"))
}
