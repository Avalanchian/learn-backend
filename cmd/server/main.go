package main

import (
	"flag"
	"log"
	"net"
)

const name string = "tcpupperecho"

func main() {
	log.SetPrefix(name + "\t")

	// Parse flags to set port (default 8080)
	port := flag.Int("p", 8080, "port to listen to")
	flag.Parse()

	// Listen on the user-specified port
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{Port: *port})
	if err != nil {
		log.Fatalf("error listening to localhost:%d: %v", *port, err)
	}
	defer listener.Close()

	log.Printf("listening to localhost: %s", listener.Addr())

	// Loop forever, listening and handling messages as they come.
	for {
		conn, err := listener.Accept() // Accept blocks until a connection is made
		if err != nil {
			log.Fatalf("error connecting on %s: %v", listener.Addr(), err)
		}

		go echoUpper(conn, conn) // Spawn goroutine to run business logic.
	}
}
