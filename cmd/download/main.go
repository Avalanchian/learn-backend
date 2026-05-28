// download is a command-line tool to download a file from a URL
// usage: download [-timeout duration] url filename
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

func main() {
	dir := flag.String("dir", ".", "directory to save file")
	timeout := flag.Duration("timeout", 30*time.Second, "timeout for download")
	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		log.Fatal("usage: download [-timeout duration] url filename")
	}
	url, filename := args[0], args[1]

	c := http.Client{Timeout: *timeout}

	if err := downloadAndSave(context.TODO(), &c, url, *dir, filename); err != nil {
		log.Fatal(err)
	}
}

func downloadAndSave(ctx context.Context, c *http.Client, url, dir, dst string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating request: GET %q: %w", url, err)
	}

	res, err := c.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("response status: %s", res.Status)
	}

	dstPath := filepath.Join(dir, dst)
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, res.Body); err != nil {
		return fmt.Errorf("error copying file: %w", err)
	}
	return nil
}
