package fetch

import (
	"errors"
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
	"time"
)

type Fetcher struct {
	DestinationFolder string
}

type Result struct {
	Ok       bool
	Filename string
}

func New(destination_folder string) *Fetcher {
	return &Fetcher{
		DestinationFolder: destination_folder,
	}
}

func (f Fetcher) Fetch(urlCh <-chan string, resCh chan<- Result) {
	for url := range urlCh {
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("Error while fetching %s\n", url)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {

		}

		filename := f.getDstFilename(url, resp.Header.Get("content-type"))
		dstPath := filepath.Join(f.DestinationFolder, filename)
		dst, err := os.Create(dstPath)
		if err != nil {
			log.Fatalf("Unable to create new file '%s': %s\n", dstPath, err)
		}
		_, err = io.Copy(dst, resp.Body)
		if err != nil {
			fmt.Printf("Unable to write data to '%s'\n", dstPath)
		}

		resCh <- Result{Ok: true, Filename: filename}
	}
}

func (f Fetcher) getDstFilename(url_ string, contentType string) string {
	u, err := url.Parse(url_)
	if err != nil {
		log.Fatalf("Unable to parse url: %s", url_)
	}

	if _, err := os.Stat(f.DestinationFolder); errors.Is(err, os.ErrNotExist) {
		if err = os.Mkdir(f.DestinationFolder, os.ModePerm); err != nil {
			log.Fatalf("Unable to create destination directory %s: %s\n", f.DestinationFolder, err)
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

	fp := filepath.Join(f.DestinationFolder, filename)

	if _, err := os.Stat(fp); errors.Is(err, os.ErrExist) {
		ext := filepath.Ext(filename)
		fn := filename[:len(filename)-len(filepath.Ext(filename))]
		return fn + strconv.Itoa(int(now.UnixNano())) + ext
	}

	return filename
}
