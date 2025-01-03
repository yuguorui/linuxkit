package main

import (
	"github.com/spf13/cobra"
)

func pushCmd() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "push",
		Short: "push a VM image to a cloud provider",
		Long:  `Push a VM image to a cloud provider.`,
	}

	// Please keep cases in alphabetical order
	cmd.AddCommand(pushAWSCmd())
	cmd.AddCommand(pushAzureCmd())
	cmd.AddCommand(pushGCPCmd())
	cmd.AddCommand(pushAlibabaCloudCmd())
	cmd.AddCommand(pushOpenstackCmd())
	cmd.AddCommand(pushEquinixMetalCmd())
	cmd.AddCommand(pushScalewayCmd())
	cmd.AddCommand(pushVCenterCmd())

	return cmd
}
