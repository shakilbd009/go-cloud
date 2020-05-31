package aws

import (
	"encoding/json"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
)

//AWSrequest object
type AWSrequest struct {
	Environment  string `json:"env"`
	Tier         string `json:"tier"`
	Osname       string `json:"os"`
	OsFlavor     string `json:"flavor"`
	Disks        string `json:"disks"`
	Min          int64  `json:"min"`
	Max          int64  `json:"max"`
	AppCode      string `json:"appCode"`
	ChangeNum    string `json:"requestNum"`
	InstanceType string `json:"instanceType"`
}

//AWSresponse object
type AWSresponse struct {
	InstanceName      string `json:"InstanceName"`
	Status            string `json:"status"`
	NetworkInterfaces string `json:"networkInterfaces,omitempty"`
}

//Post makes a POST request to aws api.
func Post(w http.ResponseWriter, r *http.Request, cfg aws.Config, payload AWSrequest, provider, key string) {

	vpcID, err := GetVpcID(r.Context(), cfg, payload.Environment)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	subnet, err := GetSubnet(r.Context(), cfg, vpcID, payload.Tier)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	amiID, err := GetAMI(r.Context(), cfg, payload.Osname, payload.OsFlavor)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	sgID, err := GetSecurityGroup(r.Context(), cfg, vpcID, payload.Tier)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	disks, err := PrepareDisks(payload.Disks)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	instanceName, err := GetInstanceName(provider, payload.Environment, payload.Osname, payload.AppCode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	status, err := CreateEC2(r.Context(), cfg, instanceName, subnet, amiID, key, sgID, payload.Environment, payload.ChangeNum, payload.InstanceType, payload.Min, payload.Max, disks)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	responses := make([]AWSresponse, 0)
	for _, v := range status.Instances {
		state, err := v.State.Name.MarshalValue()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		responses = append(responses, AWSresponse{
			InstanceName:      *v.InstanceId,
			Status:            state,
			NetworkInterfaces: *v.NetworkInterfaces[0].PrivateIpAddress,
		})
	}
	data, err := json.MarshalIndent(responses, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	w.Write(data)
}
