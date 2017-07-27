package coveapi

import (
	"net/http"
	"google.golang.org/appengine"
	"google.golang.org/appengine/datastore"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
	"encoding/json"
)

var client = &http.Client{}

func helperPost(w http.ResponseWriter,r *http.Request){
	c := appengine.NewContext(r)

	datastoreKey := datastore.NewKey(c,"Record",r.FormValue("datastoreKey"),0,nil)

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
	
	if block.RequestType=="ProductInventoryUpdate"{
		block.Payload = map[string]interface{}{
			"sku" : "100",
			"action" : []map[string]interface{}{{"new_value" : "1",
						 "status" : "sold",
						 "action" : "change_qty",
						}},
		}
	}

	// Make the COVE API post call
	payload, err := json.Marshal(block)	
	if err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}

	interaction, err := post(client, accessIdentifier, secretKey, payload)
	if err != nil{
		logErrorAndReturnInternalServerError(c, err, w)
	} 
	
	// Update the datastore with the interaction value
	record := new(Record)
	if err := datastore.Get(c, datastoreKey, record); err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}

	record.Interaction = interaction
	if _, err := datastore.Put(c, datastoreKey, record); err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}
}

func logErrorAndReturnInternalServerError(c context.Context, err error, w http.ResponseWriter){
	log.Errorf(c, err.Error())
	http.Error(w, err.Error(), http.StatusInternalServerError)
}