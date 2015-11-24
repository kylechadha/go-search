package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jaytaylor/html2text"
)

const maxReqs = 20

var urls = []string{"https://godoc.org/fmt#Sprintf", "https://getgb.io/", "https://golang.org/pkg/runtime/pprof/"}

func main() {
	// Let's do a bit of back-of-the-envelope profiling.
	start := time.Now()

	query := flag.String("query", "", "search term")
	flag.Parse()

	results := search(*query)

	for _, result := range results {
		fmt.Printf("%+v\n", result)
	}

	elapsed := time.Since(start)
	log.Printf("search took %s", elapsed)
}

// write tests :)
func search(q string) []map[string]bool {

	// Need to set up error handling story

	// maybe you can remove the urls in a separate range, and send them on a channel
	// and then the goroutines pull them off as they can do them
	// will have to figure this part out

	// wg.Add(maxReqs)
	// for i:=0; i<maxReqs; i++ {
	//     go func() {
	//         for {
	//             url, ok := <-ch
	//             if !ok {
	//                 wg.Done()
	//                 return
	//             }
	//             fetch(url)
	//         }
	//     }()
	// }

	// for i:=0; i<50; i++ {
	//     ch <- i // add i to the queue
	// }

	// close(ch)
	// wg.Wait()

	ch := make(chan map[string]bool)
	for _, url := range urls {
		go func(url string) {
			if q == "" {
				fmt.Println("No query term provided. Provide one with '-query=searchTerm'")
				os.Exit(1)
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

				text, q = strings.ToLower(text), strings.ToLower(q)
				result := strings.Contains(text, q)

				ch <- map[string]bool{url: result}
			}

			return
		}(url)
	}

	var results []map[string]bool

	// add a timeout here
	for i := 0; i < len(urls); i++ {
		select {
		case result := <-ch:
			results = append(results, result)
		}
	}

	return results
}

// contents, err := ioutil.ReadAll(response.Body)
// if err != nil {
// 	fmt.Printf("%s", err)
// 	os.Exit(1)
// }
// fmt.Printf("%s\n", string(contents))
