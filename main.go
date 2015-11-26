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
	"text/tabwriter"
	"time"

	"github.com/jaytaylor/html2text"
)

// For everything
// check variable names
// check comments
// check logs
// check all errors are handled
// add readme

// write tests :)
// test with -race (specifically using the same slice of maps)
// Need to set up error handling story

// Define the maximum number of concurrent requests to be executed.
var maxReqs = 20

// 'result' type definition.
type result struct {
	site  string
	found bool
	err   error
}

// use timehop log package?
// then can add a verbose flag
func main() {

	// Record the start time of execution.
	start := time.Now()

	// Define flags for the input file and search term.
	urlsFile := flag.String("input", "urls.txt", "location of file containing urls")
	term := flag.String("search", "", "search term")
	flag.Parse()

	// Read the input file and return a slice of urls.
	urls, err := readFile(*urlsFile)
	if err != nil {
		// ** try a format here that you can add a message too
		fmt.Println(err)
		os.Exit(1)
	}

	// Pass the search term and slice of urls to the search method.
	// We remove the first item from the slice, as it represents the column name.
	results := search(*term, urls[1:])

	// Pass the results to the writeFile method.
	err = writeFile(results)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Log the total execution time of the application.
	log.Printf("Search took %s", time.Since(start))
}

// readFile takes the file path of a csv file containing URLs in the second column,
// and returns a slice of URLs.
func readFile(path string) ([]string, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rawData, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var urls []string
	for _, row := range rawData {
		urls = append(urls, row[1])
	}

	return urls, nil
}

func writeFile(results []result) error {

	f, err := os.Create("results.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	w := new(tabwriter.Writer)

	// Format in tab-separated columns with a tab stop of 4 (the default of most text editors).
	w.Init(f, 0, 4, 0, '\t', 0)

	fileContents := "Site\tFound\tError\t\n"
	for _, result := range results {
		if result.err != nil {
			log.Printf("site:%s found:%t err:%s\n", result.site, result.found, result.err.Error())
			fileContents += fmt.Sprintf("%s\t%t\t%s\n", result.site, result.found, result.err.Error())
		} else {
			log.Printf("site:%s found:%t err:%v\n", result.site, result.found, result.err)
			fileContents += fmt.Sprintf("%s\t%t\t%v\n", result.site, result.found, result.err)
		}
	}

	n, err := fmt.Fprintf(w, fileContents)
	if err != nil {
		return err
	}

	w.Flush()

	log.Printf("%d bytes written to results.txt", n)

	return nil
}

func search(term string, urls []string) []result {

	// If no search term was provided, exit.
	if term == "" {
		// fmt.Errorf here? when is that used?
		fmt.Println("No search term was provided. Expected arguments: '-search=searchTerm'.")
		os.Exit(1)
	} else {
		// Lowercase the search term.
		term = strings.ToLower(term)
	}

	// If there are less than 20 urls, decrease maxReqs to the number of urls
	// to avoid spinning up unnecessary goroutines.
	if maxReqs > len(urls) {
		log.Printf("Changing maxReqs to: %d", len(urls))
		maxReqs = len(urls)
	}

	// Create one chan of strings, on which we will send work to be processed (urls).
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

				response, err := client.Get("http://" + site)
				if err != nil {
					// If there errors, we'll try again with the 'www' prefix.
					log.Printf("%s\n", err)
					log.Printf("Initial request failed for %s, attempting 'www' prefix.", site)

					response, err = client.Get("http://www." + site)
				}

				if err != nil {
					log.Printf("Both requests failed for %s, returning an error.", site)
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

				text, err := html2text.FromString(string(contents))
				// text, err := html2text.FromReader(response.Body)
				if err != nil {
					log.Printf("%s", err)
					done <- result{site, false, err}
					continue
				}

				found := strings.Contains(strings.ToLower(text), term)
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

	results := []result{}
	for i := 0; i < len(urls); i++ {
		select {
		case result := <-done:
			log.Printf("receiving result: %s", result.site)
			results = append(results, result)
		}
	}

	// Close the channel as a signal to the goroutines that no additional work will be processed.
	close(ch)
	wg.Wait()

	return results
}
