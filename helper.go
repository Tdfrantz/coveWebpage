package coveapi

import (
	"net/http"
	"net/url"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine/mail"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
	"encoding/json"
)

func helperPost(w http.ResponseWriter,r *http.Request){
	c := appengine.NewContext(r)

	datastoreKey, _ := datastore.DecodeKey(r.FormValue("datastoreKey"))
	log.Infof(c, "%s", datastoreKey.String())
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
			t := &taskqueue.Task{}			
			v := url.Values{}

			v.Add("datastoreKey", datastoreKey.Encode())
			v.Add("requestType", block.RequestType)
			v.Add("secretKey", secretKey)
			v.Add("accessIdentifier", accessIdentifier)
			v.Add("databaseName",block.DatabaseName)
			v.Add("databaseUsername", block.DatabaseLoginName)
			v.Add("databasePassword", block.DatabaseLoginPassword)
			v.Add("databaseServer", block.DatabaseServer)
			v.Add("apiEndpoint", block.ApiEndpoint)
			v.Add("rmsType", block.RMSType)
			v.Add("instanceName", block.InstanceName)

			t.Method="PULL"
			t.Payload, err =json.Marshal(v)
			if err!=nil{
				logErrorAndReturnInternalServerError(c, err, w)
			}

			if _, taskErr := taskqueue.Add(c, t, "cron-pull-queue"); taskErr!=nil{
				return taskErr
			}

			return nil
		}, nil)

	if err!=nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	w.Write([]byte("Interaction was successfully submitted"))
}

func helperPeek(w http.ResponseWriter,r *http.Request){

}

func helperEmail(w http.ResponseWriter, r *http.Request){
	c := appengine.NewContext(r)
	msg := &mail.Message{
		Sender: "Big Daddy Tizzle <tdfrantz@mhsytems.com>",
		To: []string{"tdfrantz@mhsytems.com"},
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
		logErrorAndReturnInternalServerError(c, err, w)
	}

}

func logErrorAndReturnInternalServerError(c context.Context, err error, w http.ResponseWriter){
	log.Errorf(c, err.Error())
	http.Error(w, err.Error(), http.StatusInternalServerError)
}