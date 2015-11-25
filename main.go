package main

import (
	"encoding/csv"
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

type result struct {
	site  string
	found bool
	err   error
}

func main() {
	start := time.Now()

	urlsFile := flag.String("input", "urls.txt", "location of urls.txt")
	term := flag.String("search", "", "search term")
	flag.Parse()

	urls := readUrls(*urlsFile)
	results := search(*term, urls[1:]) // strip column name from slice

	for site, found := range results {
		// is this the right Print to use here?
		fmt.Printf("%s: %t\n", site, found)
	}

	log.Printf("search took %s", time.Since(start))
}

func readUrls(file string) []string {

	csvfile, err := os.Open(file)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	defer csvfile.Close()

	r := csv.NewReader(csvfile)

	rawData, err := r.ReadAll()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var urls []string
	for _, row := range rawData {
		urls = append(urls, row[1])
	}

	return urls
}

// write tests :)
// test with -race (specifically using the same slice of maps)
func search(term string, urls []string) map[string]bool {

	if term == "" {
		fmt.Println("No search term was provided. Expected arguments: '-search=searchTerm'.")
		os.Exit(1)
	}
	// Need to set up error handling story

	// If there are less than 20 urls, decrease maxReqs to the number of urls.
	// This way we don't spin up unnecessary goroutines.
	if maxReqs > len(urls) {
		log.Printf("changing maxReqs to: %d", len(urls))
		maxReqs = len(urls)
	}

	ch := make(chan string)
	done := make(chan result)
	var wg sync.WaitGroup

	// what happens if the number of urls is less than maxReqs?
	// it doesn't block because the channel closes when the range urls is done
	wg.Add(maxReqs)
	for i := 0; i < maxReqs; i++ {
		go func(i int) {
			log.Println("Goroutine #:", i)
			for {
				site, ok := <-ch
				if !ok {
					wg.Done()
					log.Println("Goroutine #:", i, " done!")
					return
				}

				// u, err := url.Parse("http://bing.com/search?q=dotnet")
				// if err != nil {
				// 	fmt.Printf("%s", err)
				// 	os.Exit(1)
				// }

				timeout := time.Duration(5 * time.Second)
				client := http.Client{
					Timeout: timeout,
				}
				response, err := client.Get("http://" + site)
				// response, err := http.Get("http://" + site)
				if err != nil {
					fmt.Printf("%s", err)
					log.Println("SWITCHING TO WWW!!!!")
					response, err = client.Get("http://www." + site)
					// response, err = http.Get("http://www." + site)
				}

				if err != nil {
					log.Println("STILL ERROR WTF!")
					done <- result{site, false, err}
					break
				}

				// *else? look at some other examples
				defer response.Body.Close()
				text, err := html2text.FromReader(response.Body)
				if err != nil {
					fmt.Printf("%s", err)
					os.Exit(1)
				}
				// fmt.Printf("%s\n\n\n", text)

				text, term = strings.ToLower(text), strings.ToLower(term)
				found := strings.Contains(text, term)

				done <- result{site, found, nil}
			}
		}(i)
	}

	// Prevents us from having to use a buffer if maxReqs is less than the number of total urls.
	// Avoiding buffers is always a good practice -- you should understand why your goroutines can't accept work,
	// and if you use a buffer, you still need to implement a solution for backpressure when you exceed the buffer size.
	go func() {
		for _, site := range urls {
			log.Printf("sending url: %s", site)
			ch <- site
		}
	}()

	results := make(map[string]bool)
	// add a timeout here
	for i := 0; i < len(urls); i++ {
		select {
		case result := <-done:
			// is the append implementation better? there's a commit for it, check if you need to revert
			log.Println("receiving result")
			log.Printf("%+v", result)
			results[result.site] = result.found
		}
	}

	log.Println("closing channel")
	close(ch)
	// close(done)
	wg.Wait()

	return results
}

// is this going to fetch AND search or just fetch? if AND search, then it needs the term .. if just search, than you have to return the respond.Body? dicey
// func fetch(url string, c chan result) {

// }
