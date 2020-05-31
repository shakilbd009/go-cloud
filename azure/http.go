package azure

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
)

//AZrequest object
type AZrequest struct {
	Environment string `json:"env"`
	Tier        string `json:"tier"`
	Osname      string `json:"os"`
	OsFlavor    string `json:"flavor"`
	Disks       string `json:"disks"`
	CountTO     string `json:"countTO"`
	AppCode     string `json:"appCode"`
	ChangeNum   string `json:"requestNum"`
	RG          string `json:"resourceGroup"`
	VMname      string `json:"vmName"`
}

//AZimages object
type AZimages struct {
	Name string `json:"name"`
	Ver  string `json:"version"`
}

//AZresponse object
type AZresponse struct {
	VMname string                 `json:"VirtualMachine"`
	Status string                 `json:"status"`
	NIC    compute.VirtualMachine `json:"networkInterfaces,omitempty"`
}

//Get makes GET method to azure.
func Get(w http.ResponseWriter, r *http.Request, subscription string, payload AZrequest) {

	vm, err := GetVM(r.Context(), subscription, payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp := AZresponse{*vm.Name, *vm.VirtualMachineProperties.ProvisioningState, vm}
	data, err := json.MarshalIndent(resp, "", " ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

//Post does a POST method on azure
func Post(w http.ResponseWriter, r *http.Request, subscription, username, passwd string, payload AZrequest) {

	now := time.Now()
	avch, sbch, nich, cmch := make(chan string), make(chan string), make(chan string), make(chan string)
	rdr := io.TeeReader(r.Body, os.Stdout)
	json.NewDecoder(rdr).Decode(&payload)
	AVsetname := fmt.Sprintf("%s-%s-avs-001", provider, payload.Environment)
	//imch := make(chan []compute.VirtualMachineImageResource)
	image, err := GetImagePubOfferSku(payload.Osname, payload.OsFlavor)
	if err != nil {
		log.Println(err)
	}
	go CreateAVS(r.Context(), AVsetname, payload.RG, avSku, azRegion, subscription, avch)
	imageName, version, _ := GetImageVersion(r.Context(), image, payload.Osname, azRegion, subscription)
	vmname := GetVMname(payload.Environment, payload.Osname, payload.AppCode)
	subnetName, err := GetSubnetName(payload.Tier, payload.Environment)
	if err != nil {
		log.Println(err)
	}
	vNetname, err := GetNetwork(payload.Environment)
	if err != nil {
		log.Println(err)
	}
	go GetSubnet(r.Context(), rgNetwork, subnetName, vNetname, subscription, sbch)
	subnet := <-sbch
	avsnm := <-avch
	count := strings.Split(payload.CountTO, "-")
	var wg sync.WaitGroup
	var mx sync.Mutex
	start, err := strconv.Atoi(count[0])
	if err != nil {
		log.Fatalln(err)
	}
	end, err := strconv.Atoi(count[1])
	vmch := make(chan string, (end-start)+1)
	if err != nil {
		log.Fatalln(err)
	}
	resp := make([]AZresponse, 0)
	go func(vmch, ch chan string) {
		for i := start; i <= end; i++ {
			wg.Add(1)
			vmName := fmt.Sprintf("%s%02d", vmname, i)
			nicname := fmt.Sprintf("%s-nic-01", vmName)
			disks := GetDisks(&payload.Disks, vmName)
			go func(vmname, nic string, disks *[]compute.DataDisk) {
				mx.Lock()
				go CreateNIC(r.Context(), payload.RG, nic, subscription, azRegion, subnet, nich)
				go CreateVM(r.Context(), payload.RG, vmname, username, passwd, <-nich, avsnm, azRegion, image.Publisher,
					image.Offer, imageName, version, subscription, &payload.ChangeNum, disks, vmch)
				mx.Unlock()
				resp = append(resp, AZresponse{<-vmch, "Deployed", compute.VirtualMachine{}})
				wg.Done()
			}(vmName, nicname, &disks)
		}
		wg.Wait()
		close(vmch)
		ch <- "Success"
	}(vmch, cmch)
	for {
		select {
		case complete := <-cmch:
			fmt.Printf("deployment completed 🎉🎉🎉, %s\n", complete)
			data, err := json.MarshalIndent(resp, "", "  ")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("something went wrong, please try again later"))
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			w.Write(data)
			done := time.Since(now)
			fmt.Printf("Time took: %.2f minutes", done.Minutes())
			return
		default:
			log.Println("Deploying VM...")
			time.Sleep(time.Millisecond * 1000)
		}
	}
}
