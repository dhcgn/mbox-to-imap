package main

import (
	"fmt"
	"os"

	"github.com/dhcgn/mbox-to-imap/config"
)

// flags
var ()

func main() {

	_, err := config.LoadConfig(os.Args[1:])
	if err != nil {
		fmt.Errorf("failed to load config: %w", err)
		os.Exit(-1)
	}

}
