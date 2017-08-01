package coveapi

import (
	"net/http"
	"net/url"
	"bytes"
	"html/template"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/mail"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
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
			v.Add("secretKey", secretKey)
			v.Add("accessIdentifier", accessIdentifier)
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

	// w.WriteHeader(http.StatusOK)
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

	requestType := r.FormValue("requestType")
	msg := &mail.Message{
		Sender: "Big Daddy Tizzle <tdfrantz@mhsytems.com>",
		To: []string{r.FormValue("emailAddress")},
		Subject: "COVE API Test Report - " + requestType,
	}
	
	result := struct{
		Success string
		Reason string
		SubmittedTimestamp string
		LastPeekedTimestamp string
	}{
		Success : r.FormValue("success"),
		Reason : r.FormValue("reason"),
		SubmittedTimestamp : r.FormValue("submittedTimestamp"),
		LastPeekedTimestamp : r.FormValue("lastPeekedTimestamp"),
	}

	var body bytes.Buffer
	log.Infof(c, "Request Type: %s", requestType)
	
	if requestType=="Ping"{
			tmpl, err := template.ParseFiles("templates/ping_table.html","templates/email_base.html")
			if err!=nil{
				logErrorAndReturnInternalServerError(c, err, w)
			}
	
			log.Infof(c, "result values: %s, %s, %s, %s", result.Success, result.Reason, result.SubmittedTimestamp, result.LastPeekedTimestamp)		
			if err := tmpl.Execute(&body, result); err!=nil{
				logErrorAndReturnInternalServerError(c, err, w)
			}
			log.Infof(c,"body number 1: %s", body.String())
			msg.Body = body.String()
	}
		// case "ProductMetadataFetch":
		// 	tmpl := template.Must(template.ParseFiles("ping_table.html","email_base.html"))
		// case "ProductInventoryUpdate":
		// 	tmpl := template.Must(template.ParseFiles("ping_table.html","email_base.html"))
		// case "FullTestSuite":
		// 	tmpl := template.Must(template.ParseFiles("ping_table.html","email_base.html"))
	// s := r.FormValue("Success")
	// success := false
	// if s=="true"{
	// 	success = true
	// }

	// if success{
	// 	msg.Body = "We have liftoff!"
	// } else {
	// 	msg.Body = "That's a negatory big buddy."
	// }
	log.Infof(c,"body number 2: %s", body.String())
	if err:=mail.Send(c,msg); err!=nil{
		http.Error(w, err.Error(), http.StatusOK)
	}

	w.WriteHeader(http.StatusOK)
}
