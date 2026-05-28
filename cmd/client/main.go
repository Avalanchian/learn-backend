package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
)

const name string = "writetcp"

func main() {
	log.SetPrefix(name + "\t")

	// Parse selected port (default is 8080)
	port := flag.Int("p", 8080, "port to connect to")
	flag.Parse()

	// Connect to localhost:port
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{Port: *port})
	if err != nil {
		log.Fatalf("error connecting to localhost:%d: %v", *port, err)
	}

	log.Printf("connected to %s: will forward stdin", conn.RemoteAddr())
	defer conn.Close()

	// Spawn goroutine to read responses
	go func() {
		for connScanner := bufio.NewScanner(conn); connScanner.Scan(); {
			fmt.Printf("%s\n", connScanner.Text())

			if err := connScanner.Err(); err != nil {
				log.Fatalf("error reading from %s: %v", conn.RemoteAddr(), err)
			}
		}
	}()

	// Read lines from stdin and send them via TCP connection
	for stdinScanner := bufio.NewScanner(os.Stdin); stdinScanner.Scan(); {
		log.Printf("sent: %s\n", stdinScanner.Text())
		if _, err := conn.Write(stdinScanner.Bytes()); err != nil {
			log.Fatalf("error writing to %s: %v", conn.RemoteAddr(), err)
		}

		if _, err := conn.Write([]byte("\n")); err != nil {
			log.Fatalf("error writing to %s: %v", conn.RemoteAddr(), err)
		}

		if err := stdinScanner.Err(); err != nil {
			log.Fatalf("error reading from stdin: %v", err)
		}
	}
}
