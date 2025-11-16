package main

import (
	"fmt"
	"os"

	"github.com/dhcgn/mbox-to-imap/cmd"
)

var (
	// Build information injected by ldflags during compilation
	Version   = "dev"
	CommitID  = "unknown"
	BuildTime = "unknown"
)

func main() {
	fmt.Println("mbox-to-imap version:", Version, "commit:", CommitID, "built at:", BuildTime)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}
