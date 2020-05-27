package aws

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

//GetNewSession return a aws.Config or an error
func GetNewSession(region string) (aws.Config, error) {

	cfg, err := external.LoadDefaultAWSConfig(external.WithDefaultRegion(region))
	if err != nil {
		return aws.Config{}, err
	}
	return cfg, nil
}

//GetSubnet retruns a subnetID with given tier name or an error if any
func GetSubnet(ctx context.Context, cfg aws.Config, vpcID, tier string) (string, error) {

	subnet := ec2.New(cfg)
	req := subnet.DescribeSubnetsRequest(nil)
	resp, err := req.Send(ctx)
	if err != nil {
		return "", err
	}
	for _, sub := range resp.Subnets {
		if *sub.VpcId == vpcID {
			switch tier {
			case "app":
				if strings.Contains(*sub.Tags[0].Value, tier) {
					return *sub.SubnetId, nil
				}
			case "web":
				if strings.Contains(*sub.Tags[0].Value, tier) {
					return *sub.SubnetId, nil
				}
			case "db":
				if strings.Contains(*sub.Tags[0].Value, tier) {
					return *sub.SubnetId, nil
				}
			}
		}
	}
	return "", errors.New("Subnet not found with given tier name")
}

//GetVPCs returns all vpcs in the region.
func GetVPCs(ctx context.Context, cfg aws.Config) ([]ec2.Vpc, error) {

	vpc := ec2.New(cfg)
	input := &ec2.DescribeVpcsInput{}
	req := vpc.DescribeVpcsRequest(input)
	resp, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	return resp.Vpcs, nil
}

//GetSecurityGroup returns the SG id for the given tier or an error if any.
func GetSecurityGroup(ctx context.Context, cfg aws.Config, vpcID, tier string) (string, error) {

	tier = strings.ToLower(tier)
	sg := ec2.New(cfg)
	input := &ec2.DescribeSecurityGroupsInput{}
	req := sg.DescribeSecurityGroupsRequest(input)
	reps, err := req.Send(ctx)
	if err != nil {
		return "", err
	}
	for _, sg := range reps.SecurityGroups {
		if *sg.VpcId == vpcID {
			switch tier {
			case "app":
				if strings.Contains(*sg.GroupName, tier) {
					return *sg.GroupId, nil
				}
			case "db":
				if strings.Contains(*sg.GroupName, tier) {
					return *sg.GroupId, nil
				}
			case "web":
				if strings.Contains(*sg.GroupName, tier) {
					return *sg.GroupId, nil
				}
			}
		}
	}
	return "", errors.New("SecurityGroup not found with the tier provided")
}

//GetVpcID returns VPCid for a specific environment.
func GetVpcID(ctx context.Context, cfg aws.Config, env string) (string, error) {

	env = strings.ToLower(env)
	vpcs, err := GetVPCs(ctx, cfg)
	if err != nil {
		return "", err
	}

	for _, vpc := range vpcs {

		switch env {
		case "nonprod":
			if strings.Contains(*vpc.Tags[0].Value, "nonProd") {
				return *vpc.VpcId, nil
			}
		case "prod":
			if strings.Contains(*vpc.Tags[0].Value, "prod") {
				return *vpc.VpcId, nil
			}
		case "base":
			if strings.Contains(*vpc.Tags[0].Value, "base") {
				return *vpc.VpcId, nil
			}
		}
	}

	return "", errors.New("only dev, prod, base is allowed as enviroment")
}

//CreateSG creates a new Security group.
func CreateSG(cfg aws.Config, sgn, vpcID, sgDesc string) (*ec2.CreateSecurityGroupResponse, error) {

	input := &ec2.CreateSecurityGroupInput{
		Description: aws.String(sgDesc),
		GroupName:   aws.String(sgn),
		VpcId:       aws.String(vpcID),
	}
	SG := ec2.New(cfg)
	req := SG.CreateSecurityGroupRequest(input)
	res, err := req.Send(context.Background())
	if err != nil {
		return nil, err
	}
	return res, nil
}

