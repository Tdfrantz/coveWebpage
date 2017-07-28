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
	// "google.golang.org/appengine/log"
)

type Record struct{
	RequestType string
	SubmittedTimestamp time.Time
	LastPeekedTimestamp time.Time
	Interaction string
	Reason string
	Success bool
	Completed bool
}

func init(){
	http.HandleFunc("/", index)
	http.HandleFunc("/caller", caller)
	http.HandleFunc("/helperPost", helperPost)
	http.HandleFunc("/helperPeek", helperPeek)
	http.HandleFunc("/cron", cron)
	http.HandleFunc("/helperEmail", helperEmail)
}

func index(w http.ResponseWriter, r *http.Request){
	render(w, "static/index.html", nil)
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

func caller(w http.ResponseWriter, r *http.Request){

	c := appengine.NewContext(r)

	v, p, _ := validate(r,c)
	if v{
		// //formData := map[string]string{}
		// render(w, "static/index.html", nil)
		// return
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	}

	// Create a datastore entry for this request
	record := &Record{
		RequestType:r.FormValue("requestType"),
		SubmittedTimestamp:time.Now(),
		LastPeekedTimestamp:time.Now(),
		Success:false,
	}

	key := datastore.NewIncompleteKey(c, "Record", nil)
	err := datastore.RunInTransaction(c, func(c context.Context) error{
			key, putErr := datastore.Put(c, key, record)
			if putErr!=nil{
				return putErr	
			}
			
			v := url.Values{}
			v.Add("datastoreKey", key.Encode())
			v.Add("email_address", "tdfrantz@mhsystems.com")
			v.Add("requestType", p["requestType"])
			v.Add("secretKey", p["secretKey"])
			v.Add("accessIdentifier", p["accessIdentifier"])
			v.Add("databaseName", p["databaseName"])
			v.Add("databaseUsername", p["databaseUsername"])
			v.Add("databasePassword", p["databasePassword"])
			v.Add("databaseServer", p["databaseServer"])
			v.Add("apiEndpoint", p["apiEndpoint"])
			v.Add("rmsType", p["rmsType"])
			v.Add("instanceName", p["instanceName"])

			t := taskqueue.NewPOSTTask("/helperPost", v)

			if _, taskErr := taskqueue.Add(c, t, "caller-push-queue"); taskErr!=nil{
				return taskErr
			}

			return nil
		}, nil)

	if err!=nil{
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

}

func validate(r *http.Request,c context.Context) (bool, map[string]string, map[string]string){
	missingParams := false
	params := make(map[string]string)
	errors := make(map[string]string)


	if requestType := r.FormValue("requestType"); requestType == ""{
		missingParams=true
		errors["requestType"] = "Invalid request type value."
	} else {
		params["requestType"] = requestType
	}
	
	if secretKey := r.FormValue("secretKey"); secretKey == ""{
		missingParams=true
		errors["secretKey"] = "Invalid secret key value."
	} else {
		params["secretKey"] = secretKey
	}

	if accessIdentifier := r.FormValue("accessIdentifier"); accessIdentifier == ""{
		missingParams=true
		errors["accessIdentifier"] = "Invalid access identifier value."
	} else {
		params["accessIdentifier"] = accessIdentifier
	}

	if databaseName := r.FormValue("databaseName"); databaseName == ""{
		missingParams=true
		errors["databaseName"] = "Invalid database name value."
	} else {
		params["databaseName"] = databaseName
	}

	if databaseUsername := r.FormValue("databaseUsername"); databaseUsername == ""{
		missingParams=true
		errors["databaseUsername"] = "Invalid database login name value."
	} else {
		params["databaseUsername"] = databaseUsername
	}

	if databasePassword := r.FormValue("databasePassword"); databasePassword == ""{
		missingParams=true
		errors["databasePassword"] = "Invalid database login password value."
	} else {
		params["databasePassword"] = databasePassword
	}

	if databaseServer := r.FormValue("databaseServer"); databaseServer == ""{
		missingParams=true
		errors["databaseServer"] = "Invalid databse server value."
	} else {
		params["databaseServer"] = databaseServer
	}

	if apiEndpoint := r.FormValue("apiEndpoint"); apiEndpoint == ""{
		missingParams=true
		errors["apiEndpoint"] = "Invalid api endpoint value."
	} else {
		params["apiEndpoint"] = apiEndpoint
	}

	if rmsType := r.FormValue("rmsType"); rmsType == ""{
		missingParams=true
		errors["rmsType"] = "Invalid rms type value."
	} else {
		params["rmsType"] = rmsType
	}

	if instanceName := r.FormValue("instanceName"); instanceName == ""{
		missingParams=true
		errors["instanceName"] = "Invalid instance name value."
	} else {
		params["instanceName"] = instanceName
	}

	return missingParams, params, errors
}