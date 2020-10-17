package aws

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
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
	InstanceName string
	Provider     string
	VPCid        *string
	SubnetID     *string
	SecurityGID  *string
	AmiID        *string
	Key          *string
	DisksF       []ec2.BlockDeviceMapping
	Config       aws.Config
	Ctx          context.Context
}

//AWSresponse object
type AWSresponse struct {
	InstanceName      string `json:"InstanceName"`
	Status            string `json:"status"`
	NetworkInterfaces string `json:"networkInterfaces,omitempty"`
}

type BuildFunc func() error

func Builder(fns ...BuildFunc) error {
	for _, fn := range fns {
		if err := fn(); err != nil {
			return err
		}
	}
	return nil
}

//Post makes a POST request to aws api.
func Post(w http.ResponseWriter, payload AWSrequest) {

	if err := Builder(
		payload.GetVpcID,
		payload.GetSubnet,
		payload.GetAMI,
		payload.GetSecurityGroup,
		payload.PrepareDisks,
		payload.GetInstanceName,
	); err == nil {
		responses, err := payload.BuildEC2()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data, err := json.MarshalIndent(responses, "", "  ")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		w.Write(data)
	} else {
		fmt.Println("error happend here")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
