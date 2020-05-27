package gcp

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"google.golang.org/api/compute/v1"
)

//GetInstance returns a compute.Instance object and an error if any
func GetInstance(svc *compute.Service, projectID, zone, instanceName string) (*compute.Instance, error) {

	instanceService := compute.NewInstancesService(svc)
	instanceCall := instanceService.Get(projectID, zone, instanceName)
	resp, err := instanceCall.Do()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

//GetZones return a slice of zone and an error if any.
func GetZones(svc *compute.Service, projectID string) ([]*compute.Zone, error) {

	zones := compute.NewZonesService(svc)
	zonesCall := zones.List(projectID)
	zonesList, err := zonesCall.Do()
	if err != nil {
		return nil, err
	}
	return zonesList.Items, nil
}

//GetZonesString a slice of zones and an error if any.
func GetZonesString(svc *compute.Service, projectID, region string) ([]string, error) {

	zones, err := GetZones(svc, projectID)
	if err != nil {
		return nil, err
	}
	Zones := make([]string, 0)
	for _, v := range zones {
		switch region {
		case "us-east1":
			if strings.Contains(v.Name, region) {
				Zones = append(Zones, v.Name)
			}
		case "us-west1":
			if strings.Contains(v.Name, region) {
				Zones = append(Zones, v.Name)
			}
		case "us-central1":
			if strings.Contains(v.Name, region) {
				Zones = append(Zones, v.Name)
			}
		}
	}
	return Zones, nil
}

//GetSession returns session and an error if any.
func GetSession(ctx context.Context) (session *compute.Service, err error) {
	return compute.NewService(ctx)
}

//GetSubnetName return a subnet name and an error.
func GetSubnetName(svc *compute.Service, projectID, vpc, tier string) (string, error) {

	subnets, err := GetSubnetsName(svc, projectID, vpc)
	if err != nil {
		return "", err
	}
	for _, subnet := range subnets {
		switch {
		case strings.Contains(subnet, tier):
			return subnet, nil
		}
	}
	err = errors.New("No subnet found for the specific tier")
	return "", err
}

//GetPersistantDisks return a slice of persistent disk and error if any.
func GetPersistantDisks(disklist, instanceName, zone, projectID string) ([]*compute.AttachedDisk, error) {

	list := strings.Split(disklist, ",")
	totalDisks := make([]*compute.AttachedDisk, len(list))
	for i, v := range list {
		v = strings.TrimSuffix(strings.ToLower(v), "gb")
		size, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		totalDisks[i] = &compute.AttachedDisk{
			AutoDelete: false,
			Mode:       "READ_WRITE",
			Type:       "PERSISTENT",
			Kind:       "compute#attachedDisk",
			InitializeParams: &compute.AttachedDiskInitializeParams{
				DiskName:   fmt.Sprintf("%s%02d", instanceName, i+1),
				DiskType:   fmt.Sprintf("projects/%s/zones/%s/diskTypes/pd-standard", projectID, zone),
				DiskSizeGb: size,
			},
		}
	}
	return totalDisks, nil
}

//GetImageProjectNfamily returns image projectID name,family and error if any.
func GetImageProjectNfamily(os, version string) (project string, family string, err error) {

	imageProject := []string{"centos-cloud", "gce-uefi-images", "rhel-cloud", "suse-cloud", "windows-cloud", "ubuntu-os-cloud", "ml-images"}
	ubuntuFamily := []string{"ubuntu-1804-lts", "ubuntu-1910", "ubuntu-2004-lts"}
	windwsFamily := []string{"windows-2012-r2", "windows-2016", "windows-2019"}
	centosFamily := []string{"centos-6", "centos-7", "centos-8"}
	debianFamily := []string{"tf-1-13", "tf-1-14", "tf-1-15"}
	redhatFamily := []string{"rhel-6", "rhel-7", "rhel-8"}
	suseElFamily := []string{"sles-12", "sles-15"}
	os = strings.ToLower(strings.TrimSpace(os))
	version = strings.ToLower(strings.TrimSpace(version))
	switch os {
	case "windows":
		if strings.Contains(version, "12") {
			return imageProject[4], windwsFamily[0], nil
		} else if strings.Contains(version, "16") {
			return imageProject[4], windwsFamily[1], nil
		} else if strings.Contains(version, "19") {
			return imageProject[4], windwsFamily[2], nil
		}
	case "centos":
		if strings.Contains(version, "6") {
			return imageProject[0], centosFamily[0], nil
		} else if strings.Contains(version, "7") {
			return imageProject[0], centosFamily[1], nil
		} else if strings.Contains(version, "8") {
			return imageProject[0], centosFamily[2], nil
		}
	case "redhat":
		if strings.Contains(version, "6") {
			return imageProject[2], redhatFamily[0], nil
		} else if strings.Contains(version, "7") {
			return imageProject[2], redhatFamily[1], nil
		} else if strings.Contains(version, "8") {
			return imageProject[2], redhatFamily[2], nil
		}
	case "debian":
		if strings.Contains(version, "13") {
			return imageProject[6], debianFamily[0], nil
		} else if strings.Contains(version, "14") {
			return imageProject[6], debianFamily[1], nil
		} else if strings.Contains(version, "15") {
			return imageProject[6], debianFamily[2], nil
		}
	case "ubuntu":
		if strings.Contains(version, "18") {
			return imageProject[5], ubuntuFamily[0], nil
		} else if strings.Contains(version, "19") {
			return imageProject[5], ubuntuFamily[1], nil
		} else if strings.Contains(version, "20") {
			return imageProject[5], ubuntuFamily[2], nil
		}
	case "suse":
		if strings.Contains(version, "12") {
			return imageProject[3], suseElFamily[0], nil
		} else if strings.Contains(version, "15") {
			return imageProject[3], suseElFamily[1], nil
		}
	}
	err = errors.New("only windows, redhat, suse, debian, ubuntu and centos is allows as OS")
	return "", "", err
}

//GetImage returns selflink of image and error if any.
func GetImage(svc *compute.Service, os, version string) (string, error) {

	project, family, err := GetImageProjectNfamily(os, version)
	if err != nil {
		return "", err
	}
	images := compute.NewImagesService(svc)
	imagesCall := images.GetFromFamily(project, family)
	resp, err := imagesCall.Do()
	if err != nil {
		return "", err
	}
	return resp.SelfLink, nil
}

//GetInstanceName returns instance name following naming standard and error if any.
func GetInstanceName(provider, envname, os, app string) (string, error) {

	env := strings.ToLower(envname)
	os = strings.ToLower(os)
	b := "b"
	d := "d"
	p := "p"
	w := "w"
	x := "x"
	s := "s"
	switch {
	case env == "base":
		if os == "windows" {
			return fmt.Sprintf("%s%s%se%s%s", provider, b, w, d, app), nil
		}
		if os == "redhat" || os == "centos" || os == "suse" || os == "debian" || os == "ubuntu" {
			return fmt.Sprintf("%s%s%sw%s%s", provider, b, x, d, app), nil
		}
	case env == "prod":
		if os == "windows" {
			return fmt.Sprintf("%s%s%se%s%s", provider, p, w, p, app), nil
		}
		if os == "redhat" || os == "centos" || os == "suse" || os == "debian" || os == "ubuntu" {
			return fmt.Sprintf("%s%s%se%s%s", provider, p, x, p, app), nil
		}
	case env == "dev":
		if os == "windows" {
			return fmt.Sprintf("%s%s%se%s%s", provider, s, w, d, app), nil
		}
		if os == "redhat" || os == "centos" || os == "suse" || os == "debian" || os == "ubuntu" {
			return fmt.Sprintf("%s%s%se%s%s", provider, s, x, d, app), nil
		}
	}

	return "", errors.New("Instance name could not be generated with given OS details")
}

//GetSubnetsName returns slice of subnets and error if any.
func GetSubnetsName(svc *compute.Service, projectID, vpc string) ([]string, error) {

	_, subnets, err := GetVPC(svc, projectID, vpc)
	if err != nil {
		return nil, err
	}
	sub := make([]string, 3)
	for _, subnet := range subnets {
		fields := strings.Split(subnet, "/")
		for _, v := range fields {
			switch {
			case strings.Contains(v, "web"):
				sub[0] = v
			case strings.Contains(v, "app"):
				sub[1] = v
			case strings.Contains(v, "db"):
				sub[2] = v
			}
		}
	}
	return sub, nil
}

//GetAllVPC returns all VPCs in the projectID and error if any.
func GetAllVPC(svc *compute.Service, projectID string) (*compute.NetworkList, error) {

	network := compute.NewNetworksService(svc)
	resp := network.List(projectID)
	list, err := resp.Do()
	if err != nil {
		return nil, err
	}

	return list, nil
}

//GetVPCfromEnv returns the VPC name, given the env name and error if any.
func GetVPCfromEnv(svc *compute.Service, projectID, env string) (string, error) {

	list, err := GetAllVPC(svc, projectID)
	if err != nil {
		return "", err
	}
	for _, v := range list.Items {
		switch env {
		case "prod":
			if strings.Contains(v.Name, "-p-vpc") {
				return v.Name, nil
			}
		case "dev":
			if strings.Contains(v.Name, env) {
				return v.Name, nil
			}
		case "base":
			if strings.Contains(v.Name, env) {
				return v.Name, nil
			}
		}
	}
	err = errors.New("VPC could not be found by environment name")
	return "", err
}

//GetVPC return selflink and subnets selflink and error if any.
func GetVPC(svc *compute.Service, projectID, vpc string) (selflink string, subnets []string, err error) {

	network := compute.NewNetworksService(svc)
	networkCall := network.Get(projectID, vpc)
	ops, err := networkCall.Do()
	if err != nil {
		return "", nil, err
	}
	return ops.SelfLink, ops.Subnetworks, nil
}

//GetSubNetwork returns selflink of a given subnetwork and error if any.
func GetSubNetwork(svc *compute.Service, projectID, subName, region string) (string, error) {

	subnet := compute.NewSubnetworksService(svc)
	subnetCall := subnet.Get(projectID, region, subName)
	ops, err := subnetCall.Do()
	if err != nil {
		return "", err
	}
	return ops.SelfLink, nil
}

//CreateSubNetwork create a subnet and returns its selflink and error if any.
func CreateSubNetwork(svc *compute.Service, projectID, subName, region, vpcURL, cidr string) (string, error) {

	subnet := compute.NewSubnetworksService(svc)
	subnetCall := subnet.Insert(projectID, region, &compute.Subnetwork{
		EnableFlowLogs: true,
		IpCidrRange:    cidr,
		Kind:           "compute#subnetwork",
		Name:           subName,
		Network:        vpcURL,
	})
	ops, err := subnetCall.Do()
	if err != nil {
		return "", err
	}
	return ops.SelfLink, nil
}

//GetSubNetworks returns selflink of all subnetworks and error if any.
func GetSubNetworks(svc *compute.Service, projectID string) ([]*compute.UsableSubnetwork, error) {

	subnet := compute.NewSubnetworksService(svc)
	subnetCall := subnet.ListUsable(projectID)
	ops, err := subnetCall.Do()
	if err != nil {
		return nil, err
	}
	return ops.Items, nil
}

//CreateInstance creates an instance within a specified network tier and error if any.
func CreateInstance(svc *compute.Service, projectID, instanceName, desc, subnet, machineType, zone, image, serviceAccount string, disks []*compute.AttachedDisk, labels map[string]string) (string, error) {

	instance := compute.NewInstancesService(svc)
	totalDisks := make([]*compute.AttachedDisk, 0)
	totalDisks = append(totalDisks, &compute.AttachedDisk{
		AutoDelete: true,
		Mode:       "READ_WRITE",
		Type:       "PERSISTENT",
		Kind:       "compute#attachedDisk",
		Boot:       true,
		InitializeParams: &compute.AttachedDiskInitializeParams{
			DiskName:    fmt.Sprintf("%s-os-disk", instanceName),
			SourceImage: image,
		},
	})
	totalDisks = append(totalDisks, disks...)
	input := &compute.Instance{
		CpuPlatform:    "automatic",
		Description:    desc,
		Disks:          totalDisks,
		Labels:         labels,
		MachineType:    fmt.Sprintf("zones/%s/machineTypes/%s", zone, machineType),
		MinCpuPlatform: "Intel Sandy Bridge",
		Name:           instanceName,
		Kind:           "compute#instance",
		NetworkInterfaces: []*compute.NetworkInterface{
			{
				Kind:       "compute#networkInterface",
				Subnetwork: subnet,
			},
		},
		Scheduling: &compute.Scheduling{
			OnHostMaintenance: "MIGRATE",
			Preemptible:       false,
		},
		ServiceAccounts: []*compute.ServiceAccount{
			{
				Email: serviceAccount,
				Scopes: []string{
					"https://www.googleapis.com/auth/devstorage.read_only",
					"https://www.googleapis.com/auth/logging.write",
					"https://www.googleapis.com/auth/monitoring.write",
					"https://www.googleapis.com/auth/servicecontrol",
					"https://www.googleapis.com/auth/service.management.readonly",
					"https://www.googleapis.com/auth/trace.append",
				},
			},
		},
		Status: "PROVISIONING",
		// ShieldedInstanceConfig: &compute.ShieldedInstanceConfig{
		// 	EnableSecureBoot:          false,
		// 	EnableVtpm:                false,
		// 	EnableIntegrityMonitoring: false,
		// },
	}
	instanceCall := instance.Insert(projectID, zone, input)
	ops, err := instanceCall.Do()
	if err != nil {
		return "", err
	}
	return ops.Status, nil
}
