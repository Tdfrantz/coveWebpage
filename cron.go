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
		err := json.Unmarshal(t.Payload, &v)
		if err!=nil{
			logError(c, err)
			continue
		}

		datastoreKey, _:= datastore.DecodeKey(v.Get("datastoreKey"))
		record := new(Record)
		if err := datastore.Get(c, datastoreKey, record); err!=nil{
			logError(c, err)
			continue
		}

		endpoint := ""

		if record.Completed{
			// Record has already been marked as completed by the peek helper
			endpoint = "/helperEmail"
		} else if record.LastPeekedTimestamp.Sub(record.SubmittedTimestamp).Minutes()>=float64(record.PeekTimeout){
			// Record has not been marked as completed by the peek helper, but the peek timeout has been reached
			record.Completed = true
			record.Reason = "Time out value reached."
			_, err := datastore.Put(c, datastoreKey, record)
			if err!=nil{
				logError(c, err)
				continue
			}
			endpoint = "/helperEmail"
		} else {
			endpoint = "/helperPeek"
		}

		pt := taskqueue.NewPOSTTask(endpoint, v)
		if _, err := taskqueue.Add(c, pt, "caller-push-queue"); err!=nil{
			logError(c, err)
		}

		taskqueue.Delete(c, t, "cron-pull-queue")
	}
	
	// THOMAS : Need to write a response
}