package generator

import (
	"fmt"
	v1 "github.com/crossplane/crossplane/apis/apiextensions/v1"
	"github.com/giantswarm/crd-docs-generator/pkg/annotations"
	"github.com/giantswarm/crd-docs-generator/pkg/config"
	"github.com/giantswarm/crd-docs-generator/pkg/crd"
	"github.com/giantswarm/crd-docs-generator/pkg/git"
	"github.com/giantswarm/crd-docs-generator/pkg/output"
	"github.com/giantswarm/microerror"
	"github.com/spf13/cobra"
	"io/ioutil"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// CRDDocsGenerator represents an instance of this command line tool, it carries
// the cobra command which runs the process along with configuration parameters
// which come in as flags on the command line.
type CRDDocsGenerator struct {
	// Internals.
	RootCommand *cobra.Command

	// Settings/Preferences

	// Path to the config file
	ConfigFilePath string

	// Path to the cloned repository
	RepoFolder string

	// Path to the custom resource definitions
	CrdFolder string

	// Path to the custom resource examples
	CrFolder string

	// Path to the output folder
	OutputFolderPath string
}

const (
	// Within a git clone, relative path for upstream CRDs in YAML format.
	upstreamCRDFolder = "helm"

	// File name for bespoke upstream CRDs.
	upstreamFileName = "upstream.yaml"

	annotationsFolder = "/pkg/annotation"
)

// GenerateCrdDocs is the function called from our main CLI command.
func (crdDocsGenerator *CRDDocsGenerator) GenerateCrdDocs() error {
	configuration, err := config.Read(crdDocsGenerator.ConfigFilePath)
	if err != nil {
		return microerror.Mask(err)
	}

	// Full names of CRDs found
	crdNames := make(map[string]bool)

	// Loop over configured repositories
	// defer os.RemoveAll(crdDocsGenerator.RepoFolder)
	for _, sourceRepo := range configuration.SourceRepositories {
		// List of source YAML files containing CRD definitions.
		crdFiles := make(map[string]bool)

		log.Printf("INFO - repo %s (%s)", sourceRepo.ShortName, sourceRepo.URL)
		clonePath := crdDocsGenerator.RepoFolder + "/" + sourceRepo.Organization + "/" + sourceRepo.ShortName
		isRepository, err := git.IsRepository(clonePath)
		if !isRepository {
			log.Printf("INFO - clonePath %s - not a Git repo", clonePath)
			// Clone the repositories containing CRDs
			log.Printf("INFO - repo %s - cloning repository", sourceRepo.ShortName)
			err = git.CloneRepositoryShallow(
				sourceRepo.URL,
				sourceRepo.CommitReference,
				clonePath)
			if err != nil {
				return microerror.Mask(err)
			}
		}

		// Collect our own CRD YAML files
		thisCRDFolder := clonePath + "/" + crdDocsGenerator.CrdFolder
		err = filepath.Walk(thisCRDFolder, func(path string, info os.FileInfo, err error) error {
			if strings.HasSuffix(path, ".yaml") {
				crdFiles[path] = true
			}
			return nil
		})
		if err != nil {
			return microerror.Mask(err)
		}

		// Collect upstream CRD YAML files
		thisUpstreamCRDFolder := clonePath + "/" + upstreamCRDFolder
		err = filepath.Walk(thisUpstreamCRDFolder, func(path string, info os.FileInfo, err error) error {
			if strings.HasSuffix(path, upstreamFileName) {
				crdFiles[path] = true
			}
			return nil
		})
		if err != nil {
			return microerror.Mask(err)
		}

		// Process annotation details
		// Collect annotation info
		thisAnnotationsFolder := clonePath + "/" + annotationsFolder
		log.Printf("INFO - repo %s - collecting annotations in %s", sourceRepo.ShortName, thisAnnotationsFolder)
		repoAnnotations, err := annotations.Collect(thisAnnotationsFolder)
		if err != nil {
			log.Printf("ERROR - repo %s - collecting annotations yielded error %#v", sourceRepo.ShortName, err)
		}

		for crdFile := range crdFiles {
			log.Printf("INFO - repo %s - reading CRDs from file %s", sourceRepo.ShortName, crdFile)

			generator := crd.NewGenerator()
			resourceDefinitions, err := generator.Read(crdFile)

			if err != nil {
				log.Printf("WARN - something went wrong in crd.Read for file %s: %#v", crdFile, err)
			}

			for _, rd := range resourceDefinitions {
				switch rd.GetObjectKind().GroupVersionKind().Kind {
				default:
					log.Println("Resource kind not supported.")
					continue
				case "CompositeResourceDefinition":
					resourceDefinition := rd.(*v1.CompositeResourceDefinition)
					if _, exists := crdNames[resourceDefinition.Name]; exists {
						continue
					}

					crd, err := crd.ForCompositeResource(resourceDefinition)
					if err != nil {
						return nil
					}

					crdDocsGenerator.Write(crd, configuration, &sourceRepo, repoAnnotations)
				}
			}
		}
	}

	return nil
}

func (crdDocsGenerator *CRDDocsGenerator) Write(rd *apiextensionsv1.CustomResourceDefinition, configuration *config.FromFile, sourceRepo *config.SourceRepository, repoAnnotations []annotations.CRDAnnotationSupport) error {
	var versions []string

	for _, v := range rd.Spec.Versions {
		versions = append(versions, v.Name)
	}
	log.Printf("INFO - repo %s - processing CRD %s with versions %s", sourceRepo.ShortName, rd.Name, versions)

	// Skip hidden CRDs and CRDs with missing metadata
	meta, ok := sourceRepo.Metadata[rd.Name]
	if !ok {
		log.Printf("WARN - repo %s - skipping %s as no metadata found", sourceRepo.ShortName, rd.Name)
		return nil
	}
	if meta.Hidden {
		log.Printf("INFO - repo %s - skipping %s as hidden by configuration", sourceRepo.ShortName, rd.Name)
		return nil
	}

	// Get example CRs for this CRD (using version as key)
	exampleCRs := make(map[string]string)
	clonePath := crdDocsGenerator.RepoFolder + "/" + sourceRepo.Organization + "/" + sourceRepo.ShortName
	for _, version := range versions {
		crFileName := fmt.Sprintf("%s/%s/%s_%s_%s.yaml", clonePath, crdDocsGenerator.CrFolder, rd.Spec.Group, version, rd.Spec.Names.Singular)
		exampleCR, err := ioutil.ReadFile(crFileName)
		if err != nil {
			log.Printf("WARN - repo %s - CR example is missing for %s version %s in path %s", sourceRepo.ShortName, rd.Name, version, crFileName)
		} else {
			exampleCRs[version] = strings.TrimSpace(string(exampleCR))
		}
	}

	templatePath := path.Dir(crdDocsGenerator.ConfigFilePath) + "/" + configuration.TemplatePath
	crdAnnotations := annotations.FilterForCRD(repoAnnotations, rd.Name, "")

	if _, err := output.WritePage(
		*rd,
		crdAnnotations,
		meta,
		exampleCRs,
		crdDocsGenerator.OutputFolderPath,
		sourceRepo.URL,
		sourceRepo.CommitReference,
		templatePath); err != nil {
		log.Printf("WARN - repo %s - something went wrong in WriteCRDDocs: %#v", sourceRepo.ShortName, err)
	}

	return nil
}
