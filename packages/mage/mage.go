package mage

import (
	"bufio"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/template"

	"github.com/magefile/mage/mg"
	"github.com/princjef/mageutil/bintool"
	"github.com/princjef/mageutil/shellcmd"
)

//go:embed .golangci.yml
var golangci string

//go:embed .gomarkdoc.yml
var gomarkdoc string

var (
	golines = bintool.Must(bintool.NewGo(
		"github.com/segmentio/golines",
		"v0.11.0",
	))
	linter = bintool.Must(bintool.New(
		"golangci-lint{{.BinExt}}",
		"1.51.1",
		"https://github.com/golangci/golangci-lint/releases/download/v{{.Version}}/golangci-lint-{{.Version}}-{{.GOOS}}-{{.GOARCH}}{{.ArchiveExt}}",
	))
	documenter = bintool.Must(bintool.New(
		"gomarkdoc{{.BinExt}}",
		"1.1.0",
		"https://github.com/princjef/gomarkdoc/releases/download/v{{.Version}}/gomarkdoc_{{.Version}}_{{.GOOS}}_{{.GOARCH}}{{.ArchiveExt}}",
	))

	ginkgo = bintool.Must(bintool.NewGo(
		"github.com/onsi/ginkgo/v2/ginkgo",
		"v2.15.0",
		bintool.WithVersionCmd("{{.FullCmd}} version"),
	))

	gojunit = bintool.Must(bintool.New(
		"go-junit-report{{.BinExt}}",
		"v2.0.0",
		"https://github.com/jstemmer/go-junit-report/releases/download/{{.Version}}/go-junit-report-{{.Version}}-{{.GOOS}}-{{.GOARCH}}{{.ArchiveExt}}",
	))
)

const (
	exludePackagesManifest = "exclude-from-code-coverage.txt"
)

func ensureFormatter() error {
	return golines.Ensure()
}

func ensureLinter() error {
	return linter.Ensure()
}

func ensureDocumenter() error {
	return documenter.Ensure()
}

func ensureGinkgo() error {
	return ginkgo.Ensure()
}

func ensureGoJUnit() error {
	return gojunit.Ensure()
}

// EnsureAllTools checks to see if a valid version of the needed tools are
// installed, and downloads/installs them if not.
func EnsureAllTools() error {
	mg.Deps(ensureFormatter, ensureLinter, ensureDocumenter, ensureGinkgo, ensureGoJUnit)

	return nil
}

// Format formats the code.
func Format() error {
	if err := ensureFormatter(); err != nil {
		return err
	}

	return golines.Command("-m 80 --no-reformat-tags --base-formatter gofmt -w .").
		Run()
}

// Lint lints the code.
func Lint() error {
	if err := ensureLinter(); err != nil {
		return err
	}

	close, err := TmpFile(".golangci.yml", golangci)
	if err != nil {
		return err
	}
	defer close()

	return linter.Command("run").Run()
}

// Doc generates documents for the code.
func Doc() error {
	if err := ensureDocumenter(); err != nil {
		return err
	}

	close, err := docCfg()
	if err != nil {
		return err
	}
	defer close()

	return shellcmd.RunAll(
		documenter.Command("./..."),
		// Remove internal READMEs to prevent unnecessary thrashing.
		// TODO: See if this can be built into gomarkdoc.
		`find . -path '*/internal/*README.md' -exec rm {} +`,
		`find . -path '*/proto/*README.md' -exec rm {} +`,
	)
}

// Create a temporary gomarkdoc config with the current path.
func docCfg() (func(), error) {
	path, err := Gitwd()
	if err != nil {
		return nil, err
	}

	t, err := template.New("gomarkdoc").Delims("<<", ">>").Parse(gomarkdoc)
	if err != nil {
		return nil, err
	}

	var data strings.Builder
	if err := t.Execute(&data, path); err != nil {
		return nil, err
	}

	return TmpFile(".gomarkdoc.yml", data.String())
}

func raceCheckSkipped() bool {
	return os.Getenv("SKIP_RACE_CHECK") == "true"
}

func raceOpt() string {
	if raceCheckSkipped() {
		return ""
	}
	return "-race"
}

// Test runs the unit tests.
func Test() error {
	return shellcmd.Command(fmt.Sprintf(`go test %s -timeout 5m -cover -coverprofile=coverage.out ./...`, raceOpt())).Run()
}

