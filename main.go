package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/jaytaylor/html2text"
)

// Define the maximum number of concurrent requests to be executed.
var maxReqs = 20

// 'result' type definition.
type result struct {
	site  string
	found bool
	err   error
}

func main() {
	// Record that start time of execution.
	start := time.Now()

	// Define and parse flags for input and output files and the search term.
	urlsFile := flag.String("input", "urls.txt", "location of urls.txt")
	term := flag.String("search", "", "search term")
	flag.Parse()

	// Read the urls file and return a slice of urls.
	urls := readUrls(*urlsFile)

	// Pass the search term as well as the list of urls to the 'search' method.
	// We remove the first item from the slice, as it represents the column name.
	results := search(*term, urls[1:])

	for _, result := range results {
		if result.err != nil {
			log.Printf("site:%s found:%t err:%s\n", result.site, result.found, result.err.Error())
		} else {
			log.Printf("site:%s found:%t err:%v\n", result.site, result.found, result.err)
		}
	}

	// Log the total time taken by the search.
	log.Printf("Search took %s", time.Since(start))
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
// Need to set up error handling story
func search(term string, urls []string) map[string]result {

	if term == "" {
		fmt.Println("No search term was provided. Expected arguments: '-search=searchTerm'.")
		os.Exit(1)
	} else {
		term = strings.ToLower(term)
	}

	// If there are less than 20 urls, decrease maxReqs to the number of urls.
	// This way we don't spin up unnecessary goroutines.
	if maxReqs > len(urls) {
		log.Printf("changing maxReqs to: %d", len(urls))
		maxReqs = len(urls)
	}

	// Create one chan of strings, on which we will pass urls (as work).
	// Create one chan of type result, on which we will return results.
	// Set up a WaitGroup so we can track when all goroutines have finished processing.
	ch := make(chan string)
	done := make(chan result)
	var wg sync.WaitGroup

	wg.Add(maxReqs)
	for i := 0; i < maxReqs; i++ {
		go func() {
			// ** Should this be a GLOBAL shared by ALL goroutines??
			// https: //code.google.com/p/go/issues/detail?id=4049#c3
			client := &http.Client{
				Timeout: 10 * time.Second,
			}

			for {
				site, ok := <-ch
				if !ok {
					wg.Done()
					return
				}

				// u, err := url.Parse("http://bing.com/search?q=dotnet")
				// if err != nil {
				// 	log.Printf("%s", err)
				// 	os.Exit(1)
				// }

				response, err := client.Get("http://" + site)
				// response, err := http.Get("http://" + site)
				if err != nil {
					// perhaps you should check here that the err was specifically 'no such host'
					// can also be Client.Timeout exceeded while awaiting headers or well, any timeout
					log.Printf("%s\n", err)
					log.Printf("Initial request failed for %s, attempting 'www' prefix.", site)

					// timeout := time.Duration(30 * time.Second)
					// client := http.Client{
					// 	Timeout: timeout,
					// }
					response, err = client.Get("http://www." + site)
					// response, err = http.Get("http://www." + site)
				}

				if err != nil {
					log.Printf("Both requests failed for %s, returning an error.", site)
					// can you pass nil for bool instead of false?
					done <- result{site, false, err}
					continue
				}

				// why *else? look at some other examples
				defer response.Body.Close()

				// OR ... do this?? and then feed that into FromReader
				// res, _ := client.Do(req)
				// io.Copy(ioutil.Discard, res.Body)
				// res.Body.Close()

				contents, err := ioutil.ReadAll(response.Body)
				if err != nil {
					fmt.Printf("%s", err)
					// os.Exit(1)
				}

				// two things
				// figure out whether you're reading all properly (find an article on understanding http client / transport all this jazz)
				// and figure out whether you can use httpclient or roll your own to avoid the wrong types of timeouts

				// since we're reusing clients, will want to make sure we readall... may need to implement previous way back
				text, err := html2text.FromString(string(contents))
				// text, err := html2text.FromReader(response.Body)
				if err != nil {
					log.Printf("%s", err)
					done <- result{site, false, err}
					continue
				}

				text = strings.ToLower(text)
				found := strings.Contains(text, term)
				done <- result{site, found, nil}
			}
		}()
	}

	// Prevents us from having to use a buffer if maxReqs is less than the number of total urls.
	// Avoiding buffers is always a good practice -- you should understand why your goroutines can't accept work,
	// and if you use a buffer, you still need to implement a solution for backpressure when you exceed the buffer size.
	go func() {
		for _, site := range urls {
			log.Printf("sending url: %s", site)
			ch <- site
			// could print INDEX here to see how many have been sent ... where is the blockage? is one goroutine being busy blocking all other goroutines from accepting work?
		}
	}()

	results := make(map[string]result)
	// add a timeout here
	for i := 0; i < len(urls); i++ {
		select {
		case result := <-done:
			log.Printf("receiving result: %s", result.site)
			results[result.site] = result
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
