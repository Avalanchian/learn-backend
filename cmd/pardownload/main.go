// pardownload is a command-line tool to download files in parallel from a URL
// usage: pardownload [-timeout duration] url filename...
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
	"sync"
	"time"
)

func main() {
	var dstDir string
	var client http.Client

	flag.StringVar(&dstDir, "dst", ".", "directory to save files")
	flag.DurationVar(&client.Timeout, "timeout", 60*time.Second, "timeout for download")
	flag.Parse()

	src := flag.Args()
	if len(src) == 0 {
		log.Fatal("usage: download [-timeout duration] filename...")
	}

	dstDir, err := filepath.Abs(dstDir)
	if err != nil {
		log.Fatalf("invalid destination directory: %v", err)
	}

	dst := make([]string, len(src))
	for i := range src {
		dst[i] = filepath.Join(dstDir, filepath.Base(src[i]))
	}

	errs := make([]error, len(src))

	wg := new(sync.WaitGroup)
	wg.Add(len(src))

	now := time.Now()
	for i := range src {
		go func() {
			defer wg.Done()
			errs[i] = downloadAndSave(context.TODO(), &client, src[i], dst[i])
		}()
	}
	wg.Wait()

	log.Printf("downloaded %d files in %v", len(src), time.Since(now))
	var errorCount int

	for i := range errs {
		if errs[i] != nil {
			log.Printf("err: %s -> %s: %v", src[i], dst[i], errs[i])
			errorCount++
		} else {
			log.Printf("ok: %s -> %s", src[i], dst[i])
		}
	}
	os.Exit(errorCount)
}

func downloadAndSave(ctx context.Context, c *http.Client, url, dst string) error {
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

	dstFile, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, res.Body); err != nil {
		return fmt.Errorf("error copying file: %w", err)
	}
	return nil
}