// TestRace runs unit tests without the test cache.
// TODO: remove once integration tests no longer reference this.
func TestRace() error {
	return shellcmd.RunAll(
		`go clean -testcache`,
		`go test -race -timeout 5m -cover -coverprofile=coverage.out ./...`,
	)
}

// CleanTest runs unit tests without the test cache.
func CleanTest() error {
	return shellcmd.RunAll(
		`go clean -testcache`,
		shellcmd.Command(fmt.Sprintf(`LD_LIBRARY=./pkg/apis/v1alpha1/providers/target/rust/target/x86_64-unknown-linux-gnu/release CGO_LDFLAGS=-L./pkg/apis/v1alpha1/providers/target/rust/target/x86_64-unknown-linux-gnu/release go test %s -timeout 5m -cover -coverprofile=coverage.out ./...`, raceOpt())),
	)
}

// Retrieve the test coverage count from coverage.out file.
func PrintCoverage() error {
	file := "coverage.out"

	// check if coverage file exists
	_, err := os.Stat(file)
	if err != nil {
		// throw error if coverage file does not exist
		return fmt.Errorf("coverage file (%s) does not exist", file)
	}
	// print test coverage count
	return shellExec(fmt.Sprintf(`go tool cover -func=%s | grep total: | grep -Eo '[0-9]+\.[0-9]+'`, file), false)
}

// Cover checks code coverage from unit tests.
func Cover(file string) error {
	return shellcmd.RunAll(
		`go test -coverprofile=coverage.out -coverpkg="./..." ./...`,
		shellcmd.Command(
			fmt.Sprintf(`go tool cover -html=coverage.out -o="%s"`, file),
		),
	)
}

// Test runs both unit and suite tests.
func RunUnitTestAndSuiteTest() error {
	mg.SerialDeps(UnitTest, SuiteTest)
	return nil
}

// UnitTest runs the unit tests.
func UnitTest() error {
	mg.Deps(Clean)
	bld := strings.Builder{}
	os.Setenv("GOUNIT", "true")
	defer os.Unsetenv("GOUNIT")
	bld.WriteString(fmt.Sprintf("go test -v -cover -coverprofile=coverage.out %s -timeout 5m ./...", raceOpt()))
	if isCI() {
		mg.Deps(ensureGoJUnit)
		bld.WriteString(" 2>&1 | bin/go-junit-report -set-exit-code -iocopy -out junit-unit-tests.xml")
	}
	err := shellExec(bld.String(), true)
	if err != nil {
		return err
	}
	// Hack to remove unused packages from code coverage
	// until we purge them from the codebase.
	_, err = os.Stat(exludePackagesManifest)
	if err != nil && os.IsNotExist(err) {
		return nil
	}
	err = deleteLinesFromCoverage("coverage.out", exludePackagesManifest)
	if err != nil {
		return err
	}
	return nil
}

// SuiteTest runs the suite tests.
func SuiteTest() error {
	mg.Deps(Clean, ensureGinkgo)
	bld := strings.Builder{}
	if isCI() {
		bld.WriteString("--cover --junit-report=junit-suite-tests.xml")
	}
	return ginkgo.Command(fmt.Sprintf("%s -r", bld.String())).Run()
}

// Clean cleans the testcache
func Clean() error {
	mg.SerialDeps(
		shellcmd.Command(`go clean -testcache`).Run,
	)
	return nil
}

// deleteLinesFromCoverage deletes lines from coverage file.
func deleteLinesFromCoverage(coverageFile, exclusionFileName string) error {
	exclusionFile, err := os.Open(exclusionFileName)
	if err != nil {
		return err
	}
	defer exclusionFile.Close()

	scanner := bufio.NewScanner(exclusionFile)
	for scanner.Scan() {
		line := scanner.Text()
		err = shellcmd.Command(fmt.Sprintf(`sed -i "/%s/d" %s`, line, coverageFile)).Run()
		if err != nil {
			return err
		}
	}

	return nil
}

// CI runs format, lint, doc and test.
func CI() error {
	if err := Format(); err != nil {
		return err
	}

	if err := Lint(); err != nil {
		return err
	}

	// if err := Doc(); err != nil {
	// 	return err
	// }

	if err := Test(); err != nil {
		return err
	}

	return nil
}

// CIVerify checks if format, lint, doc and test were ran.
func CIVerify() error {
	if err := Format(); err != nil {
		return err
	}

	if err := Lint(); err != nil {
		return err
	}

	// TODO: DocVerify does not work with manual internal removal.

	if err := Test(); err != nil {
		return err
	}

	return nil
}

