package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ram"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func uploadToOSS(region, akid, aksecret, name, path, bucketname string, timeout int) error {
	log.Debugf("Uploading %s to OSS", path)

	ossclient, err := oss.New(fmt.Sprintf("http://oss-%s.aliyuncs.com", region),
		akid, aksecret, oss.Timeout(10, int64(timeout)))
	if err != nil {
		return fmt.Errorf("failed to initilize ossclient")
	}

	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}
	defer f.Close()

	bucket, err := ossclient.Bucket(bucketname)
	if err != nil {
		return fmt.Errorf("error getting bucket: %v", err)
	}

	err = bucket.PutObjectFromFile(name, path)
	if err != nil {
		return fmt.Errorf("error uploading to OSS: %v", err)
	}

	log.Debugf("Uploaded %s to OSS", path)
	return nil
}

func configRAMPolicy(region, akid, aksecret string) error {
	client, err := ram.NewClientWithAccessKey(region, akid, aksecret)
	if err != nil {
		return fmt.Errorf("Failed to create RAM client: %v", err)
	}

	// Create AliyunECSImageImportDefaultRole if not exist
	{
		request := ram.CreateGetRoleRequest()
		request.Scheme = "https"
		request.RoleName = "AliyunECSImageImportDefaultRole"

		response, _ := client.GetRole(request)
		if response == nil {
			request := ram.CreateCreateRoleRequest()
			request.Scheme = "https"
			request.RoleName = "AliyunECSImageImportDefaultRole"
			request.AssumeRolePolicyDocument = `{
				"Statement": [
				{
					"Action": "sts:AssumeRole",
					"Effect": "Allow",
					"Principal": {
					"Service": [
						"ecs.aliyuncs.com"
					]
					}
				}
			],
				"Version": "1"
			}`

			_, err := client.CreateRole(request)
			if err != nil {
				return fmt.Errorf("Failed to create RAM role: %v", err)
			}
		}
	}

	// Attach AliyunECSImageImportRolePolicy if not exist
	{
		request := ram.CreateListPoliciesForRoleRequest()
		request.Scheme = "https"
		request.RoleName = "AliyunECSImageImportDefaultRole"
		response, err := client.ListPoliciesForRole(request)
		if err != nil {
			return fmt.Errorf("Failed to list policies for role: %v", err)
		}
		for _, policy := range response.Policies.Policy {
			if policy.PolicyName == "AliyunECSImageImportRolePolicy" {
				return nil
			}
		}

		request2 := ram.CreateAttachPolicyToRoleRequest()
		request2.Scheme = "https"
		request2.PolicyType = "System"
		request2.PolicyName = "AliyunECSImageImportRolePolicy"
		request2.RoleName = "AliyunECSImageImportDefaultRole"

		_, err = client.AttachPolicyToRole(request2)
		if err != nil {
			return fmt.Errorf("Failed to attach policy to role: %v", err)
		}
	}
	return nil
}

func createECSImage(region, akid, aksecret, name, bucketname string, imagesize int, uefi, nvme bool) error {
	log.Debugf("Creating ECS image from OSS image %s", name)

	client, err := ecs.NewClientWithAccessKey(region, akid, aksecret)
	if err != nil {
		return fmt.Errorf("Failed to initilize ECS client: %v", err)
	}

	bootMode := "BIOS"
	if uefi {
		bootMode = "UEFI"
	}

	nvmeSupport := "unsupported"
	if nvme {
		nvmeSupport = "supported"
	}

	request := ecs.CreateImportImageRequest()
	request.Scheme = "https"
	request.BootMode = bootMode
	request.ImageName = name
	request.Description = "Created by linuxkit"
	request.DiskDeviceMapping = &[]ecs.ImportImageDiskDeviceMapping{
		{
			OSSObject:     name,
			OSSBucket:     bucketname,
			DiskImageSize: fmt.Sprintf("%d", imagesize),
		},
	}

	resp, err := client.ImportImage(request)
	if err != nil {
		return fmt.Errorf("Failed to create ECS image: %v", err)
	}

	request2 := ecs.CreateModifyImageAttributeRequest()
	request2.Scheme = "https"
	request2.ImageId = resp.ImageId
	request2.Features.NvmeSupport = nvmeSupport
	_, err = client.ModifyImageAttribute(request2)
	if err != nil {
		return fmt.Errorf("Failed to modify ECS image: %v", err)
	}

	return nil
}

