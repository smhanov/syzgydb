package main

import (
	"fmt"
	"log"
	"os"

	"net/http"
	_ "net/http/pprof"

	"github.com/smhanov/syzgydb"
	"github.com/spf13/pflag"
)

func main() {
	// Start pprof server
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

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
		select {}
	} else {
		// Output help message
		fmt.Println("Usage:")
		pflag.PrintDefaults()
	}
}
