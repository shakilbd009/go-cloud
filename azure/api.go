package azure

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-07-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
)

//OS object
type OS struct {
	Publisher string
	Offer     string
	Sku       string
}

var (
	provider  = "az"
	avSku     = "aligned"
	azRegion  = "eastus"
	rgNetwork = "az-nonProd-rg-001"
)

//GetVM returns a VM object and error of any
func GetVM(ctx context.Context, subscription string, payload AZrequest) (compute.VirtualMachine, error) {
	client := vmClient(subscription)
	result, err := client.Get(ctx, payload.RG, payload.VMname, "")
	if err != nil {
		return compute.VirtualMachine{}, err
	}
	return result, nil
}

//GetImageVersion returns image version.
func GetImageVersion(ctx context.Context, pubnoffer OS, os, region, subscription string) (string, string, error) {
	versn, err := GetVMimages(ctx, region, pubnoffer.Publisher, pubnoffer.Offer, pubnoffer.Sku, subscription)
	if err != nil {
		return "", "", err
	}
	return pubnoffer.Sku, *(*versn)[len(*versn)-1].Name, nil
}

//GetImagePubOfferSku returns Publisher and Offer
func GetImagePubOfferSku(name, version string) (os OS, e error) {
	windows := []string{"MicrosoftWindowsServer", "WindowsServer"}
	redhat := []string{"RedHat", "RHEL"}
	suse := []string{"SUSE", "SLES"}
	osname := strings.ToLower(name)
	switch {
	case osname == "windows":
		os.Publisher = windows[0]
		os.Offer = windows[1]
		os.Sku = version
		return os, nil
	case osname == "redhat":
		os.Publisher = redhat[0]
		os.Offer = redhat[1]
		os.Sku = version
		return os, nil
	case osname == "suse":
		os.Publisher = suse[0]
		os.Offer = suse[1]
		os.Sku = version
		return os, nil
	}
	e = errors.New("only Windows, RedHat And Suse is allowed for OS")
	return
}

//GetDisks with return []compute.DataDisk.
func GetDisks(disklist *string, vmname string) []compute.DataDisk {
	sizes := strings.Split(*disklist, ",")
	disks := make([]compute.DataDisk, 0)
	for i, v := range sizes {
		v = strings.TrimSuffix(strings.ToLower(v), "gb")
		size, err := strconv.Atoi(v)
		if err != nil {
			panic(err)
		}
		disks = append(disks, compute.DataDisk{
			Lun:          to.Int32Ptr(int32(i)),
			Name:         to.StringPtr(fmt.Sprintf("%s%02d", vmname, i+1)),
			DiskSizeGB:   to.Int32Ptr(int32(size)),
			Caching:      compute.CachingTypesReadWrite,
			CreateOption: compute.DiskCreateOptionTypesEmpty,
			ManagedDisk: &compute.ManagedDiskParameters{
				StorageAccountType: compute.StorageAccountTypesStandardLRS,
			},
		})
	}
	return disks
}

//GetAVS returns an Availability set if exist.
func GetAVS(ctx context.Context, rg, name, subscription string) (string, error) {
	client := compute.NewAvailabilitySetsClient(subscription)
	authorizer, err := auth.NewAuthorizerFromCLI()
	if err == nil {
		client.Authorizer = authorizer
	}
	resp, err := client.Get(ctx, rg, name)
	if err != nil {
		return "", err
	}
	return *resp.Name, nil
}

//GetVMname returns VM name.
func GetVMname(envname, os, app string) string {
	env := strings.ToLower(envname)
	os = strings.ToLower(os)
	b := "b"
	d := "d"
	p := "p"
	w := "w"
	x := "x"
	s := "s"
	switch {
	case env == "base" && os == "windows":
		return fmt.Sprintf("az%s%sw%s%s", b, w, d, app)
	case env == "base" && os == "redhat":
		return fmt.Sprintf("az%s%sw%s%s", b, x, d, app)
	case env == "base" && os == "suse":
		return fmt.Sprintf("az%s%sw%s%s", b, x, d, app)
	case env == "prod" && os == "windows":
		return fmt.Sprintf("az%s%sw%s%s", p, w, p, app)
	case env == "prod" && os == "redhat":
		return fmt.Sprintf("az%s%sw%s%s", p, x, p, app)
	case env == "prod" && os == "suse":
		return fmt.Sprintf("az%s%sw%s%s", p, x, p, app)
	case env == "nonprod" && os == "windows":
		return fmt.Sprintf("az%s%sw%s%s", s, w, d, app)
	case env == "nonprod" && os == "redhat":
		return fmt.Sprintf("az%s%sw%s%s", s, x, d, app)
	case env == "nonprod" && os == "suse":
		return fmt.Sprintf("az%s%sw%s%s", s, x, d, app)
	}
	return ""
}

