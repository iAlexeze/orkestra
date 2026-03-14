package internal

import (
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/pkg/utils"
)

func printBanner(kfg *orkestraKfg, leader string) {
	// Logo
	fmt.Print(utils.ColorCyan)
	fmt.Println(utils.Center(utils.OrkestraLogo))
	fmt.Print(utils.ColorReset)

	fmt.Println("====================================================")
	fmt.Printf("%s        Orkestra Runtime%s (v%s)\n", utils.ColorMagenta, utils.ColorReset, kfg.konfig.App().Version)
	fmt.Printf("        Mode: %s%s%s\n", utils.ColorCyan, strings.ToUpper(kfg.konfig.Mode()), utils.ColorReset)
	fmt.Printf("        Environment: %s%s%s\n", utils.ColorBlue, kfg.konfig.App().Environment, utils.ColorReset)
	fmt.Printf("        Listening on: %s:%s%s\n", utils.ColorGreen, kfg.konfig.Health().Port, utils.ColorReset)

	if leader != "" {
		fmt.Printf("        Leader: %s%s%s\n", utils.ColorYellow, leader, utils.ColorReset)
	} else {
		fmt.Printf("        Leader: %sPENDING%s\n", utils.ColorRed, utils.ColorReset)
	}

	fmt.Println("====================================================")

	// Endpoints
	fmt.Println("Orkestra Endpoints:")
	fmt.Printf("- Health:   %s/health%s\n", utils.ColorGreen, utils.ColorReset)
	fmt.Printf("- Ready:    %s/ready%s\n", utils.ColorGreen, utils.ColorReset)
	fmt.Printf("- Metrics:  %s/metrics%s\n", utils.ColorGreen, utils.ColorReset)

	fmt.Println()
	fmt.Println("Katalog Endpoints:")
	fmt.Printf("- Katalog:  %s/katalog%s\n", utils.ColorGreen, utils.ColorReset)
	for _, crd := range kfg.katalog.Enabled() {
		fmt.Printf("  - %s%s%s:  %s/katalog/%s%s\n", utils.ColorCyan, crd.APITypes.Kind, utils.ColorReset, utils.ColorGreen, strings.ToLower(crd.Name), utils.ColorReset)
		fmt.Printf("  - %s%s%s:  %s/katalog/%s/health%s\n", utils.ColorCyan, crd.APITypes.Kind, utils.ColorReset, utils.ColorGreen, strings.ToLower(crd.Name), utils.ColorReset)
	}
	fmt.Println("====================================================")

	// Komponents
	fmt.Println("Komponents:")
	for _, c := range *kfg.komp {
		if c.Started() {
			fmt.Printf("- %-20s %sAVAILABLE%s\n", c.Name(), utils.ColorGreen, utils.ColorReset)
		} else if c.Name() == "orkestra kontroller" {
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

		fmt.Printf("  %sGroup:%s         %s\n", utils.ColorYellow, utils.ColorReset, crd.APITypes.Group)
		fmt.Printf("  %sVersion:%s       %s\n", utils.ColorYellow, utils.ColorReset, crd.APITypes.Version)

		fmt.Printf("  %sEnabled:%s       %s\n", utils.ColorYellow, utils.ColorReset,
			map[bool]string{true: "Yes", false: "No"}[crd.Enabled],
		)

		if crd.Namespace != "" {
			fmt.Printf("  %sNamespace:%s     %s\n", utils.ColorYellow, utils.ColorReset, crd.Namespace)
		}

		fmt.Printf("  %sNamespaced:%s    %v\n", utils.ColorYellow, utils.ColorReset, crd.Namespaced)
		fmt.Printf("  %sWorkers:%s       %d\n", utils.ColorYellow, utils.ColorReset, crd.Workers)

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
				utils.ColorYellow, utils.ColorReset, strings.Join(crd.DependsOn, ", "))
		} else {
			fmt.Printf("  %sDependsOn:%s     No dependencies\n", utils.ColorYellow, utils.ColorReset)
		}

		fmt.Println()
	}
	fmt.Println("====================================================")

	fmt.Printf("%sOrkestra is konducting your CRDs...%s\n", utils.ColorMagenta, utils.ColorReset)
}
