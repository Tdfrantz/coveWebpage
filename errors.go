package coveapi

import (
	"net/http"
	"golang.org/x/net/context"
	"google.golang.org/appengine/log"
)

func logError(c context.Context, err error){
	log.Errorf(c, err.Error())
}

func logErrorAndReturnInternalServerError(c context.Context, err error, w http.ResponseWriter){
	log.Errorf(c, err.Error())
	http.Error(w, err.Error(), http.StatusInternalServerError)
}