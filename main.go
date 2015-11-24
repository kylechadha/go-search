package main

import (
	"flag"
	"net/http"
	"os"
)

var query string

func main() {

	query = *flag.String("term", "", "search term")

	flag.Parse()

	// Define the application configuration.
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Create a new router and listen on 'port'.
	router := newRouter()
	http.ListenAndServe(":"+port, router)

}
