package main

import (
	"fmt"
	"log"
	"net/http"
)

var urls = []string{"https://godoc.org/fmt#Sprintf", "https://getgb.io/", "https://golang.org/pkg/runtime/pprof/"}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello")
}

// curl -XPOST -H 'Content-Type: application/json' -d '{"dayOfTheWeek": "Saturday", "dayOfTheWeekType": "Weekend", "successfulWakeUp": true, "morningWork": true, "morningWorkType": "omnia app", "workedOut": true, "workedOutType": "swam", "plannedNextDay": true}' http://localhost:3000/api/day
func searchHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("searchHandler")
}
