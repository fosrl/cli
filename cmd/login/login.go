package login

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
)

type HostingOption string

const (
	HostingOptionCloud        HostingOption = "cloud"
	HostingOptionSelfHosted   HostingOption = "self-hosted"
)

var LoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Login to Pangolin",
	Long:  "Interactive login to select your hosting option and configure access.",
	Run: func(cmd *cobra.Command, args []string) {
		var hostingOption HostingOption
		var hostname string

		// First question: select hosting option
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[HostingOption]().
					Title("Select your hosting option").
					Options(
						huh.NewOption("Pangolin Cloud (app.pangolin.net)", HostingOptionCloud),
						huh.NewOption("Self-hosted or Dedicated instance", HostingOptionSelfHosted),
					).
					Value(&hostingOption),
			),
		)

		if err := form.Run(); err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}

		// If self-hosted, prompt for hostname
		if hostingOption == HostingOptionSelfHosted {
			hostnameForm := huh.NewForm(
				huh.NewGroup(
					huh.NewInput().
						Title("Enter hostname URL").
						Placeholder("https://your-instance.example.com").
						Value(&hostname),
				),
			)

			if err := hostnameForm.Run(); err != nil {
				fmt.Printf("Error: %v\n", err)
				return
			}
		} else {
			// For cloud, set the default hostname
			hostname = "app.pangolin.net"
		}

		// Print the result
		fmt.Printf("\nSelected hosting option: %s\n", hostingOption)
		fmt.Printf("Hostname: %s\n", hostname)
	},
}

