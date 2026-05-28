package main

import (
	"bytes"
	"encoding"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// Check interfaces are implemented at compile time
var _, _ fmt.Stringer = (*Request)(nil), (*Response)(nil)
var _, _ encoding.TextMarshaler = (*Request)(nil), (*Response)(nil)

type Header struct{ Key, Value string }

// -------------------------------- Request ----------------------------------------
type Request struct {
	Method  string
	Path    string
	Headers []Header
	Body    string
}

func NewRequest(method, path, host, body string) (*Request, error) {
	switch {
	case method == "":
		return nil, errors.New("missing required argument: method")
	case path == "":
		return nil, errors.New("missing required argument: path")
	case !strings.HasPrefix(path, "/"):
		return nil, errors.New("path must start with /")
	case host == "":
		return nil, errors.New("missing required argument: host")
	default:
		headers := make([]Header, 2)
		headers[0] = Header{"Host", host}
		if body != "" {
			headers = append(headers, Header{"Content-Length", fmt.Sprintf("%d", len(body))})
		}
		return &Request{
			Method:  method,
			Path:    path,
			Headers: headers,
			Body:    body,
		}, nil
	}
}

func ParseRequest(raw string) (req Request, err error) {
	lines := splitLines(raw)
	log.Println(lines)

	if len(lines) < 3 {
		return Request{}, fmt.Errorf("malformed request: should have at least 3 lines")
	}

	// handle the first line
	first := strings.Fields(lines[0])
	req.Method, req.Path = first[0], first[1]
	if !strings.HasPrefix(req.Path, "/") {
		return Request{}, fmt.Errorf("malformed request: path should start with /")
	}
	if !strings.Contains(first[2], "HTTP") {
		return Request{}, fmt.Errorf("malformed request: first line should contain HTTP version")
	}

	var foundHost bool
	var bodyStart int

	// handle the headers
	for i := 1; i < len(lines); i++ {
		if lines[i] == "" {
			bodyStart = i + 1
			break
		}

		key, val, ok := strings.Cut(lines[i], ": ")
		if !ok {
			return Request{}, fmt.Errorf("malformed request: header %q should be of the form 'key: value'", lines[i])
		}

		if key == "Host" {
			foundHost = true
		}
		key = AsTitle(key)

		req.Headers = append(req.Headers, Header{key, val})
	}

	// recombine the body using windows newlines (\r\n) skipping empty final line
	end := len(lines) - 1
	req.Body = strings.Join(lines[bodyStart:end], "\r\n")
	if !foundHost {
		return Request{}, fmt.Errorf("malformed request: missing Host header")
	}

	return
}

func (req *Request) WithHeader(key, value string) *Request {
	req.Headers = append(req.Headers, Header{AsTitle(key), value})
	return req
}

func (req *Request) WriteTo(w io.Writer) (n int64, err error) {
	// using a closure to reduce repetition
	printf := func(format string, args ...any) error {
		m, err := fmt.Fprintf(w, format, args...)
		n += int64(m)
		return err
	}

	// write the request line
	if err := printf("%s %s HTTP/1.1\r\n", req.Method, req.Path); err != nil {
		return n, err
	}

	// write the headers
	for _, h := range req.Headers {
		if err := printf("%s: %s\r\n", h.Key, h.Value); err != nil {
			return n, err
		}
	}

	// write empty line between headers and body
	printf("\r\n")

	// write body and terminate with a newline
	err = printf("%s\r\n", req.Body)
	return
}

func (req *Request) String() string {
	b := new(strings.Builder)
	req.WriteTo(b)
	return b.String()
}

func (req *Request) MarshalText() ([]byte, error) {
	b := new(bytes.Buffer)
	req.WriteTo(b)
	return b.Bytes(), nil
}

// ---------------------------------- Response ---------------------------------------
type Response struct {
	StatusCode int
	Headers    []Header
	Body       string
}

func NewResponse(status int, body string) (*Response, error) {
	switch {
	case status < 100 || status > 599:
		return nil, errors.New("invalid status code")
	default:
		if body == "" {
			body = http.StatusText(status)
		}
		headers := []Header{Header{"Content-Length", fmt.Sprintf("%d", len(body))}}

		return &Response{status, headers, body}, nil
	}
}

func ParseResponse(raw string) (res *Response, err error) {
	lines := splitLines(raw)
	log.Println(lines)

	// handle first line
	first := strings.SplitN(lines[0], " ", 3)
	if !strings.Contains(first[0], "HTTP") {
		return nil, fmt.Errorf("malformed response: first line should contain HTTP version")
	}

	res = new(Response)
	res.StatusCode, err = strconv.Atoi(first[1])
	if err != nil {
		return nil, fmt.Errorf("malformed response: expected status code to be an integer, got %q", first[1])
	}

	if first[2] == "" || http.StatusText(res.StatusCode) != first[2] {
		log.Printf("missing or incorrect status text for status code %d: expected %q but got %q", res.StatusCode, http.StatusText(res.StatusCode), first[2])
	}

	var bodyStart int

	// handle headers
	for i := 1; i < len(lines); i++ {
		log.Println(i, lines[i])
		if lines[i] == "" {
			bodyStart = i + 1
			break
		}

		key, val, ok := strings.Cut(lines[i], ": ")
		if !ok {
			return nil, fmt.Errorf("malformed response: header %q should be of form 'key: value'", lines[i])
		}
		key = AsTitle(key)
		res.Headers = append(res.Headers, Header{key, val})
	}

	// recombine body with windows newlines (\r\n)
	res.Body = strings.TrimSpace(strings.Join(lines[bodyStart:], "\r\n"))

	return
}

func (res *Response) WithHeader(key, value string) *Response {
	res.Headers = append(res.Headers, Header{AsTitle(key), value})
	return res
}

func (res *Response) WriteTo(w io.Writer) (n int64, err error) {
	// using a closure to reduce repetition
	printf := func(format string, args ...any) error {
		m, err := fmt.Fprintf(w, format, args...)
		n += int64(m)
		return err
	}

	// write the response line
	if err := printf("HTTP/1.1 %d %s\r\n", res.StatusCode, http.StatusText(res.StatusCode)); err != nil {
		return n, err
	}

	// write the headers
	for _, h := range res.Headers {
		if err := printf("%s: %s\r\n", h.Key, h.Value); err != nil {
			return n, err
		}
	}

	// write empty line between headers and body
	printf("\r\n")

	// write body and terminate with a newline
	err = printf("%s\r\n", res.Body)
	return
}

func (res *Response) String() string {
	b := new(strings.Builder)
	res.WriteTo(b)
	return b.String()
}

func (res *Response) MarshalText() ([]byte, error) {
	b := new(bytes.Buffer)
	res.WriteTo(b)
	return b.Bytes(), nil
}

// ------------------------------- Helper Functions ----------------------------------
func AsTitle(key string) string {
	if key == "" {
		panic("empty header key")
	}

	if isTitleCase(key) {
		return key
	}

	return newTitleCase(key)
}

func isTitleCase(key string) bool {
	for i := range key {
		if i == 0 || key[i-1] == '-' {
			if key[i] >= 'a' && key[i] <= 'z' {
				return false
			}
		} else if key[i] >= 'A' && key[i] <= 'Z' {
			return false
		}
	}
	return true
}

func newTitleCase(key string) string {
	var b strings.Builder
	b.Grow(len(key))

	for i := range key {
		if i == 0 || key[i-1] == '-' {
			b.WriteByte(upper(key[i]))
		} else {
			b.WriteByte(lower(key[i]))
		}
	}
	return b.String()
}

func upper(c byte) byte {
	if c >= 'a' && c <= 'z' {
		return c + 'A' - 'a'
	}
	return c
}

func lower(c byte) byte {
	if c >= 'A' && c <= 'Z' {
		return c - 'A' + 'a'
	}
	return c
}

// splitLines splits a string on the windows newline "\r\n" returning a slice of strings.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}

	var lines []string
	i := 0
	for {
		j := strings.Index(s[i:], "\r\n")
		if j == -1 {
			lines = append(lines, s[i:])
			return lines
		}
		lines = append(lines, s[i:i+j])
		i += j + 2 // skips \r\n
	}
}
