package coveapi

import (
	"net/http"
	// "golang.org/x/net/context"
	"strconv"
	"google.golang.org/appengine/datastore"
	"net/url"
	"encoding/json"
	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/taskqueue"
)

func cron(w http.ResponseWriter, r *http.Request){
	c := appengine.NewContext(r)
	// Pull a group of tasks from the pull queue
	tasks, err := taskqueue.Lease(c, 60, "cron-pull-queue", 60)

	if err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)	
	}

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
			log.Infof(c,"Already completed %s", record.Completed)
			// Record has already been marked as completed by the peek helper
			endpoint = "/helperEmail"
		} else if record.LastPeekedTimestamp.Sub(record.SubmittedTimestamp).Seconds()>=float64(record.PeekTimeout){
			// Record has not been marked as completed by the peek helper, but the peek timeout has been reached
			record.Completed = true
			record.Reason = "Time out value reached."
			log.Infof(c,"Ran out of time...")
			_, err := datastore.Put(c, datastoreKey, record)
			if err!=nil{
				logError(c, err)
				continue
			}
			log.Infof(c,"Record has been marked invalid: %s", record.Reason)
			endpoint = "/helperEmail"
		} else {
			log.Infof(c,"Putting it back in the queue")
			endpoint = "/helperPeek"
		}
		if endpoint=="/helperEmail"{
			v.Add("requestType", record.RequestType)
			v.Add("success", strconv.FormatBool(record.Success))
			v.Add("reason", record.Reason)
			v.Add("submittedTimestamp", record.SubmittedTimestamp.String())
			v.Add("lastPeekedTimestamp", record.LastPeekedTimestamp.String())
		}

		pt := taskqueue.NewPOSTTask(endpoint, v)
		log.Infof(c, "We're going to the endpoint: %s", endpoint)
		if _, err := taskqueue.Add(c, pt, "caller-push-queue"); err!=nil{
			logError(c, err)
		}

		if err := taskqueue.Delete(c, t, "cron-pull-queue"); err!= nil{
			logError(c, err)
		}
	}
	
	w.WriteHeader(http.StatusOK)
}