package main

import (
	"encoding/json"
	"fmt"
	"github.com/giantswarm/crd-docs-generator/pkg/generator"
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	"log"
	"os"
)

func main() {
	var err error

	var crdDocsGenerator generator.CRDDocsGenerator
	{
		c := &cobra.Command{
			Use:          "crd-docs-generator",
			Short:        "crd-docs-generator is a command line tool for generating markdown files that document Giant Swarm's custom resources",
			SilenceUsage: true,
			RunE: func(cmd *cobra.Command, args []string) error {
				return crdDocsGenerator.GenerateCrdDocs()
			},
		}

		c.PersistentFlags().StringVar(&crdDocsGenerator.ConfigFilePath, "config", "./config.yaml", "Path to the configuration file.")
		c.PersistentFlags().StringVar(&crdDocsGenerator.CrFolder, "crFolder", "docs/cr", "Path to example CRs in YAML format.")
		c.PersistentFlags().StringVar(&crdDocsGenerator.CrdFolder, "crdFolder", "config/crd", "Path to CRDs in YAML format.")
		c.PersistentFlags().StringVar(&crdDocsGenerator.OutputFolderPath, "outputFolderPath", "./output", "Path to the output folder")
		c.PersistentFlags().StringVar(&crdDocsGenerator.RepoFolder, "repoFolder", "/tmp/gitclone", "Path to the cloned repository")
		crdDocsGenerator.RootCommand = c
	}

	if err = crdDocsGenerator.RootCommand.Execute(); err != nil {
		printStackTrace(err)
		os.Exit(1)
	}
}

func printStackTrace(err error) {
	fmt.Println("\n--- Stack Trace ---")
	var stackedError microerror.JSONError
	jsonErr := json.Unmarshal([]byte(microerror.JSON(err)), &stackedError)
	if jsonErr != nil {
		fmt.Println("Error when trying to Unmarshal JSON error:")
		log.Printf("%#v", jsonErr)
		fmt.Println("\nOriginal error:")
		log.Printf("%#v", err)
	}

	for i, j := 0, len(stackedError.Stack)-1; i < j; i, j = i+1, j-1 {
		stackedError.Stack[i], stackedError.Stack[j] = stackedError.Stack[j], stackedError.Stack[i]
	}

	for _, entry := range stackedError.Stack {
		log.Printf("%s:%d", entry.File, entry.Line)
	}
}
