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
	pflag.String("dump", "", "Dump the index from the specified file")
	pflag.Parse()

	// After parsing flags, check if --dump is specified
	dumpFile := pflag.Lookup("dump").Value.String()
	if dumpFile != "" {
		// Call DumpIndex with the specified filename
		syzgydb.DumpIndex(dumpFile)
		return
	}
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