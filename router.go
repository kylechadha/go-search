package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

func newRouter() *mux.Router {

	// Create a new mux Router.
	router := mux.NewRouter().StrictSlash(true)

	// Index route.
	router.HandleFunc("/", indexHandler).Methods("GET")
	router.HandleFunc("/", searchHandler).Methods("POST")

	// Public routes.
	router.PathPrefix("/libs").Handler(http.FileServer(http.Dir("./public/")))
	router.PathPrefix("/scripts").Handler(http.FileServer(http.Dir("./public/")))
	router.PathPrefix("/styles").Handler(http.FileServer(http.Dir("./public/")))
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("./public/views")))

	return router
}
