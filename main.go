package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"text/tabwriter"
	"time"

	"github.com/jaytaylor/html2text"
	"github.com/timehop/golog/log"
)

// Set the maximum number of concurrent requests to be executed.
var maxReqs = 20

// result type definition.
type result struct {
	site  string
	found bool
	err   error
}

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

	// Define flags for the input file, search term, and log level.
	term := flag.String("search", "", "required: please provide a search term")
	path := flag.String("input", "urls.txt", "enter the location of the file containing URLs")
	verbose := flag.Bool("verbose", false, "verbose logging option")
	flag.Parse()

	// Set the log level based on the -verbose flag.
	if *verbose == true {
		log.SetLevel(4)
	}

	// Read the input file.
	urls, err := readFile(*path)
	if err != nil {
		log.Fatal("go-search", "Error reading from urls file", "error", err)
	}

	// Pass the search term and slice of URLs to the search method.
	// Note: Remove the first item of the urls slice (the column name).
	results := search(*term, urls[1:])

	// Write to the output file.
	err = writeFile(results)
	if err != nil {
		log.Fatal("go-search", "Error writing to results file", "error", err)
	}

	// Log the total execution time.
	log.Info("go-search", fmt.Sprintf("Search took %s", time.Since(start)))
}

// readFile takes the file path of a csv file containing
// URLs in the second column, and returns a slice of URLs.
func readFile(path string) ([]string, error) {

	log.Info("go-search", "Reading from the input file")

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

	log.Info("go-search", "Writing to the output file")

	// Create the file.
	f, err := os.Create("results.txt")
	if err != nil {
		return err
	}
	defer f.Close()

	// Create a new tabwriter.Writer and specify the output and
	// formatting (tab-separated columns with a tab stop of 4).
	// FYI: This will look nice in a text editor, but not notepad.
	w := new(tabwriter.Writer)
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
			fileContents += fmt.Sprintf("%s\t%s\t%s\n", result.site, "", result.err.Error())
		} else {
			fileContents += fmt.Sprintf("%s\t%t\t%v\n", result.site, result.found, "")
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
	log.Info("go-search", fmt.Sprintf("%d bytes written to results.txt", n))

	return nil
}

// search takes a search term and a slice of URLs, fetches the
// page content for each URL, performs a search, and then returns
// a slice of results containing the result and any errors encountered.
func search(term string, urls []string) []result {

	// If no search term was provided, exit.
	if term == "" {
		log.Fatal("go-search", "No search term was provided. Expected arguments: '-search=searchTerm'.")
	} else {
		// Lowercase the search term so our comparisons will be case-insensitive.
		term = strings.ToLower(term)
	}

	// Create a chan of strings to send work to be processed (urls).
	// Create a chan of type result to send results.
	// Set up a WaitGroup so we can track when all goroutines have finished processing.
	ch := make(chan string)
	done := make(chan result)
	var wg sync.WaitGroup

	// Create a single http Client with an 8 second timeout.
	// From the docs: "Clients should be reused instead of created as
	// needed. Clients are safe for concurrent use by multiple goroutines."
	client := &http.Client{
		Timeout: 8 * time.Second,
	}

	// If there are less than 20 urls in urls.txt, decrease maxReqs to
	// the number of urls to avoid spinning up unnecessary goroutines.
	if maxReqs > len(urls) {
		maxReqs = len(urls)
	}

	log.Info("go-search", "Fetching and searching urls...")
	log.Info("go-search", "Go ahead, queue up your favorite jam: this will take ~30 seconds")

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

				// Provide some visual feedback to the user for each url processed.
				v := flag.Lookup("verbose")
				if v.Value.String() == "false" {
					fmt.Print(".")
				}

				// Fetch the page content.
				response, err := client.Get("http://" + site)
				if err != nil {
					// If there are errors, try again with the 'www' host prefix.
					log.Debug("go-search", fmt.Sprintf("Initial request failed for %s, attempting 'www' prefix.", site), "error", err)

					response, err = client.Get("http://www." + site)
				}

				// If there are still errors, return the error message and continue looping.
				if err != nil {
					log.Debug("go-search", fmt.Sprintf("Both requests failed for %s, returning an error.", site), "error", err)

					done <- result{site, false, err}
					continue
				}

				// Extract the human-readable text from the response.
				// Note that FromReader uses html.Parse under the hood,
				// which reads to EOF in the same manner as ioutil.ReadAll.
				// https://github.com/jaytaylor/html2text/blob/master/html2text.go#L167
				text, err := html2text.FromReader(response.Body)
				response.Body.Close()
				if err != nil {
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
			log.Debug("go-search", fmt.Sprintf("Sending work: %s", site))
			ch <- site
		}
	}()

	// Receive the results on the done chan.
	results := []result{}
	for i := 0; i < len(urls); i++ {
		select {
		case result := <-done:
			log.Debug("go-search", fmt.Sprintf("Receiving result: %s", result.site))
			results = append(results, result)
		}
	}

	// Close the channel as a signal to the goroutines that no additional work needs to be processed.
	close(ch)

	// Wait for the goroutines to be done processing.
	wg.Wait()

	fmt.Print("Done!\n")
	return results
}
