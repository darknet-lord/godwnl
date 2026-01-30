package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"sync"

	"github.com/darknet-lord/godwnl/internal/fetch"
)

var maxWorkers int
var dstFolder string

func init() {
	flag.StringVar(&dstFolder, "dest", "out", "Destination folder")
	flag.IntVar(&maxWorkers, "max-workers", 4, "Maximum parallel workers")
	flag.Parse()
}

func readUrls(urlCh chan string) {
	defer close(urlCh)
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" {
			urlCh <- line
		}
	}
}

func main() {
	fetcher := fetch.New(dstFolder)
	urlCh := make(chan string, maxWorkers)
	resCh := make(chan fetch.Result)

	go func() {
		readUrls(urlCh)
	}()

	var resWg sync.WaitGroup
	resWg.Add(1)
	go func() {
		defer resWg.Done()
		for res := range resCh {
			switch res.Ok {
			case true:
				fmt.Printf("Download completed: %s\n", res.Filename)
			case false:
				fmt.Printf("Download failed: %s\n", res.Filename)
			}
		}
	}()

	var fetchWg sync.WaitGroup
	for i := 0; i < maxWorkers; i++ {
		fetchWg.Add(1)
		go func() {
			defer fetchWg.Done()
			fetcher.Fetch(urlCh, resCh)
		}()
	}

	fetchWg.Wait()
	close(resCh)
	resWg.Wait()
	fmt.Println("Completed")
}
