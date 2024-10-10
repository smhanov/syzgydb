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

	// Define the flags
	pflag.Bool("serve", false, "Start the server")
	pflag.String("dump", "", "Dump the index from the specified file")
	pflag.String("export", "", "Export the collection from the specified file to stdout")
	pflag.String("import", "", "Import a collection from the specified JSON file")
	pflag.String("output", "", "Specify the output file for import (required with --import)")
	pflag.Parse()

	// Handle --dump flag
	dumpFile := pflag.Lookup("dump").Value.String()
	if dumpFile != "" {
		// Call DumpIndex with the specified filename
		syzgydb.DumpIndex(dumpFile)
		return
	}

	// Handle --export flag
	exportFile := pflag.Lookup("export").Value.String()
	if exportFile != "" {
		// Open the collection (assuming a function to open collection exists)
		collection, err := syzgydb.NewCollection(syzgydb.CollectionOptions{
			Name:           exportFile,        // Use the exportFile as the input file
			DistanceMethod: syzgydb.Euclidean, // or syzgydb.Cosine based on your requirement
			DimensionCount: 128,               // specify the number of dimensions
			Quantization:   64,                // specify the quantization level
			FileMode:       syzgydb.ReadOnly,  // specify the file mode
		})

		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening collection: %v\n", err)
			os.Exit(1)
		}

		// Export the collection to JSON, writing to stdout
		if err := syzgydb.ExportJSON(collection, os.Stdout); err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting collection: %v\n", err)
			os.Exit(1)
		}

		return
	}

	// Handle --import flag
	importFile := pflag.Lookup("import").Value.String()
	outputFile := pflag.Lookup("output").Value.String()
	if importFile != "" {
		if outputFile == "" {
			fmt.Fprintf(os.Stderr, "Error: --output flag is required when using --import\n")
			os.Exit(1)
		}

		// Open the JSON file
		jsonFile, err := os.Open(importFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening JSON file: %v\n", err)
			os.Exit(1)
		}
		defer jsonFile.Close()

		// Import the JSON into a new collection
		if err := syzgydb.ImportJSON(outputFile, jsonFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error importing collection: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Collection successfully imported to: %s\n", outputFile)
		return
	}

	// Handle --serve flag
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
