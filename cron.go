package coveapi

import (
	"net/http"
	// "golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"net/url"
	"encoding/json"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
)

func cron(w http.ResponseWriter, r *http.Request){
	c := appengine.NewContext(r)
	log.Infof(c,"Entering the cron")
	// Pull a group of tasks from the pull queue
	tasks, err := taskqueue.Lease(c, 60, "cron-pull-queue", 60)

	if err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)	
	}

	log.Infof(c,"Got the tasks!")

	for _, t := range tasks{
		log.Infof(c, "Cron payload: %s", t.Payload)
		v := url.Values{}
		err := json.Unmarshal(t.Payload, v)
		if err!=nil{
			logErrorAndReturnInternalServerError(c, err, w)
		}

		datastoreKey, _:= datastore.DecodeKey(v.Get("datastoreKey"))
		record := new(Record)
		if err := datastore.Get(c, datastoreKey, record); err!=nil{
			logErrorAndReturnInternalServerError(c, err, w)
		}

		if record.Completed{
			// Send it on to the email endpoint
		} else if record.LastPeekedTimestamp.Sub(record.SubmittedTimestamp).Minutes()>=2{
			log.Infof(c, "More than 2 minutes!")
			// Mark it as completed with reason timed_out and send it on to the email endpoint
		} else {
			// Send it back to the push queue with the helperPeek endpoint
		}

		// Delete the task
	}
}