package main

import (
	"context"
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

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/shakilbd009/go-cloud/azure"
)

var (
	subscription = os.Args[1]
	provider     = "az"
	region       = "eastus"
	RGname       = "az-nonProd-rg-001"
	avSku        = "aligned"
	username     = "usertest"
	passwd       = "useRword123$"
)

type azDeployment struct {
	Environment string `json:"env"`
	Tier        string `json:"tier"`
	Osname      string `json:"os"`
	OsFlavor    string `json:"flavor"`
	Disks       string `json:"disks"`
	CountTO     string `json:"countTO"`
	AppCode     string `json:"appCode"`
	ChangeNum   string `json:"requestNum"`
}

type images struct {
	Name string `json:"name"`
	Ver  string `json:"version"`
}

type response struct {
	VMname string `json:"VirtualMachine"`
	Status string `json:"status"`
}

func main() {
	http.HandleFunc("/azure", az)
	log.Fatalln(http.ListenAndServe(":9999", nil))
}

func az(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	avch, sbch, nich, cmch := make(chan string), make(chan string), make(chan string), make(chan string)
	rdr := io.TeeReader(r.Body, os.Stdout)
	payload := azDeployment{}
	json.NewDecoder(rdr).Decode(&payload)
	AVsetname := fmt.Sprintf("%s-%s-avs-001", provider, payload.Environment)
	imch := make(chan *[]compute.VirtualMachineImageResource)
	image, err := azure.GetImagePubOfferSku(payload.Osname, payload.OsFlavor)
	if err != nil {
		log.Println(err)
	}
	go azure.CreateAVS(r.Context(), AVsetname, RGname, avSku, region, subscription, avch)
	imageName, version := azure.GetImageVersion(r.Context(), image, payload.Osname, region, subscription, imch)
	vmname := azure.GetVMname(payload.Environment, payload.Osname, payload.AppCode)
	subnetName, err := azure.GetSubnetName(payload.Tier, payload.Environment)
	if err != nil {
		log.Println(err)
	}
	vNetname, err := azure.GetNetwork(payload.Environment)
	if err != nil {
		log.Println(err)
	}
	go azure.GetSubnet(r.Context(), RGname, subnetName, vNetname, subscription, sbch)
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
	resp := make([]response, 0)
	go func(vmch, ch chan string) {
		for i := start; i <= end; i++ {
			wg.Add(1)
			vmName := fmt.Sprintf("%s%02d", vmname, i)
			nicname := fmt.Sprintf("%s-nic-01", vmName)
			disks := azure.GetDisks(&payload.Disks, vmName)
			go func(vmname, nic string, disks *[]compute.DataDisk) {
				mx.Lock()
				go azure.CreateNIC(r.Context(), RGname, nic, subscription, region, subnet, nich)
				go azure.CreateVM(r.Context(), RGname, vmname, username, passwd, <-nich, avsnm, region, image.Publisher,
					image.Offer, imageName, version, subscription, &payload.ChangeNum, disks, vmch)
				mx.Unlock()
				resp = append(resp, response{<-vmch, "Deployed"})
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

func getImageVersion(ctx context.Context, pubnoffer azure.OS, ch chan *[]compute.VirtualMachineImageResource) []images {
	go getVMimages(ctx, region, pubnoffer.Publisher, pubnoffer.Offer, pubnoffer.Sku, ch)
	versn := (*<-ch)
	img := images{}
	versions := make([]images, len(versn))
	for i, v := range versn {
		img.Name = *v.Name
		img.Ver = pubnoffer.Sku
		versions[i] = img
	}
	return versions
}

func getVMimages(ctx context.Context, region, publisher, offer, skus string, ch chan *[]compute.VirtualMachineImageResource) {
	client := compute.NewVirtualMachineImagesClient(subscription)
	authorizer, err := auth.NewAuthorizerFromCLI()
	if err == nil {
		client.Authorizer = authorizer
	}
	defer errRecover()
	result, err := client.List(ctx, region, publisher, offer, skus, "", nil, "")
	if err != nil {
		panic(err)
	}
	ch <- result.Value
	close(ch)
}

func errRecover() {
	if r := recover(); r != nil {
		fmt.Println("An error has occured:")
		fmt.Printf("%s\n\n", strings.Repeat("💀", 20))
		fmt.Println(r)
		fmt.Printf("\n")
		fmt.Println(strings.Repeat("💀", 20))
		//os.Exit(1) //optional, if you want to stop the excution if error occurs.
	}
}
