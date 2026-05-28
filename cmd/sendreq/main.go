package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"
)

var (
	host, path, method string
	port               int
)

func main() {
	// define and parse flags
	flag.StringVar(&method, "method", "GET", "HTTP method to use")
	flag.StringVar(&host, "host", "localhost", "host to connect to")
	flag.StringVar(&path, "path", "/", "path to request")
	flag.IntVar(&port, "port", 8080, "port to connect to")
	flag.Parse()

	// Resolve the ip address of the host:port
	ip, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		log.Fatalf("could not resolve TCP address: %s:%d: %v", host, port, err)
	}

	// Create a TCP connection using the ip that was resolved
	conn, err := net.DialTCP("tcp", nil, ip)
	if err != nil {
		log.Fatalf("could not create tcp connection to %s: %v", ip, err)
	}

	log.Printf("connected to %s (@ %s)", host, conn.RemoteAddr())
	defer conn.Close()

	// Write HTTP lines
	var reqFields = []string{
		fmt.Sprintf("%s %s HTTP/1.1", method, path),
		"Host: " + host,
		"User-Agent: httpget",
		"",
	}

	request := strings.Join(reqFields, "\r\n") + "\r\n" // Windows line endings

	conn.Write([]byte(request))
	log.Printf("sent request:\n%s", request)

	for scanner := bufio.NewScanner(conn); scanner.Scan(); {
		line := scanner.Bytes()
		if _, err := fmt.Fprintf(os.Stdout, "%s\n", line); err != nil {
			log.Printf("error writing to connection: %v", err)
		}

		if err := scanner.Err(); err != nil {
			log.Printf("error reading from connection: %v", err)
			return
		}
	}
}
