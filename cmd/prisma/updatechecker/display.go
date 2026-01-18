package updatechecker

import (
	"fmt"
	"strings"
)

// DisplayUpdateNotification shows a formatted update notification
func DisplayUpdateNotification(info *UpdateInfo) {
	if info == nil {
		return
	}

	fmt.Println()
	fmt.Println("┌─────────────────────────────────────────────────────┐")

	// Calculate padding for version line
	versionText := fmt.Sprintf("Update available: %s → %s", info.CurrentVersion, info.LatestVersion)
	padding := 53 - len(versionText)
	fmt.Printf("│  %s%s│\n", versionText, strings.Repeat(" ", padding))

	fmt.Println("│                                                     │")
	fmt.Println("│  Run to update:                                     │")

	// Show specific version instead of @latest
	installCmd := fmt.Sprintf("go install github.com/carlosnayan/prisma-go-client/cmd/prisma@v%s", info.LatestVersion)
	fmt.Printf("│    %s", installCmd)

	// Add padding to align the right border
	cmdPadding := 49 - len(installCmd)
	fmt.Printf("%s│\n", strings.Repeat(" ", cmdPadding))

	fmt.Println("└─────────────────────────────────────────────────────┘")
	fmt.Println()
}
