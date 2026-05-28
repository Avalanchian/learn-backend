package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"strings"
)

func echoUpper(w io.Writer, r io.Reader) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := scanner.Text()
		log.Printf("received: %s", line)
		fmt.Fprintf(w, "%s\n", strings.ToUpper(line))
	}
	if err := scanner.Err(); err != nil {
		log.Printf("error processing message: %v", err)
	}
}
