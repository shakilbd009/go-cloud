package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/shakilbd009/go-cloud/aws"
	"github.com/shakilbd009/go-cloud/azure"
	"github.com/shakilbd009/go-cloud/gcp"
)

var (
	subscription   = ""
	username       = "usertest"
	passwd         = "useRword123$"
	gregion        = "us-east1"
	aregion        = "us-east-2"
	zone           = "us-east1-c"
	key            = "my-key-pair"
	projectID      = ""
	desc           = "my go sdk deployent test"
	serviceAccount = ""
)

func main() {
	parseFlags()
	http.HandleFunc("/azure", azureHandler)
	http.HandleFunc("/gcp", gcpHandler)
	http.HandleFunc("/aws", awsHandler)
	log.Fatalln(http.ListenAndServe(":9999", nil))
}

func parseFlags() {

	flag.StringVar(&projectID, "prjID", "", "project ID needs to be passed")
	flag.StringVar(&serviceAccount, "serviceAccount", "", "service account needs to be passed")
	flag.StringVar(&subscription, "subscription", "", "suscription needs to be passed")
	flag.Parse()
	if projectID == "" || serviceAccount == "" || subscription == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
}

func awsHandler(w http.ResponseWriter, r *http.Request) {
	provider := "aws"
	cfg, err := aws.GetNewSession(aregion)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()
	payload := aws.AWSrequest{}
	rdr := io.TeeReader(r.Body, os.Stdout)
	if err := json.NewDecoder(rdr).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	payload.Ctx = r.Context()
	payload.Provider = provider
	payload.Config = cfg
	if r.Method == http.MethodPost {
		aws.Post(w, payload)
	}
}

func gcpHandler(w http.ResponseWriter, r *http.Request) {

	provider := "gcp"
	payload := gcp.GCPrequest{}
	defer r.Body.Close()
	rdr := io.TeeReader(r.Body, os.Stdout)
	if err := json.NewDecoder(rdr).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	svc, err := gcp.GetSession(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodPost {
		gcp.Post(w, r, svc, payload, projectID, provider, gregion, serviceAccount)
	}
	if r.Method == http.MethodGet {
		gcp.Get(w, r, svc, payload, projectID, provider, zone, payload.Instance)
	}
}

func azureHandler(w http.ResponseWriter, r *http.Request) {

	payload := azure.AZrequest{}
	defer r.Body.Close()
	rdr := io.TeeReader(r.Body, os.Stdout)
	if err := json.NewDecoder(rdr).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if r.Method == http.MethodPost {
		azure.Post(w, r, subscription, username, passwd, payload)
	}
	if r.Method == http.MethodGet {
		azure.Get(w, r, subscription, payload)
	}
}
