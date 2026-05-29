package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/Avalanchian/learn-backend/middleware/clientmw"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("target url required")
	}

	target := os.Args[1]
	client := &http.Client{Transport: clientMiddleware(), Timeout: 5 * time.Second}

	req, err := http.NewRequestWithContext(context.TODO(), "GET", target, nil)
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	io.Copy(os.Stdout, resp.Body)
}

func clientMiddleware() http.RoundTripper {
	var rt RoundTripFunc

	const wait, tries = 10 * time.Millisecond, 3

	rt = clientmw.RetryOn5xx(http.DefaultTransport, wait, tries)
	rt = clientmw.Log(rt)
	rt = clientmw.Trace(rt)
	return rt
}
