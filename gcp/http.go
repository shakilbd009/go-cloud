package gcp

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

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
	InstanceName      string `json:"InstanceName,omitempty"`
	Status            string `json:"status,omitempty"`
	NetworkInterfaces string `json:"networkInterfaces,omitempty"`
	Zone              string `json:"zone,omitempty"`
	Error             string `json:"error,omitempty"`
}

//Get responds to GET method
func Get(w http.ResponseWriter, r *http.Request, svc *compute.Service, payload GCPrequest, projectID, provider, zone, instanceName string) {

	instance, err := GetInstance(svc, projectID, zone, instanceName)
	if err != nil {
		errResp(w, err)
		return
	}
	resp := GCPresponse{
		InstanceName:      instanceName,
		Zone:              zone,
		NetworkInterfaces: instance.NetworkInterfaces[0].NetworkIP,
	}
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		errResp(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func errResp(w http.ResponseWriter, err error) {

	resp := GCPresponse{
		Error: err.Error(),
	}
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	w.Write(data)
}

//Post makes a POST request.
func Post(w http.ResponseWriter, r *http.Request, svc *compute.Service, payload GCPrequest, projectID, provider, region, serviceAccount string) {

	instanceName, err := GetInstanceName(provider, payload.Environment, payload.Osname, payload.AppCode)
	if err != nil {
		errResp(w, err)
		return
	}

	vpc, err := GetVPCfromEnv(svc, projectID, payload.Environment)
	if err != nil {
		errResp(w, err)
		return
	}
	subnet, err := GetSubnetName(svc, projectID, vpc, payload.Tier)
	if err != nil {
		errResp(w, err)
		return
	}
	subnetURL, err := GetSubNetwork(svc, projectID, subnet, region)
	if err != nil {
		errResp(w, err)
		return
	}
	image, err := GetImage(svc, payload.Osname, payload.OsFlavor)
	if err != nil {
		errResp(w, err)
		return
	}
	zones, err := GetZonesString(svc, projectID, region)
	if err != nil {
		errResp(w, err)
		return
	}
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	count := strings.Split(payload.CountTO, "-")
	start, err := strconv.Atoi(count[0])
	if err != nil {
		errResp(w, err)
		return
	}
	stop, err := strconv.Atoi(count[1])
	if err != nil {
		errResp(w, err)
		return
	}
	resp := make([]GCPresponse, 0, (stop-start)+1)
	labels := map[string]string{"appcode": payload.AppCode, "os": payload.Osname, "env": payload.Environment, "change": payload.ChangeNum}
	var wg sync.WaitGroup
	for i := start; i <= stop; i++ {
		wg.Add(1)
		go func(i int, payload GCPrequest) {
			zone := zones[rnd.Intn(len(zones))]
			instanceNm := fmt.Sprintf("%s%02d", instanceName, i)
			disks, err := GetPersistantDisks(payload.Disks, instanceNm, zone, projectID)
			if err != nil {
				errResp(w, err)
				return
			}
			status, err := CreateInstance(svc, projectID, instanceNm, payload.Desc, subnetURL, payload.MachineType, zone, image, serviceAccount, disks, labels)
			if err != nil {
				errResp(w, err)
				return
			}
			resp = append(resp, GCPresponse{
				InstanceName: instanceNm,
				Status:       status,
				Zone:         zone,
			})
			wg.Done()
		}(i, payload)

	}
	wg.Wait()

	//resp := GCPresponse{instanceName, status, ""}
	data, err := json.MarshalIndent(resp, "", "  ")
	if err != nil {
		errResp(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}
