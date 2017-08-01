package coveapi

import (
	"net/http"
	"net/url"
	"time"
	"html/template"
	"golang.org/x/net/context"
	"google.golang.org/appengine/datastore"
	"google.golang.org/appengine/taskqueue"
	"google.golang.org/appengine"
	"strconv"
	"google.golang.org/appengine/log"
)

type FormField struct{
	Value string
	ErrorMsg string
}

type Record struct{
	RequestType string
	SubmittedTimestamp time.Time
	LastPeekedTimestamp time.Time
	PeekTimeout int
	Interaction string
	Reason string
	Success bool
	Completed bool
	Result string
}

func init(){
	http.HandleFunc("/", index)
	http.HandleFunc("/helperPost", helperPost)
	http.HandleFunc("/helperPeek", helperPeek)
	http.HandleFunc("/cron", cron)
	http.HandleFunc("/helperEmail", helperEmail)
	http.HandleFunc("/favicon.ico", favicon)
}

func index(w http.ResponseWriter, r *http.Request){
	c := appengine.NewContext(r)
	if r.Method=="POST"{
		log.Infof(c, "Form has been submitted. Validating data now.")
		if missingData, formFields:= validate(r); missingData{
			log.Infof(c, "Data is missing! Rendering index")


			// Make sure the formFields are in the right order 
			// tableFields := []FormField{
			// 	formFields["peekTimeout"],
			// 	formFields["accessIdentifier"],
			// 	formFields["secretKey"],
			// 	formFields["databaseName"],
			// 	formFields["databaseUsername"],
			// 	formFields["databasePassword"],
			// 	formFields["databaseServer"],
			// 	formFields["apiEndpoint"],
			// 	formFields["rmsType"],
			// 	formFields["instanceName"],
			// 	formFields["emailAddress"],
			// }


			render(w, "templates/index.html", formFields)
		} else {
			log.Infof(c, "We're good to go, on to sender.")
			caller(c, w, formFields)
			data := make(map[string]FormField)
			data["confirmation"]=FormField{Value:"Thank you! Your request has been sent submitted. A report will be e-mailed to you shortly."}
			render(w, "templates/index.html", data)
		}
	} else {
		log.Infof(c, "Not a POST, rendering basic index")
		data := make(map[string]FormField)
		render(w, "templates/index.html", data)
	}
}