//CreateVPC creates a new VPC.
func CreateVPC(cfg aws.Config, block string) (ec2.CreateVpcResponse, error) {

	input := &ec2.CreateVpcInput{
		CidrBlock: aws.String(block),
	}
	VPC := ec2.New(cfg)
	req := VPC.CreateVpcRequest(input)
	res, err := req.Send(context.TODO())
	if err != nil {
		return ec2.CreateVpcResponse{}, err
	}
	return *res, nil
}

//CreateSubnet creates a new subnet.
func CreateSubnet(ctx context.Context, cfg aws.Config, vpc *string, az []ec2.AvailabilityZone, cidr string) (*ec2.Subnet, error) {

	sub := ec2.New(cfg)
	input := &ec2.CreateSubnetInput{
		AvailabilityZone: az[rand.Intn(len(az)-1)].ZoneName,
		VpcId:            vpc,
		CidrBlock:        aws.String(cidr),
	}
	req := sub.CreateSubnetRequest(input)
	res, err := req.Send(ctx)
	if err != nil {
		return &ec2.Subnet{}, err
	}
	return res.Subnet, nil
}

//GetAZs gets the AZs for a region.
func GetAZs(cfg aws.Config) ([]ec2.AvailabilityZone, error) {

	az := ec2.New(cfg)
	input := &ec2.DescribeAvailabilityZonesInput{}
	req := az.DescribeAvailabilityZonesRequest(input)
	res, err := req.Send(context.TODO())
	if err != nil {
		return nil, err
	}
	return res.AvailabilityZones, nil
}

//CreateKey creates a new keypair for login.
func CreateKey(ctx context.Context, cfg aws.Config, keyPair string) (string, error) {

	key := ec2.New(cfg)
	input := &ec2.CreateKeyPairInput{
		KeyName: aws.String(keyPair),
	}
	req := key.CreateKeyPairRequest(input)
	res, err := req.Send(ctx)
	if err != nil {
		return "", err
	}
	return *res.KeyName, nil
}

//GetAllKeys returns a ssh keypair for login or an error if any.
func GetAllKeys(ctx context.Context, cfg aws.Config, keyPair string) ([]ec2.KeyPairInfo, error) {

	key := ec2.New(cfg)
	input := &ec2.DescribeKeyPairsInput{
		Filters: []ec2.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{keyPair},
			},
		},
	}
	req := key.DescribeKeyPairsRequest(input)
	res, err := req.Send(ctx)
	if err != nil {
		return nil, err
	}
	return res.KeyPairs, nil
}

//PrepareDisks returns a slice of disks of type ec2.BlockDeviceMapping or an error if any.
func PrepareDisks(disks string) ([]ec2.BlockDeviceMapping, error) {
	device := make([]ec2.BlockDeviceMapping, 0)
	Disks := strings.Split(disks, ",")
	deviceName := []string{"/dev/sdb", "/dev/sdc", "/dev/sdd", "/dev/sde"}
	for i, v := range Disks {
		v = strings.TrimSuffix(strings.ToLower(v), "gb")
		size, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return nil, err
		}
		device = append(device, ec2.BlockDeviceMapping{
			DeviceName: aws.String(deviceName[i]),
			Ebs: &ec2.EbsBlockDevice{
				DeleteOnTermination: aws.Bool(false),
				Encrypted:           aws.Bool(true),
				VolumeSize:          aws.Int64(size),
				VolumeType:          ec2.VolumeTypeGp2,
			},
		})
	}
	return device, nil
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
		if os == "redhat" || os == "centos" || os == "suse" || os == "debian" || os == "ubuntu" || os == "amazon" {
			return fmt.Sprintf("%s%s%sw%s%s", provider, b, x, d, app), nil
		}
	case env == "prod":
		if os == "windows" {
			return fmt.Sprintf("%s%s%se%s%s", provider, p, w, p, app), nil
		}
		if os == "redhat" || os == "centos" || os == "suse" || os == "debian" || os == "ubuntu" || os == "amazon" {
			return fmt.Sprintf("%s%s%se%s%s", provider, p, x, p, app), nil
		}
	case env == "dev":
		if os == "windows" {
			return fmt.Sprintf("%s%s%se%s%s", provider, s, w, d, app), nil
		}
		if os == "redhat" || os == "centos" || os == "suse" || os == "debian" || os == "ubuntu" || os == "amazon" {
			return fmt.Sprintf("%s%s%se%s%s", provider, s, x, d, app), nil
		}
	}

	return "", errors.New("Instance name could not be generated with given OS details")
}