//CreateNIC creates a NIC and returns its ID over a chan.
func CreateNIC(ctx context.Context, rg, nicname, subscription, loc, subid string, ch chan string) {
	client := network.NewInterfacesClient(subscription)
	authorizer, err := auth.NewAuthorizerFromCLI()
	if err == nil {
		client.Authorizer = authorizer
	}
	//defer errRecover()
	resp, err := client.CreateOrUpdate(ctx,
		rg,
		nicname,
		network.Interface{
			Location: to.StringPtr(loc),
			InterfacePropertiesFormat: &network.InterfacePropertiesFormat{
				IPConfigurations: &[]network.InterfaceIPConfiguration{
					{
						Name: to.StringPtr("ipConfig"),
						InterfaceIPConfigurationPropertiesFormat: &network.InterfaceIPConfigurationPropertiesFormat{
							PrivateIPAllocationMethod: network.Dynamic,
							PrivateIPAddressVersion:   network.IPv4,
							Subnet: &network.Subnet{
								ID: to.StringPtr(subid),
							},
						},
					},
				},
				//EnableAcceleratedNetworking: to.BoolPtr(true),
			},
		},
	)
	//time.Sleep(time.Second * 5)
	inter, err := resp.Result(client)
	if err != nil {
		panic(err)
	}
	ch <- *inter.ID
}

func vmClient(subscription string) compute.VirtualMachinesClient {
	vmClient := compute.NewVirtualMachinesClient(subscription)
	authorizer, err := auth.NewAuthorizerFromCLI()
	if err == nil {
		vmClient.Authorizer = authorizer
	}
	return vmClient
}

//CreateVM create a VM.
func CreateVM(ctx context.Context, rg, vmname, username, passwd, nic, avsID, region, publisher, offer, sku, version, subscription string, crq *string, datadisks *[]compute.DataDisk, ch chan string) {
	client := vmClient(subscription)
	//defer errRecover()
	resp, err := client.CreateOrUpdate(ctx,
		rg,
		vmname,
		compute.VirtualMachine{
			Location: to.StringPtr(region),
			VirtualMachineProperties: &compute.VirtualMachineProperties{
				HardwareProfile: &compute.HardwareProfile{
					VMSize: compute.VirtualMachineSizeTypesStandardB1s,
				},
				StorageProfile: &compute.StorageProfile{
					OsDisk: &compute.OSDisk{
						Name:         to.StringPtr(fmt.Sprintf("%s-os", vmname)),
						Caching:      compute.CachingTypesReadWrite,
						CreateOption: compute.DiskCreateOptionTypesFromImage,
						ManagedDisk: &compute.ManagedDiskParameters{
							StorageAccountType: compute.StorageAccountTypesStandardLRS,
						},
					},
					ImageReference: &compute.ImageReference{
						Publisher: to.StringPtr(publisher),
						Offer:     to.StringPtr(offer),
						Sku:       to.StringPtr(sku),
						Version:   to.StringPtr(version),
					},
					DataDisks: datadisks,
				},
				OsProfile: &compute.OSProfile{
					ComputerName:  to.StringPtr(vmname),
					AdminUsername: to.StringPtr(username),
					AdminPassword: to.StringPtr(passwd),
				},
				NetworkProfile: &compute.NetworkProfile{
					NetworkInterfaces: &[]compute.NetworkInterfaceReference{
						{
							ID: to.StringPtr(nic),
						},
					},
				},
				AvailabilitySet: &compute.SubResource{
					ID: to.StringPtr(avsID),
				},
			},
			Tags: map[string]*string{"Request#": crq},
		},
	)
	if err != nil {
		panic(err)
	}
	err = resp.WaitForCompletionRef(ctx, client.Client)
	if err != nil {
		panic(err)
	}
	inter, err := resp.Result(client)
	if err != nil {
		panic(err)
	}
	if err != nil {
		panic(err)
	}
	ch <- *inter.Name
}

