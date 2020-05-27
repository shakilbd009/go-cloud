package gcp

import (
	"encoding/json"
	"math/rand"
	"net/http"

	"google.golang.org/api/compute/v1"
)

//GCPrequest object
type GCPrequest struct {
	Environment string `json:"env"`
	Tier        string `json:"tier"`
	Osname      string `json:"os"`
	OsFlavor    string `json:"flavor"`
	Disks       string `json:"disks"`
	CountTO     string `json:"countTO"`
	AppCode     string `json:"appCode"`
	ChangeNum   string `json:"requestNum"`
	MachineType string `json:"machineType"`
	Desc        string `json:"description"`
	Instance    string `json:"instanceName"`
}

//GCPresponse object
type GCPresponse struct {
	InstanceName      string `json:"InstanceName"`
	Status            string `json:"status"`
	NetworkInterfaces string `json:"networkInterfaces,omitempty"`
}

//Get responds to GET method
func Get(w http.ResponseWriter, r *http.Request, svc *compute.Service, payload GCPrequest, projectID, provider, zone, instanceName string) {

	instance, err := GetInstance(svc, projectID, zone, instanceName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := GCPresponse{instance.Name, instance.Status, instance.NetworkInterfaces[0].NetworkIP}
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

//Post makes a POST request.
func Post(w http.ResponseWriter, r *http.Request, svc *compute.Service, payload GCPrequest, projectID, provider, region, serviceAccount string) {

	instanceName, err := GetInstanceName(provider, payload.Environment, payload.Osname, payload.AppCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	vpc, err := GetVPCfromEnv(svc, projectID, payload.Environment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	subnet, err := GetSubnetName(svc, projectID, vpc, payload.Tier)
	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	subnetURL, err := GetSubNetwork(svc, projectID, subnet, region)
	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	image, err := GetImage(svc, payload.Osname, payload.OsFlavor)
	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	zones, err := GetZonesString(svc, projectID, region)
	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	zone := zones[rand.Intn(len(zones)-1)]
	disks, err := GetPersistantDisks(payload.Disks, instanceName, zone, projectID)
	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	labels := map[string]string{"appcode": payload.AppCode, "os": payload.Osname, "env": payload.Environment, "change": payload.ChangeNum}
	status, err := CreateInstance(svc, projectID, instanceName, payload.Desc, subnetURL, payload.MachineType, zone, image, serviceAccount, disks, labels)
	if err != nil {

		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := GCPresponse{instanceName, status, ""}
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}
