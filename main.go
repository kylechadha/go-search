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
// try out gb
// run with 10timeout

// write tests :)
// test with -race (specifically using the same slice of maps)
// Need to set up error handling story

// Set the maximum number of concurrent requests to be executed.
var maxReqs = 20

// result type definition.
type result struct {
	site  string
	found bool
	err   error
}

// use timehop log package?
// then can add a verbose flag
func main() {
	// Code used to profile the application.
	// cfg := profile.Config{
	// 	MemProfile: true,
	// 	CPUProfile: true,
	// }
	// p := profile.Start(&cfg)
	// defer p.Stop()

	// Record the start time of execution.
	start := time.Now()

	// Define flags for the input file and search term.
	urlsFile := flag.String("input", "urls.txt", "location of file containing URLs")
	term := flag.String("search", "", "search term")
	flag.Parse()

	// Read the input file.
	urls, err := readFile(*urlsFile)
	if err != nil {
		fmt.Printf("Error reading from urls file: %s\n", err)
		os.Exit(1)
	}

	// Pass the search term and slice of URLs to the search method,
	// with the first item removed (the column name).
	results := search(*term, urls[1:])

	// Write to the output file.
	err = writeFile(results)
	if err != nil {
		fmt.Printf("Error writing to results file: %s\n", err)
		os.Exit(1)
	}

	// Log the total execution time.
	log.Printf("Search took %s", time.Since(start))
}

// readFile takes the file path of a csv file containing
// URLs in the second column, and returns a slice of URLs.
func readFile(path string) ([]string, error) {

	// Open the file.
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Read the csv data.
	r := csv.NewReader(f)
	rawData, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	// Construct a slice of URLs.
	var urls []string
	for _, row := range rawData {
		urls = append(urls, row[1])
	}

	return urls, nil
}

// writeFile takes a slice of results and writes them
// to 'results.txt' in tab-separated columns.
func writeFile(results []result) error {

	// Create the file.
	f, err := os.Create("results.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	// Create a new tabwriter.Writer.
	w := new(tabwriter.Writer)

	// Specify the file and formatting (tab-separated columns with a tab stop of 4).
	w.Init(
		f,    // output
		0,    // minwidth
		4,    // tabwidth
		0,    // padding
		'\t', // padchar
		0,    // flags
	)

	// Range through the results and construct the fileContents.
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

	// Write the fileContents to the file.
	n, err := fmt.Fprintf(w, fileContents)
	if err != nil {
		return err
	}

	// Flush the writer.
	err = w.Flush()
	if err != nil {
		return err
	}

	// Log the number of bytes written.
	log.Printf("%d bytes written to results.txt", n)
	return nil
}

// search takes a search term and a slice of URLs, fetches the
// page content for each URL, performs a search, and then returns
// a slice of results containing the result and any errors encountered.
func search(term string, urls []string) []result {

	// If no search term was provided, exit.
	if term == "" {
		fmt.Println("No search term was provided. Expected arguments: '-search=searchTerm'.")
		os.Exit(1)
	} else {
		// Lowercase the search term so our comparisons will be case-insensitive.
		term = strings.ToLower(term)
	}

	// Create one chan of strings, on which we will send work to be processed (urls).
	// Create one chan of type result, on which we will return results.
	// Set up a WaitGroup so we can track when all goroutines have finished processing.
	ch := make(chan string)
	done := make(chan result)
	var wg sync.WaitGroup

	// Create a single http client with a 5 second timeout.
	// From the docs: "Clients should be reused instead of created as
	// needed. Clients are safe for concurrent use by multiple goroutines."
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// If there are less than 20 urls, decrease maxReqs to the number of
	// urls to avoid spinning up unnecessary goroutines.
	if maxReqs > len(urls) {
		maxReqs = len(urls)
	}

	// Spin up 'maxReqs' number of goroutines.
	wg.Add(maxReqs)
	for i := 0; i < maxReqs; i++ {
		go func() {
			for {
				// Recieve work from the chan of strings (urls).
				site, ok := <-ch
				if !ok {
					// If the channel is closed, there is no more work to be done and we can return.
					wg.Done()
					return
				}

				// Fetch the page content.
				response, err := client.Get("http://" + site)
				if err != nil {
					// If there are errors, try again with the 'www' host prefix.
					log.Println(err)
					log.Printf("Initial request failed for %s, attempting 'www' prefix.", site)

					response, err = client.Get("http://www." + site)
				}

				// If there are still errors, return the error message and 'continue' looping.
				if err != nil {
					log.Println(err)
					log.Printf("Both requests failed for %s, returning an error.", site)
					done <- result{site, false, err}
					continue
				}

				// Extract the page text from the response.
				// Note that FromReader uses html.Parse under the hood,
				// which reads to EOF in the same manner as ioutil.ReadAll.
				// https://github.com/jaytaylor/html2text/blob/master/html2text.go#L167
				text, err := html2text.FromReader(response.Body)
				response.Body.Close()
				if err != nil {
					log.Println(err)
					done <- result{site, false, err}
					continue
				}

				// Search for the search term in the page text and return the final result.
				found := strings.Contains(strings.ToLower(text), term)
				done <- result{site, found, nil}
			}
		}()
	}

	// Send work to be processed as goroutines become available.
	go func() {
		for _, site := range urls {
			log.Printf("sending url: %s", site)
			ch <- site
		}
	}()

	// Receive the results on the done chan.
	results := []result{}
	for i := 0; i < len(urls); i++ {
		select {
		case result := <-done:
			log.Printf("receiving result: %s", result.site)
			results = append(results, result)
		}
	}

	// Close the channel as a signal to the goroutines that no additional work needs to be processed.
	close(ch)
	wg.Wait()

	return results
}

// go tool pprof -pdf ./go-search /var/folders/nx/4vjf6d_53pdgwsz8s9_p_5nw0000gp/T/profile729622153/cpu.pprof > cpu_profile2.pdf
// go tool pprof -pdf ./go-search /var/folders/nx/4vjf6d_53pdgwsz8s9_p_5nw0000gp/T/profile729622153/mem.pprof > mem_profile2.pdf
