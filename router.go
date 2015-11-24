package main

import "github.com/gorilla/mux"

func newRouter() *mux.Router {

	// ** Probably makes sense to just do all this in main

	// Create a new mux Router.
	router := mux.NewRouter().StrictSlash(true)

	// Search route.
	router.HandleFunc("/", indexHandler).Methods("GET")
	router.HandleFunc("/", searchHandler).Methods("POST")

	return router
}
