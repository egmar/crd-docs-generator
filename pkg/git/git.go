package git

import (
	"fmt"
	"os/exec"

	"github.com/giantswarm/microerror"

	errorpkg "github.com/giantswarm/crd-docs-generator/error"
)

// CloneRepositoryShallow will clone repository in a given directory.
func CloneRepositoryShallow(url string, tag string, destDir string) error {
	{
		cmd := exec.Command("git", "clone", "-b", tag, "--depth", "1", fmt.Sprintf("%s.git", url), destDir)
		err := cmd.Run()
		if err != nil {
			return microerror.Maskf(errorpkg.ExecutionError, "Could not `git clone` source repository.\nTried to execute: %s\n%s", cmd.String(), err.Error())
		}
	}

	return nil
}
