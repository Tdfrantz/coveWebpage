package coveapi

import (
	"net/http"
	"time"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine"
)

type Record struct{
	RequestType string
	SubmittedTimestamp time.Time
}

func init(){
	http.HandleFunc("/", index)
	http.HandleFunc("/caller", caller)
	http.HandleFunc("/post", post)
}

func index(w http.ResponseWriter, r *http.Request){
	http.ServeFile(w, r, "static/index.html")
}

func caller(w http.ResponseWriter, r *http.Request){
	requestType := r.FormValue("requestType")
	c := appengine.NewContext(r)
	
	// Create a datastore entry for this request
	record := &Record{
		RequestType:requestType,
		SubmittedTimestamp:time.Now(),
	}

	key := datastore.NewIncompleteKey(c, "Record", nil)
	if _, err := datastore.Put(c, key, record); err != nil {
		// Handle err...jk
	}

	// Send it off to the push queue

}