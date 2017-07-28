package coveapi

import (	
	"net/http"
	"google.golang.org/appengine/urlfetch"
	"encoding/hex"
	"encoding/json"
	"golang.org/x/net/context"
	// "google.golang.org/appengine/log"
	"crypto/md5"
	"io/ioutil"
	"bytes"
	"errors"
)

var covePostURL = "https://mhsgen2two.appspot.com/api/proxy/v0/post"
var covePeekURL = "https://mhsgen2two.appspot.com/api/proxy/v0/peek"
var coveDumpURL = "https://mhsgen2two.appspot.com/api/proxy/v0/dumpeventlog"

type connectionBlock struct{
	DatabaseName string				`json:"database_name,omitempty"`
	RMSType string					`json:"rms_type,omitempty"`
	RequestType string				`json:"request,omitempty"`
	DatabaseLoginPassword string 	`json:"database_login_password,omitempty"`
	ApiEndpoint string				`json:"api_endpoint,omitempty"`
	InstanceName string				`json:"instance_name,omitempty"`
	DatabaseServer string 			`json:"database_server,omitempty"`
	DatabaseLoginName string		`json:"database_login_name,omitempty"`
	Payload map[string]interface{}	`json:"payload,omitempty"`
}

func calcPayloadSignature(secretKey string, accessIdentifier string, payload []byte) string{
	h := md5.New()
	h.Write([]byte(secretKey))
	h.Write([]byte(accessIdentifier))
	h.Write(payload)
	return hex.EncodeToString(h.Sum(nil))
}

func post(ctx context.Context, accessIdentifier string, secretKey string, payload []byte) (string, error){

	client := urlfetch.Client(ctx)

	req, _ := http.NewRequest("POST", covePostURL, bytes.NewReader(payload))
	req.Header.Add("X-Access-Identifier", accessIdentifier)
	req.Header.Add("X-Payload-Signature", calcPayloadSignature(secretKey,accessIdentifier,payload))
	res, err := client.Do(req)
	interaction := ""

	if err!=nil{
		return interaction, err
	}
	defer res.Body.Close()

	b, _ := ioutil.ReadAll(res.Body)
	var resJson map[string]interface{}
	err = json.Unmarshal(b, &resJson)
	if err!=nil{
		// should never go wrong, this means that what I got back from COVE wasn't proper JSON
		// mark it as bad and log it
		return interaction, err
	}

	if interaction, ok := resJson["interaction"].(string); ok {
		return interaction, nil
	} else {
		reason, _ := json.Marshal(resJson["reason"])
		return interaction, errors.New(string(reason[:]))
	}
}