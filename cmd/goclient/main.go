package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func NewGetRequest() (*http.Request, error) {
	ctx := context.TODO()

	var body io.Reader
	const method = "GET"
	const url = "https://eblog.fly.dev/index.html"
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	return req, err
}

func NewPostRequest() (*http.Request, error) {
	ctx := context.TODO()

	var body io.Reader = strings.NewReader("Hello World")
	const method = "POST"
	const url = "https://eblog.fly.dev/index.html"
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	return req, err
}

func main() {
	const method = "GET"
	const path = "https://scryfall.com/search"
	v := make(url.Values)

	v.Add("q", `"of Emrakul"`)
	v.Add("order", "released")
	v.Add("dir", "asc")

	dst := path + "?" + v.Encode()

	req, err := http.NewRequestWithContext(context.TODO(), method, dst, nil)
	if err != nil {
		log.Fatalf("error creating new request: %v", err)
	}

	req.Write(os.Stdout)
}