func pushAlibabaCloudCmd() *cobra.Command {
	const (
		accessKeyIDVar     = "ALIBABA_CLOUD_ACCESS_KEY_ID"
		accessKeySecretVar = "ALIBABA_CLOUD_ACCESS_KEY_SECRET"
		regionIDVar        = "ALIBABA_CLOUD_REGION_ID"
	)
	var (
		timeoutFlag   int
		bucketFlag    string
		nameFlag      string
		uefiFlag      bool
		nvmeFlag      bool
		imagesizeFlag int
		akidFlag      string
		aksecretFlag  string
		regionFlag    string
	)
	cmd := &cobra.Command{
		Use:   "alibabacloud",
		Short: "push image to Alibaba Cloud",
		Long: `Push image to Alibaba Cloud.
		Single argument specifies the full path of an linuxkit image. It will be uploaded to OSS and an ECS image will be created from it.
		`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := args[0]

			akidFlag = getStringValue(accessKeyIDVar, akidFlag, "")
			if akidFlag == "" {
				return fmt.Errorf("Missing required flag --access-key-id, both environment variable %s and flag are not set", accessKeyIDVar)
			}

			aksecretFlag = getStringValue(accessKeySecretVar, aksecretFlag, "")
			if aksecretFlag == "" {
				return fmt.Errorf("Missing required flag --access-key-secret, both environment variable %s and flag are not set", accessKeySecretVar)
			}

			if regionFlag == "" {
				return fmt.Errorf("Missing required flag --region")
			}

			if bucketFlag == "" {
				return fmt.Errorf("Missing required flag --bucket")
			}

			if nameFlag == "" {
				nameFlag = filepath.Base(path)
			}

			err := uploadToOSS(regionFlag, akidFlag, aksecretFlag, nameFlag, path, bucketFlag, timeoutFlag)
			if err != nil {
				return err
			}

			err = configRAMPolicy(regionFlag, akidFlag, aksecretFlag)
			if err != nil {
				return err
			}

			return createECSImage(regionFlag, akidFlag, aksecretFlag, nameFlag, bucketFlag, imagesizeFlag, uefiFlag, nvmeFlag)
		},
	}

	cmd.Flags().IntVar(&timeoutFlag, "timeout", 600, "Upload timeout in seconds")
	cmd.Flags().StringVar(&nameFlag, "name", "", "Overrides the name used to identify the file in OSS and the VM image. Defaults to the base of 'path'.")
	cmd.Flags().BoolVar(&uefiFlag, "uefi", false, "Enable uefi boot mode.")
	cmd.Flags().BoolVar(&nvmeFlag, "nvme", false, "Indicate NVMe driver is supported.")
	cmd.Flags().IntVar(&imagesizeFlag, "size", 20, "Image size in GB.")
	cmd.Flags().StringVar(&akidFlag, "access-key-id", "", "Alibaba Cloud Access Key ID. Defaults to $ALIBABA_CLOUD_ACCESS_KEY_ID. *Required*")
	cmd.Flags().StringVar(&aksecretFlag, "access-key-secret", "", "Alibaba Cloud Access Key Secret. Defaults to $ALIBABA_CLOUD_ACCESS_KEY_SECRET. *Required*")
	cmd.Flags().StringVar(&regionFlag, "region-id", "", "Alibaba Cloud Region ID. Defaults to $ALIBABA_CLOUD_REGION_ID. *Required*")
	cmd.Flags().StringVar(&bucketFlag, "bucket", "", "OSS Bucket to upload to. *Required*")
	_ = cmd.MarkFlagRequired("bucket")
	_ = cmd.MarkFlagRequired("region-id")

	return cmd
}
