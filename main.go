package main

import (
	"net/http"
	"os"
)

func main() {

	// Define the application configuration.
	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	// Create a new router and listen on 'port'.
	router := newRouter()
	http.ListenAndServe(":"+port, router)

}
