package coveapi

import (
	"net/http"
	"net/url"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/mail"
	"golang.org/x/net/context"
	// "google.golang.org/appengine/log"
	"encoding/json"
	"time"
)

func helperPost(w http.ResponseWriter,r *http.Request){
	c := appengine.NewContext(r)

	datastoreKey, _ := datastore.DecodeKey(r.FormValue("datastoreKey"))
	secretKey := r.FormValue("secretKey")
	accessIdentifier := r.FormValue("accessIdentifier")

	block := connectionBlock{}
	block.DatabaseName = r.FormValue("databaseName")
	block.RMSType = r.FormValue("rmsType")
	block.RequestType = r.FormValue("requestType")
	block.DatabaseLoginPassword = r.FormValue("databasePassword")
	block.ApiEndpoint = r.FormValue("apiEndpoint")
	block.InstanceName = r.FormValue("instanceName")
	block.DatabaseServer = r.FormValue("databaseServer")
	block.DatabaseLoginName = r.FormValue("databaseUsername")

	// if block.RequestType=="ProductInventoryUpdate"{
	// 	block.Payload = map[string]interface{}{
	// 		"sku" : "100",
	// 		"action" : []map[string]interface{}{{"new_value" : "1",
	// 					 "status" : "sold",
	// 					 "action" : "change_qty",
	// 					}},
	// 	}
	// }

	// Make the COVE API post call
	payload, err := json.Marshal(block)	
	if err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}

	interaction, err := post(c, accessIdentifier, secretKey, payload)
	if err != nil{
		logErrorAndReturnInternalServerError(c, err, w)
	} 
	// Get the datastore record to be updated
	record := new(Record)
	if err := datastore.Get(c, datastoreKey, record); err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}

	// Do the record update and task creation in the same transaction
	err = datastore.RunInTransaction(c, func(c context.Context) error{

			//Update the record
			record.Interaction = interaction
			if _, putErr := datastore.Put(c, datastoreKey, record); err!=nil{
				return putErr
			}
			//Create and submit the task
			t := &taskqueue.Task{Method:"PULL",}			
			v := url.Values{}

			v.Add("datastoreKey", datastoreKey.Encode())
			// v.Add("requestType", block.RequestType)
			v.Add("secretKey", secretKey)
			v.Add("accessIdentifier", accessIdentifier)
			// v.Add("databaseName",block.DatabaseName)
			// v.Add("databaseUsername", block.DatabaseLoginName)
			// v.Add("databasePassword", block.DatabaseLoginPassword)
			// v.Add("databaseServer", block.DatabaseServer)
			// v.Add("apiEndpoint", block.ApiEndpoint)
			// v.Add("rmsType", block.RMSType)
			// v.Add("instanceName", block.InstanceName)
			v.Add("emailAddress", r.FormValue("emailAddress"))
			v.Add("interaction", interaction)

			t.Payload, err =json.Marshal(v)
			if err != nil{
				return err
			}

			if _, taskErr := taskqueue.Add(c, t, "cron-pull-queue"); taskErr!=nil{
				return taskErr
			}
			return nil
		}, nil)

	if err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}

	w.WriteHeader(http.StatusOK)
}

func helperPeek(w http.ResponseWriter,r *http.Request){

	c := appengine.NewContext(r)

	secretKey := r.FormValue("secretKey")
	accessIdentifier := r.FormValue("accessIdentifier")

	block := connectionBlock{}
	block.Payload = make(map[string]interface{})
	block.Payload["interaction"] = r.FormValue("interaction")
	block.Payload["timeout_millis"] = 1024

	payload, err := json.Marshal(block.Payload)	
	if err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}

	body, err := peek(c, accessIdentifier, secretKey, payload)
	if err != nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}
	
	var jbody map[string]interface{}
	err = json.Unmarshal(body, &jbody)
	if err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}

	// Update the last peeked timestamp with the current time
	datastoreKey, _ := datastore.DecodeKey(r.FormValue("datastoreKey"))
	record := new(Record)
	if err := datastore.Get(c, datastoreKey, record); err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}

	record.LastPeekedTimestamp = time.Now()

	// Parse the jbody response

	// First check if response_message_unavailable is in there 
	// If it is then we're not done, send it on to the cron queue		
	v := url.Values{}
	
	if message, ok := jbody["message"].(map[string]interface{}); ok {
		record.Completed = true
		if payload, ok := message["payload"].(map[string]interface{}); ok {
			if success, ok := payload["success"].(bool); ok{
				if success{
					record.Success = true
				}
			}
		}

		if asset_url, ok := jbody["asset_url"]; ok {
			record.Result = asset_url.(string)
		}
	} else {
		v.Add("secretKey", secretKey)
		v.Add("accessIdentifier", accessIdentifier)
		v.Add("interaction", block.Payload["interaction"].(string))
	}

	v.Add("datastoreKey", datastoreKey.Encode())
	v.Add("emailAddress", r.FormValue("emailAddress"))

	t := &taskqueue.Task{Method:"PULL",}	
	t.Payload, err = json.Marshal(v)

	if err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}

	if _, err := datastore.Put(c, datastoreKey, record); err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}

	if _, err := taskqueue.Add(c, t, "cron-pull-queue"); err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}

	w.WriteHeader(http.StatusOK)
}

func helperEmail(w http.ResponseWriter, r *http.Request){
	c := appengine.NewContext(r)
	msg := &mail.Message{
		Sender: "tdfrantz@mhsytems.com",
		To: []string{r.FormValue("emailAddress")},
		Subject: "See you tonight",
	}

	s := r.FormValue("success")
	success := false
	if s=="true"{
		success = true
	}

	if success{
		msg.Body = "We have liftoff!"
	} else {
		msg.Body = "That's a negatory big buddy."
	}

	if err:=mail.Send(c,msg); err!=nil{
		http.Error(w, err.Error(), http.StatusOK)
	}

	w.WriteHeader(http.StatusOK)
}
