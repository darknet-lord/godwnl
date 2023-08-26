package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
)

var urlsFile string
var maxWorkers int

func init() {
	flag.StringVar(&urlsFile, "urls-file", "urls.txt", "List with urls to download")
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

func fetch(urlCh <-chan string, resCh chan<- string) {
	for url := range urlCh {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error while fetching %s\n", url)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			fmt.Printf("Unable to read data for %s\n", url)
		}
		resCh <- string(data)
	}
}

func main() {
	urlCh := make(chan string, maxWorkers)
	resCh := make(chan string)

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
			fmt.Println(len(res))
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
