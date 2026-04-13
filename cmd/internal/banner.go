package internal

import (
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/pkg/utils"
	"github.com/ialexeze/orkestra/pkg/version"
)

func printBanner(kfg *orkestraKfg, konductor string) {
	// Logo
	fmt.Print(utils.ColorCyan)
	fmt.Println(utils.Center(utils.OrkestraLogo))
	fmt.Print(utils.ColorReset)

	fmt.Println("====================================================")
	fmt.Printf("%s        Orkestra Runtime%s (v%s)\n", utils.ColorMagenta, utils.ColorReset, version.Version)
	fmt.Printf("        Environment: %s%s%s\n", utils.ColorBlue, kfg.konfig.Ork().Environment, utils.ColorReset)
	fmt.Printf("        Listening on: %s:%s%s\n", utils.ColorGreen, kfg.konfig.Health().Port, utils.ColorReset)

	if konductor != "" {
		fmt.Printf("        Konductor: %s%s%s\n", utils.ColorYellow, konductor, utils.ColorReset)
	} else {
		fmt.Printf("        Konductor: %sPENDING%s\n", utils.ColorRed, utils.ColorReset)
	}

	fmt.Println("====================================================")

	// Endpoints
	fmt.Println("Orkestra Endpoints:")
	fmt.Printf("- Startup:   %s/startup%s\n", utils.ColorGreen, utils.ColorReset)
	fmt.Printf("- Health:   %s/health%s\n", utils.ColorGreen, utils.ColorReset)
	fmt.Printf("- Ready:    %s/ready%s\n", utils.ColorGreen, utils.ColorReset)
	fmt.Printf("- Metrics:  %s/metrics%s\n", utils.ColorGreen, utils.ColorReset)

	fmt.Println()
	// ENABLE_ADMISSION_WEBHOOK=true
	if kfg.katalog.HasMutationRules() || kfg.katalog.HasValidationRules() || kfg.katalog.IsDeletionProtectionEnabled() {
		fmt.Println("Webhook Endpoints:")
		if kfg.katalog.HasMutationRules() {
			fmt.Printf("- Muatation:  %s/mutate%s\n", utils.ColorGreen, utils.ColorReset)
		}
		if kfg.katalog.HasValidationRules() {
			fmt.Printf("- Validation:  %s/validate%s\n", utils.ColorGreen, utils.ColorReset)
		}
		if kfg.katalog.IsDeletionProtectionEnabled() {
			fmt.Printf("- Deletion Protection:  %s/deletion-protection%s\n", utils.ColorGreen, utils.ColorReset)
		}

		// Registration configuration
		fmt.Printf("- Service Name: %s%s%s\n", utils.ColorCyan, kfg.konfig.WebhookRegistration().ServiceName, utils.ColorReset)
		fmt.Printf("- Service Namespace: %s%s%s\n", utils.ColorCyan, kfg.konfig.WebhookRegistration().ServiceNamespace, utils.ColorReset)
		fmt.Printf("- Failure Policy: %s%s%s\n", utils.ColorCyan, kfg.konfig.WebhookRegistration().FailurePolicy, utils.ColorReset)

		fmt.Println()
	}

	// ENABLE_CONVERSION=true
	if kfg.katalog.HasConversionPaths() {
		fmt.Printf("- Conversion: %s/convert%s\n", utils.ColorGreen, utils.ColorReset)
	}

	fmt.Println()
	fmt.Println("Katalog Endpoints:")
	fmt.Printf("- Katalog:  %s/katalog%s\n", utils.ColorGreen, utils.ColorReset)
	for _, crd := range kfg.katalog.Enabled() {
		if !crd.IsEnabledAllEndpoints() {
			continue
		}
		if crd.IsInfoEnabled() {
			fmt.Printf("  - %s%s%s (%s):  %s/katalog/%s%s\n", utils.ColorCyan, crd.APITypes.Kind, utils.ColorReset, crd.Name, utils.ColorGreen, strings.ToLower(crd.Name), utils.ColorReset)
		}
		if crd.IsHealthEnabled() {
			fmt.Printf("  - %s%s%s (%s):  %s/katalog/%s/health%s\n", utils.ColorCyan, crd.APITypes.Kind, utils.ColorReset, crd.Name, utils.ColorGreen, strings.ToLower(crd.Name), utils.ColorReset)
		}
	}
	fmt.Println("====================================================")

	// Komponents
	fmt.Println("Komponents:")
	for _, c := range *kfg.komp {
		if c.Started() {
			fmt.Printf("- %-20s %sAVAILABLE%s\n", c.Name(), utils.ColorGreen, utils.ColorReset)
		} else if c.Name() == "orkestra dependency kordinator" {
			fmt.Printf("- %-20s %sSTARTING%s\n", c.Name(), utils.ColorBlue, utils.ColorReset)
		} else {
			fmt.Printf("- %-20s %sUNAVAILABLE%s\n", c.Name(), utils.ColorRed, utils.ColorReset)
		}
	}
	fmt.Println("====================================================")

	// CRDs
	fmt.Println("CRDs:")
	for _, crd := range kfg.katalog.Enabled() {
		fmt.Printf("- %s%s%s\n", utils.ColorCyan, crd.APITypes.Kind, utils.ColorReset)

		fmt.Printf("  %sName:%s          %s\n", utils.ColorYellow, utils.ColorReset, crd.Name)
		fmt.Printf("  %sGroup:%s         %s\n", utils.ColorYellow, utils.ColorReset, crd.APITypes.Group)
		fmt.Printf("  %sVersion:%s       %s\n", utils.ColorYellow, utils.ColorReset, crd.APITypes.Version)

		fmt.Printf("  %sEnabled:%s       %s\n", utils.ColorYellow, utils.ColorReset,
			map[bool]string{true: "Yes", false: "No"}[crd.IsEnabled()],
		)

		if crd.Namespace != "" {
			fmt.Printf("  %sNamespace:%s     %s\n", utils.ColorYellow, utils.ColorReset, crd.Namespace)
		}

		fmt.Printf("  %sNamespaced:%s    %v\n", utils.ColorYellow, utils.ColorReset,
			map[bool]string{true: "Yes", false: "No"}[crd.IsNamespaced()])

		// Workers
		if crd.Workers > 0 {
			fmt.Printf("  %sWorkers:%s       %d\n", utils.ColorYellow, utils.ColorReset, crd.Workers)
		} else {
			fmt.Printf("  %sWorkers:%s       %d (default)\n", utils.ColorYellow, utils.ColorReset, kfg.konfig.Cluster().DefaultWorkers)
		}

		// Queue depth
		if crd.Queue.MaxQueueDepth > 0 {
			fmt.Printf("  %sMaxQueueDepth:%s %v\n", utils.ColorYellow, utils.ColorReset, crd.Queue.MaxQueueDepth)
		} else {
			fmt.Printf("  %sMaxQueueDepth:%s %v (default)\n",
				utils.ColorYellow, utils.ColorReset, kfg.konfig.Katalog().DefaultMaxQueueDepth)
		}

		// Resync
		if crd.Resync != 0 {
			fmt.Printf("  %sResync:%s        %s\n", utils.ColorYellow, utils.ColorReset, crd.Resync.String())
		} else {
			fmt.Printf("  %sResync:%s        %s (default)\n",
				utils.ColorYellow, utils.ColorReset, kfg.konfig.Cluster().DefaultResync)
		}

		// DependsOn
		if len(crd.DependsOn) > 0 {
			fmt.Printf("  %sDependsOn:%s     %s\n",
				utils.ColorYellow, utils.ColorReset, strings.Join(crd.DependsOn.Names(), ", "))
		} else {
			fmt.Printf("  %sDependsOn:%s     No dependencies\n", utils.ColorYellow, utils.ColorReset)
		}

		fmt.Println()
	}
	fmt.Println("====================================================")

	fmt.Printf("%sOrkestra is konducting your CRDs...%s\n", utils.ColorMagenta, utils.ColorReset)
}