//CreateAVS creates an AVset and returns ID over a chan
func CreateAVS(ctx context.Context, name, rg, sku, loc, subscription string, ch chan string) {
	client := compute.NewAvailabilitySetsClient(subscription)
	authorizer, err := auth.NewAuthorizerFromCLI()
	if err == nil {
		client.Authorizer = authorizer
	}
	//defer errRecover()
	avSet, err := client.CreateOrUpdate(ctx,
		rg,
		name,
		compute.AvailabilitySet{
			AvailabilitySetProperties: &compute.AvailabilitySetProperties{
				PlatformFaultDomainCount:  to.Int32Ptr(2),
				PlatformUpdateDomainCount: to.Int32Ptr(5),
			},
			Sku: &compute.Sku{
				Name: to.StringPtr(sku),
			},
			Name:     to.StringPtr(name),
			Location: to.StringPtr(loc),
		},
	)
	if err != nil {
		panic(err)
	}
	ch <- *avSet.ID
	close(ch)
}

//GetVMimages makes a call to azure and sends image details over a chan
func GetVMimages(ctx context.Context, region, publisher, offer, skus, subscription string) (*[]compute.VirtualMachineImageResource, error) {
	client := compute.NewVirtualMachineImagesClient(subscription)
	authorizer, err := auth.NewAuthorizerFromCLI()
	if err == nil {
		client.Authorizer = authorizer
	}
	//defer errRecover()
	result, err := client.List(ctx, region, publisher, offer, skus, "", nil, "")
	if err != nil {
		return nil, err
	}
	return result.Value, nil
}

//GetSubnetName returns subnet name and an error.
func GetSubnetName(tier, envname string) (string, error) {
	baseAppSub := []string{"az-base-sub-001", "az-base-app-sub-002"}
	ProdAppSub := []string{"az-Prod-sub-001", "az-Prod-app-sub-002"}
	nonProdAppSub := []string{"az-nonProd-sub-001", "az-nonProd-app-sub-002"}
	envname = strings.TrimSpace(strings.ToLower(envname))
	tiername := strings.TrimSpace(strings.ToLower(tier))
	switch {
	case envname == "nonprod" && tiername == "app":
		return nonProdAppSub[1], nil
	case envname == "nonprod" && tiername == "web":
		return nonProdAppSub[0], nil
	case envname == "prod" && tiername == "app":
		return ProdAppSub[1], nil
	case envname == "prod" && tiername == "web":
		return ProdAppSub[0], nil
	case envname == "base" && tiername == "app":
		return baseAppSub[1], nil
	case envname == "base" && tiername == "web":
		return baseAppSub[0], nil
	}
	err := errors.New("please pass an env as prod, base or nonprod. And tier can only be app or web")
	return "", err
}

//GetNetwork returns a vnet name or an error.
func GetNetwork(envname string) (string, error) {
	network := []string{"az-base-vnet-001", "az-nonProd-vnet-001", "az-Prod-vnet-001"}
	envname = strings.TrimSpace(strings.ToLower(envname))
	switch {
	case envname == "base":
		return network[0], nil
	case envname == "nonprod":
		return network[1], nil
	case envname == "prod":
		return network[2], nil
	}
	err := errors.New("provide a network envname, i.e base, dev, prod")
	return "", err
}

//GetSubnet sends the subnet ID over a chan
func GetSubnet(ctx context.Context, rg, sname, vname, subscription string, ch chan string) {
	client := network.NewSubnetsClient(subscription)
	authorizer, err := auth.NewAuthorizerFromCLI()
	//defer errRecover()
	if err == nil {
		client.Authorizer = authorizer
	}
	resp, err := client.Get(ctx, rg, vname, sname, "")
	if err != nil {
		panic(err)
	}
	ch <- *resp.ID
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

func getImageVersion(ctx context.Context, pubnoffer OS, ch chan *[]compute.VirtualMachineImageResource, subscription, region string) []AZimages {
	go getVMimages(ctx, region, pubnoffer.Publisher, pubnoffer.Offer, pubnoffer.Sku, subscription, ch)
	versn := (*<-ch)
	img := AZimages{}
	versions := make([]AZimages, len(versn))
	for i, v := range versn {
		img.Name = *v.Name
		img.Ver = pubnoffer.Sku
		versions[i] = img
	}
	return versions
}

func getVMimages(ctx context.Context, region, publisher, offer, skus, subscription string, ch chan *[]compute.VirtualMachineImageResource) {
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
