// +build integration

package tests

import (
	"os"
	"path/filepath"
	"testing"

	"get.porter.sh/porter/pkg/porter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstall_relativePathPorterHome(t *testing.T) {
	p := porter.NewTestPorter(t)
	p.SetupIntegrationTest() // This creates a temp porter home directory
	defer p.CleanupIntegrationTest()
	p.Debug = false

	// Crux for this test: change Porter's home dir to a relative path
	homeDir, err := p.Config.GetHomeDir()
	require.NoError(t, err)
	curDir, err := os.Getwd()
	require.NoError(t, err)
	relDir, err := filepath.Rel(curDir, homeDir)
	require.NoError(t, err)
	p.SetHomeDir(relDir)

	// Bring in a porter manifest that has an install action defined
	p.TestConfig.TestContext.AddTestFile(filepath.Join(p.TestDir, "testdata/bundle-with-custom-action.yaml"), "porter.yaml")
	p.TestConfig.TestContext.AddTestFile(filepath.Join(p.TestDir, "testdata/helpers.sh"), "helpers.sh")

	installOpts := porter.InstallOptions{}
	err = installOpts.Validate([]string{}, p.Porter)
	require.NoError(t, err)

	// Install the bundle, assert no error occurs due to Porter home as relative path
	err = p.InstallBundle(installOpts)
	require.NoError(t, err)
}

func TestInstall_fileParam(t *testing.T) {
	p := porter.NewTestPorter(t)
	p.SetupIntegrationTest()
	defer p.CleanupIntegrationTest()
	p.Debug = false

	p.TestConfig.TestContext.AddTestFile(filepath.Join(p.TestDir, "testdata/bundle-with-file-params.yaml"), "porter.yaml")
	p.TestConfig.TestContext.AddTestFile(filepath.Join(p.TestDir, "testdata/helpers.sh"), "helpers.sh")
	p.TestConfig.TestContext.AddTestFile(filepath.Join(p.TestDir, "testdata/myfile"), "./myfile")
	p.TestConfig.TestContext.AddTestFile(filepath.Join(p.TestDir, "testdata/myotherfile"), "./myotherfile")

	installOpts := porter.InstallOptions{}
	installOpts.Params = []string{"myfile=./myfile"}
	installOpts.ParameterSets = []string{filepath.Join(p.TestDir, "testdata/parameter-set-with-file-param.json")}

	err := installOpts.Validate([]string{}, p.Porter)
	require.NoError(t, err)

	err = p.InstallBundle(installOpts)
	require.NoError(t, err)

	// TODO: We can't check this yet because docker driver is printing directly to stdout instead of to the given writer
	// output := p.TestConfig.TestContext.GetOutput()
	// require.Contains(t, output, "Hello World!", "expected action output to contain provided file contents")

	outputs, err := p.Claims.ReadLastOutputs(p.Manifest.Name)
	require.NoError(t, err, "ReadLastOutput failed")
	myfile, ok := outputs.GetByName("myfile")
	require.True(t, ok, "expected myfile output to be persisted")
	assert.Equal(t, "Hello World!", string(myfile.Value), "expected output to match the decoded file contents")
	myotherfile, ok := outputs.GetByName("myotherfile")
	require.True(t, ok, "expected myotherfile output to be persisted")
	assert.Equal(t, "Hello Other World!", string(myotherfile.Value), "expected output 'myotherfile' to match the decoded file contents")

}

func TestInstall_withDockerignore(t *testing.T) {
	p := porter.NewTestPorter(t)
	p.SetupIntegrationTest()
	defer p.CleanupIntegrationTest()
	p.Debug = false

	p.TestConfig.TestContext.AddTestDirectory(filepath.Join(p.TestDir, "testdata/bundles/outputs-example"), ".")

	// Create .dockerignore file which ignores the helpers script
	err := p.FileSystem.WriteFile(".dockerignore", []byte("helpers.sh"), 0644)
	require.NoError(t, err)

	opts := porter.InstallOptions{}
	err = opts.Validate([]string{}, p.Porter)
	require.NoError(t, err)

	// Verify Porter uses the .dockerignore file (no helpers script added to installer image)
	err = p.InstallBundle(opts)
	// The following line would be seen from the daemon, but is printed directly to stdout:
	// Error: couldn't run command ./helpers.sh dump-config: fork/exec ./helpers.sh: no such file or directory
	// We should check this once https://github.com/cnabio/cnab-go/issues/78 is closed
	require.EqualError(t, err, "1 error occurred:\n\t* container exit code: 1, message: <nil>\n\n")
}