//CreateEC2 creates a new EC2 instance
func CreateEC2(ctx context.Context, cfg aws.Config, instanceName, sub, imageID, key, sgID, envv, requestNum, cpu string, min, max int64, disks []ec2.BlockDeviceMapping) (*ec2.RunInstancesResponse, error) {

	Ec2 := ec2.New(cfg)
	input := &ec2.RunInstancesInput{
		BlockDeviceMappings: disks,
		ImageId:             aws.String(imageID),
		KeyName:             aws.String(key),
		SubnetId:            aws.String(sub),
		MaxCount:            aws.Int64(max),
		MinCount:            aws.Int64(min),
		InstanceType:        ec2.InstanceTypeT2Micro,
		SecurityGroupIds:    []string{sgID},
		TagSpecifications: []ec2.TagSpecification{
			{
				ResourceType: ec2.ResourceTypeInstance,
				Tags: []ec2.Tag{
					{Key: aws.String("env"),
						Value: aws.String(envv),
					},
					{Key: aws.String("ChangeNum"),
						Value: aws.String(string(requestNum)),
					},
					{Key: aws.String("Name"),
						Value: aws.String(instanceName),
					},
				},
			},
		},
	}
	req := Ec2.RunInstancesRequest(input)
	res, err := req.Send(ctx)
	if err != nil {
		return &ec2.RunInstancesResponse{}, err
	}
	return res, nil
}

//GetOSami returns
func GetOSami(ctx context.Context, cfg aws.Config, os, version string) (string, error) {

	os = strings.TrimSpace(strings.ToLower(os))
	version = strings.TrimSpace(strings.ToLower(version))

	switch os {
	case "windows":
		return fmt.Sprintf("Windows_Server-%s-English-Full-Base-*", version), nil
	case "redhat":
		return fmt.Sprintf("RHEL-%s_HVM-*", version), nil
	case "suse":
		return fmt.Sprintf("suse-sles-%s", version), nil
	case "amazon":
		return fmt.Sprintf("amzn%s-ami-hvm-*", version), nil
	default:
		return "", errors.New("Windows, RedHat, Suse and amazon linux are acceptable as OS")
	}
}

type awsAMI struct {
	ID           string
	CreationTime time.Time
}

type awsAMIs []*awsAMI

func (a awsAMIs) Len() int           { return len(a) }
func (a awsAMIs) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a awsAMIs) Less(i, j int) bool { return a[i].CreationTime.After(a[j].CreationTime) }

//GetAMI return the latest AMI or error if any
func GetAMI(ctx context.Context, cfg aws.Config, os, version string) (string, error) {

	ami := ec2.New(cfg)
	amiS, err := GetOSami(ctx, cfg, os, version)
	if err != nil {
		return "", err
	}
	input := &ec2.DescribeImagesInput{
		Filters: []ec2.Filter{
			{
				Name:   aws.String("name"),
				Values: []string{amiS},
			},
		},
	}
	req := ami.DescribeImagesRequest(input)
	amis, err := req.Send(ctx)
	if err != nil {
		return "", err
	}
	amiToSort := make([]*awsAMI, 0)
	for _, ami := range amis.Images {
		amiCreationTime, err := time.Parse(time.RFC3339, *ami.CreationDate)
		if err != nil {
			return "", err
		}
		if len(ami.ProductCodes) > 0 {
			continue
		}
		amiToSort = append(amiToSort, &awsAMI{
			ID:           *ami.ImageId,
			CreationTime: amiCreationTime,
		})
	}
	sort.Sort(awsAMIs(amiToSort))
	return amiToSort[0].ID, nil
}