func render(w http.ResponseWriter, filename string, data interface{}){
	tmpl, err := template.ParseFiles(filename)
	if err!=nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
	if err := tmpl.Execute(w, data); err!=nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func caller(c context.Context, w http.ResponseWriter, data map[string]FormField){

	timeout, _ := strconv.Atoi(data["peekTimeout"].Value)
	// Create a datastore entry for this request
	record := &Record{
		RequestType:data["requestType"].Value,
		SubmittedTimestamp:time.Now(),
		LastPeekedTimestamp:time.Now(),
		PeekTimeout:timeout,
		Success:false,
		Completed:false,
	}

	key := datastore.NewIncompleteKey(c, "Record", nil)
	err := datastore.RunInTransaction(c, func(c context.Context) error{
			key, putErr := datastore.Put(c, key, record)
			if putErr!=nil{
				return putErr	
			}
			
			v := url.Values{}
			v.Add("datastoreKey", key.Encode())
			v.Add("emailAddress", data["emailAddress"].Value)
			v.Add("requestType", record.RequestType)
			v.Add("secretKey", data["secretKey"].Value)
			v.Add("accessIdentifier", data["accessIdentifier"].Value)
			v.Add("databaseName", data["databaseName"].Value)
			v.Add("databaseUsername", data["databaseUsername"].Value)
			v.Add("databasePassword", data["databasePassword"].Value)
			v.Add("databaseServer", data["databaseServer"].Value)
			v.Add("apiEndpoint", data["apiEndpoint"].Value)
			v.Add("rmsType",data["rmsType"].Value)
			v.Add("instanceName", data["instanceName"].Value)

			t := taskqueue.NewPOSTTask("/helperPost", v)

			if _, taskErr := taskqueue.Add(c, t, "caller-push-queue"); taskErr!=nil{
				return taskErr
			}

			return nil
		}, nil)

	if err!=nil{
		logErrorAndReturnInternalServerError(c, err, w)
	}

}

func validate(r *http.Request) (missingParams bool, formFields map[string]FormField){
	missingParams = false
	formFields = make(map[string]FormField)

	requestType := FormField{ErrorMsg:"",}
	if requestType.Value = r.FormValue("requestType"); requestType.Value == ""{
		missingParams=true
		requestType.ErrorMsg = "Invalid request type value."
	}
	formFields["requestType"] = requestType
	
	secretKey := FormField{ErrorMsg:"",}
	if secretKey.Value = r.FormValue("secretKey"); secretKey.Value == ""{
		missingParams=true
		secretKey.ErrorMsg = "Invalid secret key value."
	}
	formFields["secretKey"] = secretKey

	accessIdentifier := FormField{ErrorMsg:"",}
	if accessIdentifier.Value = r.FormValue("accessIdentifier"); accessIdentifier.Value == ""{
		missingParams=true
		accessIdentifier.ErrorMsg = "Invalid access identifier value."
	}
	formFields["accessIdentifier"] = accessIdentifier

	databaseName := FormField{ErrorMsg:""}
	if databaseName.Value = r.FormValue("databaseName"); databaseName.Value == ""{
		missingParams=true
		databaseName.ErrorMsg = "Invalid database name value."
	}
	formFields["databaseName"] = databaseName

	databaseUsername := FormField{ErrorMsg:"",}
	if databaseUsername.Value = r.FormValue("databaseUsername"); databaseUsername.Value == ""{
		missingParams=true
		databaseUsername.ErrorMsg = "Invalid database login name value."
	}
	formFields["databaseUsername"] = databaseUsername

	databasePassword := FormField{ErrorMsg:"",}
	if databasePassword.Value = r.FormValue("databasePassword"); databasePassword.Value == ""{
		missingParams=true
		databasePassword.ErrorMsg = "Invalid database login password value."
	}
	formFields["databasePassword"] = databasePassword

	databaseServer := FormField{ErrorMsg:"",}
	if databaseServer.Value = r.FormValue("databaseServer"); databaseServer.Value == ""{
		missingParams=true
		databaseServer.ErrorMsg = "Invalid database server value."
	}
	formFields["databaseServer"] = databaseServer

	apiEndpoint := FormField{ErrorMsg:"",}
	if apiEndpoint.Value = r.FormValue("apiEndpoint"); apiEndpoint.Value == ""{
		missingParams=true
		apiEndpoint.ErrorMsg = "Invalid api endpoint value."
	}
	formFields["apiEndpoint"] = apiEndpoint

	rmsType := FormField{ErrorMsg:"",}
	if rmsType.Value = r.FormValue("rmsType"); rmsType.Value == ""{
		missingParams=true
		rmsType.ErrorMsg = "Invalid rms type value."
	}
	formFields["rmsType"] = rmsType

	instanceName := FormField{ErrorMsg:"",}
	if instanceName.Value = r.FormValue("instanceName"); instanceName.Value == ""{
		missingParams=true
		instanceName.ErrorMsg = "Invalid instance name value."
	}
	formFields["instanceName"] = instanceName

	emailAddress := FormField{ErrorMsg:"",}
	if emailAddress.Value = r.FormValue("emailAddress"); emailAddress.Value == ""{
		missingParams=true
		emailAddress.ErrorMsg = "Invalid email address value."
	}
	formFields["emailAddress"] = emailAddress

	peekTimeout := FormField{ErrorMsg:"",}
	if peekTimeout.Value = r.FormValue("peekTimeout"); peekTimeout.Value == ""{
		missingParams=true
		peekTimeout.ErrorMsg = "Invalid peek timeout value."
	}
	formFields["peekTimeout"] = peekTimeout

	return missingParams, formFields
}

func favicon(w http.ResponseWriter, r *http.Request){
	http.ServeFile(w, r, "assets/favicon.ico")
}