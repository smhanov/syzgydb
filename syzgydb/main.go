package main

import (
	"fmt"
	"os"

	"github.com/smhanov/syzgydb"
	"github.com/spf13/pflag"
)

func main() {
	// Define the --serve flag
	pflag.Bool("serve", false, "Start the server")
	pflag.Parse()

	// Check if --serve is specified
	if pflag.Lookup("serve").Value.String() == "true" {
		// Load configuration
		if err := LoadConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Error loading configuration: %v\n", err)
			os.Exit(1)
		}

		// Start the server
		syzgydb.RunServer()
	} else {
		// Output help message
		fmt.Println("Usage:")
		pflag.PrintDefaults()
	}
}