// ensureBuildxBuilder ensures that buildx is created and set up.
func ensureBuildxBuilder() error {
	checkCmd := exec.Command("docker", "buildx", "ls")
	output, err := checkCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to list buildx builders: %v, output: %s", err, output)
	}
	if !strings.Contains(string(output), "default") {
		createCmd := exec.Command("docker", "buildx", "create", "--use", "--name", "default")
		createOutput, err := createCmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to create buildx builder: %v, output: %s", err, createOutput)
		}
		fmt.Println("Created buildx builder:", string(createOutput))
	} else {
		fmt.Println("Buildx builder 'default' already exists.")
	}
	return nil
}

// buildAndPushDockerImage builds and pushes the Docker image for all specified platforms.
func buildAndPushDockerImage(dockerFile, buildContext, imageTag string) error {
	if err := ensureBuildxBuilder(); err != nil {
		return err
	}

	platforms := "linux/amd64,linux/arm64,linux/arm/v7"
	buildCmd := exec.Command("docker", "buildx", "build", "--platform", platforms, "-f", dockerFile, "-t", imageTag, buildContext, "--push")
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return fmt.Errorf("failed to build and push image with Dockerfile %s: %v, output: %s", dockerFile, err, output)
	}
	fmt.Println("Build and push output:", string(output)) // Debugging line

	return nil
}

// buildWithDockerCompose builds the Docker image using Docker Compose.
func buildWithDockerCompose(composeFile string) error {
	buildCmd := exec.Command("docker", "compose", "-f", composeFile, "build")
	output, err := buildCmd.CombinedOutput()
	if err != nil {
		fmt.Println(string(output))
		return fmt.Errorf("failed to build with Docker Compose file %s: %v, output: %s", composeFile, err, output)
	}
	fmt.Println("Docker Compose build output:", string(output)) // Debugging line

	return nil
}

// Build docker image with docker compose.
func DockerBuild() error {
	return buildWithDockerCompose("docker-compose.yaml")
}

// Build Symphony API multi-platfrom docker image
func DockerBuildAPIMultiPlatform() error {
	return buildAndPushDockerImage("Dockerfile", "..", "ghcr.io/eclipse-symphony/symphony-api-multi:latest")
}

// Build Symphony K8s multi-platfrom docker image
func DockerBuildK8sMultiPlatform() error {
	return buildAndPushDockerImage("Dockerfile", "..", "ghcr.io/eclipse-symphony/symphony-k8s-multi:latest")
}

// Build Symphony target agent multi-platfrom docker image
func DockerBuildTargetAgentMultiPlatform() error {
	return buildAndPushDockerImage("Dockerfile.target-agent", "..", "ghcr.io/eclipse-symphony/symphony-target-agent-multi:latest")
}
func DockerBuildTargetAgent() error {
	return buildWithDockerCompose("docker-compose-target-agent.yaml")
}

// Build Symphony poll agent multi-platfrom docker image
func DockerBuildPollAgentMultiPlatform() error {
	return buildAndPushDockerImage("Dockerfile.target-agent", "..", "ghcr.io/eclipse-symphony/symphony-poll-agent-multi:latest")
}
func DockerBuildPollAgent() error {
	return buildWithDockerCompose("docker-compose-poll-agent.yaml")
}

func DockerBuildWithOverrideImg(buildBaseImg string, targetBaseImg string) error {
	var arg string
	if buildBaseImg != "" {
		arg += fmt.Sprintf(" --build-arg BUILD_BASE_IMAGE=%s", buildBaseImg)
	}
	if targetBaseImg != "" {
		arg += fmt.Sprintf(" --build-arg TARGET_BASE_IMAGE=%s", targetBaseImg)
	}

	return shellcmd.Command(fmt.Sprintf("docker-compose -f docker-compose.yaml build %s", arg)).Run()
}

// Run a command with | or other things that do not work in shellcmd
func shellExec(cmd string, printCmdOrNot bool) error {
	if printCmdOrNot {
		fmt.Println(">", cmd)
	}

	execCmd := exec.Command("sh", "-c", cmd)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr

	return execCmd.Run()
}

// isCI returns true if running in CI.
func isCI() bool {
	_, ok := os.LookupEnv("BUILD_BUILDID") // rudimentary check for Azure DevOps
	return ok
}
