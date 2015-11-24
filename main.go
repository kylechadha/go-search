package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jaytaylor/html2text"
)

var maxReqs = 20
var urls = []string{"https://godoc.org/fmt#Sprintf", "https://getgb.io/", "https://golang.org/pkg/runtime/pprof/"}

type result struct {
	url   string
	found bool
}

func main() {
	// Let's do a bit of back-of-the-envelope profiling.
	start := time.Now()

	term := flag.String("search", "", "search term")
	flag.Parse()

	results := search(*term)

	for url, found := range results {
		// is this the right Print to use here?
		fmt.Printf("%s: %t\n", url, found)
	}

	elapsed := time.Since(start)
	log.Printf("search took %s", elapsed)
}

// write tests :)
// test with -race (specifically using the same slice of maps)
func search(term string) map[string]bool {

	if term == "" {
		fmt.Println("No search term was provided. Expected arguments: '-search=searchTerm'.")
		os.Exit(1)
	}
	// Need to set up error handling story

	// *** pretty sure we'll just use the bottom for loop to add to the queue and then it all works gravy
	// maybe you can remove the urls in a separate range, and send them on a channel
	// and then the goroutines pull them off as they can do them
	// will have to figure this part out

	// does this need to be buffered?
	// the buffered number needs to be greater than maxReqs though, apparently (why?)
	// chu := make(chan string, 50)
	chu := make(chan string)
	chr := make(chan result)
	var wg sync.WaitGroup

	// If there are less than 20 urls, decrease maxReqs to the number of urls.
	// This way we don't spin up unnecessary goroutines.
	if maxReqs > len(urls) {
		maxReqs = len(urls)
	}

	// what happens if the number of urls is less than maxReqs?
	// it doesn't block because the channel closes when the range urls is done
	wg.Add(maxReqs)
	for i := 0; i < maxReqs; i++ {
		go func() {
			for {
				url, ok := <-chu
				if !ok {
					log.Println("Channel closed I believeso")
					wg.Done()
					return
				}

				response, err := http.Get(url)
				if err != nil {
					fmt.Printf("%s", err)
					os.Exit(1)
				} else {
					defer response.Body.Close()
					text, err := html2text.FromReader(response.Body)
					if err != nil {
						fmt.Printf("%s", err)
						os.Exit(1)
					}
					// fmt.Printf("%s\n\n\n", text)

					text, term = strings.ToLower(text), strings.ToLower(term)
					found := strings.Contains(text, term)

					chr <- result{url, found}
				}
			}
		}()
	}

	for _, url := range urls {
		log.Printf("sending url: %s", url)
		chu <- url
	}

	results := make(map[string]bool)
	// add a timeout here
	for i := 0; i < len(urls); i++ {
		select {
		case result := <-chr:
			// is the append implementation better? there's a commit for it, check if you need to revert
			log.Println("receiving result")
			results[result.url] = result.found
		}
	}

	log.Println("closing channel")
	close(chu)
	wg.Wait()

	return results
}

// is this going to fetch AND search or just fetch? if AND search, then it needs the term .. if just search, than you have to return the respond.Body? dicey
// func fetch(url string, c chan result) {

// }
