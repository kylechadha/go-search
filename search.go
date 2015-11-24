package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jaytaylor/html2text"
)

var urls = []string{"https://godoc.org/fmt#Sprintf", "https://getgb.io/", "https://golang.org/pkg/runtime/pprof/"}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello")
}

// curl -XPOST -H 'Content-Type: application/json' -d '{"searchTerm": "feature"}' http://localhost:3000/
func searchHandler(w http.ResponseWriter, r *http.Request) {
	// Let's do a bit of back-of-the-envelope profiling.
	start := time.Now()

	// Need to set up error handling story
	// Create an app handler

	ch := make(chan string)

	for _, url := range urls {
		go func(url string) {
			log.Println(url)
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

				ch <- text
			}

			return
		}(url)
	}

	// add a timeout here
	for i := 0; i < len(urls); i++ {
		select {
		case text := <-ch:
			fmt.Printf("%s\n\n\n", text[:200])
		}
	}

	elapsed := time.Since(start)
	log.Printf("search took %s", elapsed)
}
