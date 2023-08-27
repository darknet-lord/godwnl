package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var urlsFile string
var maxWorkers int
var dstFolder string

type Result struct {
	ok       bool
	filename string
}

func init() {
	flag.StringVar(&urlsFile, "urls-file", "urls.txt", "List with urls to download")
	flag.StringVar(&dstFolder, "dest", "out", "Destination folder")
	flag.IntVar(&maxWorkers, "max-workers", 4, "Maximum parallel workers")
	flag.Parse()
}

func readFile(urlCh chan string) {
	defer close(urlCh)
	f, err := os.Open(urlsFile)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		urlCh <- scanner.Text()
	}
}

func getDstFilename(url_ string, contentType string) string {
	u, err := url.Parse(url_)
	if err != nil {
		log.Fatalf("Unable to parse url: %s", url_)
	}

	if _, err := os.Stat(dstFolder); errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(dstFolder, os.ModePerm); err != nil {
			log.Fatalf("Unable to create destination directory %s: %s\n", dstFolder, err)
		}
	}

	now := time.Now()
	if u.Path == "" {
		exts, err := mime.ExtensionsByType(contentType)
		genName := strconv.Itoa(int(now.UnixNano()))
		if err == nil {
			genName += exts[0]
		}
		return genName
	}
	parts := strings.Split(u.Path, "/")
	filename := parts[len(parts)-1]

	fp := filepath.Join(dstFolder, filename)

	if _, err := os.Stat(fp); errors.Is(err, os.ErrExist) {
		ext := filepath.Ext(filename)
		fn := filename[:len(filename)-len(filepath.Ext(filename))]
		return fn + strconv.Itoa(int(now.UnixNano())) + ext
	}

	return filename
}

func fetch(urlCh <-chan string, resCh chan<- Result) {
	for url := range urlCh {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error while fetching %s\n", url)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {

		}

		filename := getDstFilename(url, resp.Header.Get("content-type"))
		dstPath := filepath.Join(dstFolder, filename)
		dst, err := os.Create(dstPath)
		if err != nil {
			log.Fatalf("Unable to create new file '%s': %s\n", dstPath, err)
		}
		_, err = io.Copy(dst, resp.Body)
		if err != nil {
			fmt.Printf("Unable to write data to '%s'\n", dstPath)
		}

		resCh <- Result{ok: true, filename: filename}
	}
}

func main() {
	urlCh := make(chan string, maxWorkers)
	resCh := make(chan Result)

	if _, err := os.Stat(urlsFile); errors.Is(err, os.ErrNotExist) {
		log.Fatalf("File %s not found", urlsFile)
	}

	go func() {
		readFile(urlCh)
	}()

	var resWg sync.WaitGroup
	resWg.Add(1)
	go func() {
		defer resWg.Done()
		for res := range resCh {
			switch res.ok {
			case true:
				fmt.Printf("Download completed: %s\n", res.filename)
			case false:
				fmt.Printf("Download failed: %s\n", res.filename)
			}
		}
	}()

	var fetchWg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		fetchWg.Add(1)
		go func() {
			defer fetchWg.Done()
			fetch(urlCh, resCh)
		}()
	}

	fetchWg.Wait()
	close(resCh)
	resWg.Wait()
	fmt.Println("Completed")
}
