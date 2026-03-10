package kontroller

import (
	"fmt"
	"strings"

	"github.com/ialexeze/orkestra/domain"
	"github.com/ialexeze/orkestra/initialize"
	"github.com/ialexeze/orkestra/pkg/konfig"
	"github.com/ialexeze/orkestra/pkg/utils"
)

type BannerKonfig struct {
	Konfig     *konfig.Konfig
	Komponents []domain.Komponent
	Leader     string
	AllCRDs    []initialize.CRDEntry
}

func (c *DependencyKontroller) printBanner(b *BannerKonfig) {
	// Logo
	fmt.Print(utils.ColorCyan)
	fmt.Println(utils.Center(utils.OrkestraLogo))
	fmt.Print(utils.ColorReset)

	fmt.Println("====================================================")
	fmt.Printf("%s        Orkestra Runtime%s (v%s)\n", utils.ColorMagenta, utils.ColorReset, b.Konfig.App().Version)
	fmt.Printf("        Environment: %s%s%s\n", utils.ColorBlue, b.Konfig.App().Environment, utils.ColorReset)
	fmt.Printf("        Listening on: %s:%s%s\n", utils.ColorGreen, b.Konfig.Health().Port, utils.ColorReset)

	// if b.Leader != "" {
	//     fmt.Printf("        Leader: %s%s%s\n", utils.ColorYellow, b.Leader, utils.ColorReset)
	// } else {
	//     fmt.Printf("        Leader: %sPENDING%s\n", utils.ColorRed, utils.ColorReset)
	// }

	fmt.Println("====================================================")

	// Endpoints
	fmt.Println("Endpoints:")
	fmt.Printf("- Health:   %s/health%s\n", utils.ColorGreen, utils.ColorReset)
	fmt.Printf("- Ready:    %s/ready%s\n", utils.ColorGreen, utils.ColorReset)
	fmt.Printf("- Metrics:  %s/metrics%s\n", utils.ColorGreen, utils.ColorReset)
	fmt.Println("====================================================")

	// Komponents
	fmt.Println("Komponents:")
	for _, c := range b.Komponents {
		if c.Started() {
			fmt.Printf("- %-20s %sAVAILABLE%s\n", c.Name(), utils.ColorGreen, utils.ColorReset)
		} else {
			fmt.Printf("- %-20s %sUNAVAILABLE%s\n", c.Name(), utils.ColorRed, utils.ColorReset)
		}
	}
	fmt.Println("====================================================")

	// CRDs
	fmt.Println("CRDs:")
	for _, crd := range b.AllCRDs {
		fmt.Printf("- %s%s%s\n", utils.ColorCyan, crd.Kind, utils.ColorReset)
		fmt.Printf("  %sGroup:%s       %s\n", utils.ColorYellow, utils.ColorReset, crd.Group)
		fmt.Printf("  %sVersion:%s     %s\n", utils.ColorYellow, utils.ColorReset, crd.Version)
		if crd.Enabled {
			fmt.Printf("  %sEnabled:%s     %s\n", utils.ColorYellow, utils.ColorReset, "Yes")
		} else {
			fmt.Printf("  %sEnabled:%s     %s\n", utils.ColorYellow, utils.ColorReset, "No")
		}
		if crd.Namespace != "" {
			fmt.Printf("  %sNamespace:%s   %s\n", utils.ColorYellow, utils.ColorReset, crd.Namespace)
		}
		fmt.Printf("  %sNamespaced:%s  %v\n", utils.ColorYellow, utils.ColorReset, crd.Namespaced)
		fmt.Printf("  %sWorkers:%s     %d\n", utils.ColorYellow, utils.ColorReset, crd.Workers)

		if crd.Resync != 0 {
			fmt.Printf("  %sResync:%s      %s\n", utils.ColorYellow, utils.ColorReset, crd.Resync.String())
		} else {
			fmt.Printf("  %sResync:%s      %s(default)\n", utils.ColorYellow, utils.ColorReset, b.Konfig.Cluster().DefaultResync)
		}

		if len(crd.DependsOn) > 0 {
			fmt.Printf("  %sDependsOn:%s   %v\n", utils.ColorYellow, utils.ColorReset, strings.Join(crd.DependsOn, ", "))
		} else {
			fmt.Printf("  %sDependsOn:%s   No dependencies\n", utils.ColorYellow, utils.ColorReset)
		}

		fmt.Printf("  %sDescription:%s         %s\n", utils.ColorYellow, utils.ColorReset, crd.Description)
		fmt.Println()
	}
	fmt.Println("====================================================")

	fmt.Printf("%sOrkestra is conducting your CRDs...%s\n", utils.ColorMagenta, utils.ColorReset)
}
