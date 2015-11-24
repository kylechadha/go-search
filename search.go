package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jaytaylor/html2text"
)

// use flags for this and the search term
var urls = []string{"https://godoc.org/fmt#Sprintf", "https://getgb.io/", "https://golang.org/pkg/runtime/pprof/"}

// const search string

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello")
}

// curl -XPOST -H 'Content-Type: application/json' -d '{"query,": "feature"}' http://localhost:3000/
func searchHandler(w http.ResponseWriter, r *http.Request) {
	// Let's do a bit of back-of-the-envelope profiling.
	start := time.Now()

	// Need to set up error handling story
	// Create an app handler
	// Or just write out an error message

	// Also need to make a chan of errors
	ch := make(chan map[string]bool)

	for _, url := range urls {
		go func(url string) {
			if query == "" {
				fmt.Println("No query term provided. Provide one with '-query=searchTerm'")
				os.Exit(1)
			}

			response, err := http.Get(url)
			if err != nil {
				fmt.Printf("%s", err)
				os.Exit(1)
			} else {
				defer response.Body.Close()
				contents, err := ioutil.ReadAll(response.Body)
				if err != nil {
					fmt.Printf("%s", err)
					os.Exit(1)
				}

				// fmt.Printf("%s\n", string(contents))

				text, err := html2text.FromString(string(contents))
				if err != nil {
					fmt.Printf("%s", err)
					os.Exit(1)
				}
				// fmt.Printf("%s\n\n\n", text)

				result := strings.Contains(text, query)

				ch <- map[string]bool{url: result}
			}

			return
		}(url)
	}

	// add a timeout here
	for i := 0; i < len(urls); i++ {
		select {
		case result := <-ch:
			log.Printf("%+v", result)
		}
	}

	elapsed := time.Since(start)
	log.Printf("search took %s", elapsed)
}
