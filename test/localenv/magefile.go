//go:build mage

/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

/*
Use this tool to quickly get started developing in the symphony ecosystem. The
tool provides a set of common commands to make development easier for the team.
To get started using Minikube, run 'mage build minikube:start minikube:load deploy'.
*/
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/magefile/mage/mg"
	"github.com/princjef/mageutil/shellcmd"
)

const (
	RELEASE_NAME              = "ecosystem"
	LOCAL_HOST_URL            = "http://localhost"
	OSS_CONTAINER_REGISTRY    = "ghcr.io/eclipse-symphony"
	NAMESPACE                 = "default"
	DOCKER_TAG                = "latest"
	CHART_PATH                = "../../packages/helm/symphony"
	GITHUB_PAT                = "CR_PAT"
	LOG_ROOT                  = "/tmp/symphony-integration-test-logs"
	MINIKUBE_START_OPTIONS    = ""
	ENABLE_TLS_OTEL_SETUP     = "false"
	ENABLE_NON_TLS_OTEL_SETUP = "false"
)

var platform = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

// Print parameters for mage local testing
func PrintParams() error {
	fmt.Println("OSS_CONTAINER_REGISTRY: ", getContainerRegistry())
	fmt.Println("DOCKER_TAG: ", getDockerTag())
	fmt.Println("CHART_NAMESPACE: ", getChartNamespace())
	fmt.Println("RELEASE_NAME: ", getReleaseName())
	fmt.Println("CHART_PATH: ", getChartPath())
	fmt.Println("SKIP_GHCR_VALUES: ", skipGhcrValues())
	fmt.Println("GHCR_VALUES_OPTIONS: ", ghcrValuesOptions())
	fmt.Println("LOG_ROOT: ", getLogRoot())
	fmt.Println("MINIKUBE_START_OPTIONS: ", getMinikubeStartOptions())
	fmt.Println("ENABLE_TLS_OTEL_SETUP: ", enableTlsOtelSetup())
	fmt.Println("ENABLE_NON_TLS_OTEL_SETUP: ", enableNonTlsOtelSetup())
	return nil
}

// global variables
func enableTlsOtelSetup() bool {
	return os.Getenv("ENABLE_TLS_OTEL_SETUP") == "true"
}

func enableNonTlsOtelSetup() bool {
	return os.Getenv("ENABLE_NON_TLS_OTEL_SETUP") == "true"
}

func getLogRoot() string {
	if os.Getenv("LOG_ROOT") != "" {
		return os.Getenv("LOG_ROOT")
	} else {
		return LOG_ROOT
	}
}

func getContainerRegistry() string {
	if os.Getenv("OSS_CONTAINER_REGISTRY") != "" {
		return os.Getenv("OSS_CONTAINER_REGISTRY")
	} else {
		return OSS_CONTAINER_REGISTRY
	}
}

func getDockerTag() string {
	if os.Getenv("DOCKER_TAG") != "" {
		return os.Getenv("DOCKER_TAG")
	} else {
		return DOCKER_TAG
	}
}

func getChartNamespace() string {
	if os.Getenv("CHART_NAMESPACE") != "" {
		return os.Getenv("CHART_NAMESPACE")
	} else {
		return NAMESPACE
	}
}

func getReleaseName() string {
	if os.Getenv("RELEASE_NAME") != "" {
		return os.Getenv("RELEASE_NAME")
	} else {
		return RELEASE_NAME
	}
}

func getChartPath() string {
	if os.Getenv("CHART_PATH") != "" {
		return os.Getenv("CHART_PATH")
	} else {
		return CHART_PATH
	}
}

func skipGhcrValues() bool {
	return os.Getenv("SKIP_GHCR_VALUES") == "true"
}

func ghcrValuesOptions() string {
	if skipGhcrValues() {
		return ""
	}
	if enableTlsOtelSetup() {
		return "-f symphony-ghcr-values.otel.yaml --skip-crds"
	} else if enableNonTlsOtelSetup() {
		return "-f symphony-ghcr-values.otel.non-tls.yaml"
	} else {
		return "-f symphony-ghcr-values.yaml"
	}
}

func getMinikubeStartOptions() string {
	if os.Getenv("MINIKUBE_START_OPTIONS") != "" {
		return os.Getenv("MINIKUBE_START_OPTIONS")
	} else {
		return MINIKUBE_START_OPTIONS
	}
}

var reWhiteSpace = regexp.MustCompile(`\n|\t| `)

type Minikube mg.Namespace
type Build mg.Namespace
type Pull mg.Namespace
type Cluster mg.Namespace
type Test mg.Namespace
type License mg.Namespace

/******************** Targets ********************/

// Deploys the symphony ecosystem to your local Minikube cluster.
func (Cluster) Deploy() error {
	fmt.Printf("Deploying symphony to minikube\n")
	mg.Deps(ensureMinikubeUp)

	if enableTlsOtelSetup() {
		err := ensureSecureOtelCollectorPrereqs()
		if err != nil {
			return err
		}
	}

	certsToVerify := []string{"symphony-api-serving-cert ", "symphony-serving-cert"}
	commands := []shellcmd.Command{
		shellcmd.Command(fmt.Sprintf("helm upgrade %s %s --install -n %s --create-namespace --wait -f ../../packages/helm/symphony/values.yaml %s --set symphonyImage.tag=%s --set paiImage.tag=%s", getReleaseName(), getChartPath(), getChartNamespace(), ghcrValuesOptions(), getDockerTag(), getDockerTag())),
	}
	for _, cert := range certsToVerify {
		commands = append(commands, shellcmd.Command(fmt.Sprintf("kubectl wait --for=condition=ready certificates %s -n %s --timeout=90s", cert, getChartNamespace())))
	}
	return shellcmd.RunAll(commands...)
}

// Deploys the symphony ecosystem to your local Minikube cluster with the provided settings. Note that this would also deploy cert-manager separately.
// E.g. mage deployWithSettings '--set some.key=some_value --set another.key=another_value'
func (Cluster) DeployWithSettings(values string) error {
	fmt.Printf("Deploying symphony to minikube with settings, %s\n", values)
	mg.Deps(ensureMinikubeUp)

	if enableTlsOtelSetup() {
		err := ensureSecureOtelCollectorPrereqs()
		if err != nil {
			return err
		}
	}

	certsToVerify := []string{"symphony-api-serving-cert ", "symphony-serving-cert"}
	commands := []shellcmd.Command{
		shellcmd.Command(fmt.Sprintf("helm upgrade %s %s --install -n %s --create-namespace --wait -f ../../packages/helm/symphony/values.yaml %s --set symphonyImage.tag=%s --set paiImage.tag=%s %s", getReleaseName(), getChartPath(), getChartNamespace(), ghcrValuesOptions(), getDockerTag(), getDockerTag(), values)),
	}
	for _, cert := range certsToVerify {
		commands = append(commands, shellcmd.Command(fmt.Sprintf("kubectl wait --for=condition=ready certificates %s -n %s --timeout=90s", cert, getChartNamespace())))
	}
	return shellcmd.RunAll(commands...)
}

// Up brings the minikube cluster up with symphony deployed
func Up() error {
	// Delete if a minikube cluster already exists
	mk := &Minikube{}
	_ = mk.Delete()

	c := &Cluster{}
	if err := c.Up(); err != nil {
		return err
	}

	data, err := os.ReadFile("header.txt")
	if err == nil {
		fmt.Println(string(data))
	}

	fmt.Println("Press any key to shutdown")

	reader := bufio.NewReader(os.Stdin)
	_, _, _ = reader.ReadRune()

	fmt.Println("Cleaning up minikube cluster")

	if err := mk.Delete(); err != nil {
		return err
	}

	fmt.Println("done")

	return nil
}

// PullUp pulls the latest images and starts the local environment
func PullUp() error {
	mkTask := runBg(recreateMinikube)
	p := &Pull{}

	if err := p.All(); err != nil {
		return err
	}

	if err := runBgResult(mkTask); err != nil {
		return err
	}

	if err := Up(); err != nil {
		return err
	}

	return nil
}

// Add license headers to files under relative path, e.g. mage license:addHeaders api
func (License) AddHeaders(relativePath string) error {
	// leverage https://github.com/johann-petrak/licenseheaders to add license headers
	fmt.Println("Install licenseheaders...")
	err := shellcmd.Command("pip install licenseheaders").Run()
	if err != nil {
		return err
	}

	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	licenseHeaderPath := filepath.Join(wd, "licenseheader.txt")
	data, err := os.ReadFile(licenseHeaderPath)
	if err == nil {
		fmt.Println("----------------License header----------------")
		fmt.Println(string(data))
		fmt.Println("----------------License header----------------")
	}
	repoRoot := filepath.Join(wd, "..", "..")
	fmt.Println("Changing directory to REPO Root", repoRoot)
	err = os.Chdir(repoRoot) // repo root
	if err != nil {
		return err
	}

	path := filepath.Join(repoRoot, relativePath)
	fmt.Println("Adding license header to files under", path)

	err = shellcmd.Command(fmt.Sprintf("licenseheaders -t %s -d %s --additional-extensions script=.ps1", licenseHeaderPath, path)).Run()
	if err != nil {
		return err
	}

	return nil
}

// BuildUp builds the latest images and starts the local environment
func BuildUp() error {
	mkTask := runBg(recreateMinikube)
	b := &Build{}

	if err := b.All(); err != nil {
		return err
	}

	if err := runBgResult(mkTask); err != nil {
		return err
	}

	if err := Up(); err != nil {
		return err
	}

	return nil
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

func shellExecWithoutOutput(cmd string, printCmdOrNot bool) error {
	if printCmdOrNot {
		fmt.Println(">", cmd)
	}
	execCmd := exec.Command("sh", "-c", cmd)
	execCmd.Stdout = nil
	execCmd.Stderr = nil
	return execCmd.Run()
}

// Collect Symphony logs to a log folder provided
func Logs(logRootFolder string) error {
	// api logs
	apiLogFile := fmt.Sprintf("%s/api.log", logRootFolder)
	k8sLogFile := fmt.Sprintf("%s/k8s.log", logRootFolder)
	otelCollectorLogFile := fmt.Sprintf("%s/otel-collector.log", logRootFolder)
	otelForwarderLogFile := fmt.Sprintf("%s/otel-forwarder.log", logRootFolder)

	err := shellExec(fmt.Sprintf("kubectl logs 'deployment/symphony-api' --all-containers -n %s > %s", getChartNamespace(), apiLogFile), true)
	if err != nil {
		fmt.Printf("Failed to collect api logs: %s\n", err)
	}
	err = shellExec(fmt.Sprintf("kubectl logs 'deployment/symphony-controller-manager' --all-containers -n %s > %s", getChartNamespace(), k8sLogFile), true)
	if err != nil {
		fmt.Printf("Failed to collect controller-manager logs: %s\n", err)
	}
	err = shellExecWithoutOutput(fmt.Sprintf("kubectl logs 'deployment/symphony-otel-collector' --all-containers -n %s > %s", getChartNamespace(), otelCollectorLogFile), true)
	if err != nil {
		fmt.Printf("Cannot collect otel-collector logs: %s, it's ok when otel-collector is not deployed\n", err)
	}
	err = shellExecWithoutOutput(fmt.Sprintf("kubectl logs 'ds/symphony-otel-forwarder' --all-containers -n %s > %s", getChartNamespace(), otelForwarderLogFile), true)
	if err != nil {
		fmt.Printf("Cannot to collect otel-forwarder logs: %s, it's ok when otel-forwarder is not deployed\n", err)
	}

	return nil
}

// Dump symphony api and k8s logs for tests
func DumpSymphonyLogsForTest(testName string) {
	normalizedTestName := strings.Replace(testName, "/", "_", -1)
	normalizedTestName = strings.Replace(normalizedTestName, " ", "_", -1)

	logFolderName := fmt.Sprintf("test_%s_%s", normalizedTestName, time.Now().Format("20060102150405"))
	logRootFolder := fmt.Sprintf("%s/%s", getLogRoot(), logFolderName)

	_ = shellcmd.Command(fmt.Sprintf("mkdir -p %s", logRootFolder)).Run()

	_ = Logs(logRootFolder)
}

// Uninstall all components, e.g. mage destroy all
func Destroy(flags string) error {
	err := shellcmd.RunAll(
		shellcmd.Command(fmt.Sprintf("helm uninstall %s -n %s --wait", getReleaseName(), getChartNamespace())),
	)
	if err != nil {
		return err
	}

	// to indicate if we should wait for cleanup to finish
	shouldWait := true
	for _, flag := range strings.Split(reWhiteSpace.ReplaceAllString(strings.ToLower(flags), ``), ",") {
		if flag == "nowait" {
			shouldWait = false
		}
	}

	if shouldWait {
		// Wait for all pods to go away
		if err := waitForServiceCleanup(); err != nil {
			return err
		}
	} else {
		// Wait for all pods to go away
		if err := waitForSymphonyPodsCleanup(); err != nil {
			return err
		}
	}

	return nil
}

// Build builds all containers
func (Build) All() error {
	defer logTime(time.Now(), "build:all")

	err := ensureBuildxBuilder()
	if err != nil {
		return err
	}

	err = buildRustBinding()
	if err != nil {
		return err
	}

	err = buildAPI()
	if err != nil {
		return err
	}

	err = buildK8s()
	if err != nil {
		return err
	}

	err = buildAgent()
	if err != nil {
		return err
	}

	return nil
}

// Store the docker images to tar files
func (Build) Save() error {
	defer logTime(time.Now(), "build:save")

	k8s_tar_file := fmt.Sprintf("symphony-k8s:%s.tar", getDockerTag())
	api_tar_file := fmt.Sprintf("symphony-api:%s.tar", getDockerTag())
	k8s_image_tag := fmt.Sprintf("%s/symphony-k8s:%s", getContainerRegistry(), getDockerTag())
	api_image_tag := fmt.Sprintf("%s/symphony-api:%s", getContainerRegistry(), getDockerTag())
	err := saveDockerImageToTarFile(k8s_tar_file, k8s_image_tag)
	if err != nil {
		return err
	}

	err = saveDockerImageToTarFile(api_tar_file, api_image_tag)
	if err != nil {
		return err
	}

	return nil
}

func saveDockerImageToTarFile(tarFilePath string, imageTag string) error {
	return shellcmd.Command(fmt.Sprintf("docker image save -o %s %s", tarFilePath, imageTag)).Run()
}

func (Build) RustBinding() error {
	return buildRustBinding()
}
func buildRustBinding() error {
	return shellcmd.Command("cargo build --release --manifest-path ../../api/pkg/apis/v1alpha1/providers/target/rust/Cargo.toml").Run()
}

// Build api container
func (Build) Api() error {
	return buildAPI()
}
func buildAPI() error {
	imageName := "ghcr.io/eclipse-symphony/symphony-api"
	return shellcmd.Command(fmt.Sprintf("docker buildx build --platform %s -f ../../api/Dockerfile -t %s \"../..\" --load", platform, imageName)).Run() //oss
}

func (Build) ApiAzure() error {
	return buildAPIAzure()
}
func buildAPIAzure() error {
	imageName := "ghcr.io/eclipse-symphony/symphony-api"
	return shellcmd.Command(fmt.Sprintf("docker buildx build --platform %s -f ../../api/Dockerfile -t %s --build-arg BUILDFLAG=azure \"../..\" --load", platform, imageName)).Run() //oss
}

func (Build) ApiFault() error {
	return buildAPIFault()
}

func buildAPIFault() error {
	imageName := "ghcr.io/eclipse-symphony/symphony-api"
	return shellcmd.Command(fmt.Sprintf("docker buildx build --platform %s -f ../../api/Dockerfile -t %s --build-arg FAULT_INJECTION_ENABLED=true \"../..\" --load", platform, imageName)).Run() //oss
}

func buildAgent() error {
	pollAgentImageName := "ghcr.io/eclipse-symphony/symphony-poll-agent"
	targetAgentImageName := "ghcr.io/eclipse-symphony/symphony-target-agent"
	return shellcmd.RunAll(
		shellcmd.Command(fmt.Sprintf("docker buildx build --platform %s -f ../../api/Dockerfile.poll-agent -t %s \"../..\" --load", platform, pollAgentImageName)),
		shellcmd.Command(fmt.Sprintf("docker buildx build --platform %s -f ../../api/Dockerfile.target-agent -t %s \"../..\" --load", platform, targetAgentImageName)),
	) //oss
}

// Build k8s container
func (Build) K8s() error {
	return buildK8s()
}
func buildK8s() error {
	imageName := "ghcr.io/eclipse-symphony/symphony-k8s"
	return shellcmd.Command(fmt.Sprintf("docker buildx build --platform %s -f ../../k8s/Dockerfile -t %s \"../..\" --load", platform, imageName)).Run() //oss
}

func (Build) K8sAzure() error {
	return buildK8sAzure()
}
func buildK8sAzure() error {
	imageName := "ghcr.io/eclipse-symphony/symphony-k8s"
	return shellcmd.Command(fmt.Sprintf("docker buildx build --platform %s -f ../../k8s/Dockerfile -t %s --build-arg BUILDFLAG=azure \"../..\" --load", platform, imageName)).Run() //oss
}

func (Build) K8sFault() error {
	return buildK8sFault()
}
func buildK8sFault() error {
	// Pass fault arguments if required
	imageName := "ghcr.io/eclipse-symphony/symphony-k8s"
	return shellcmd.Command(fmt.Sprintf("docker buildx build --platform %s -f ../../k8s/Dockerfile -t %s  --build-arg FAULT_INJECTION_ENABLED=true \"../..\" --load", platform, imageName)).Run() //oss
}

/******************** Minikube ********************/

// Installs the Minikube binary on your machine.
func (Minikube) Install() error {
	whereIsMinikube, err := shellcmd.Command("whereis minikube").Output()
	if err != nil {
		return err
	}

	// Normalize 'whereis' command output to identify if Minikube is installed
	if reWhiteSpace.ReplaceAllString(string(whereIsMinikube), ``) != "minikube:" {
		return shellcmd.Command("minikube version").Run()
	}

	err = shellcmd.Command(`curl -o "minikube" -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64`).Run()
	if err != nil {
		return err
	}

	err = shellcmd.Command(`sudo install "minikube" /usr/local/bin/minikube`).Run()
	if err != nil {
		return err
	}

	err = shellcmd.Command(`rm minikube`).Run()
	if err != nil {
		return err
	}

	return nil
}

// Starts the Minikube cluster w/ select addons.
func (Minikube) Start() error {
	err := shellcmd.Command(fmt.Sprintf("minikube start %s", getMinikubeStartOptions())).Run()
	if err != nil {
		return err
	}

	err = shellcmd.Command("minikube addons enable metrics-server").Run()
	if err != nil {
		return err
	}

	return nil
}

// Stops the Minikube cluster.
func (Minikube) Stop() error {
	return shellcmd.Command("minikube stop").Run()
}

// Loads symphony component docker images onto the Minikube VM.
func (Minikube) Load() error {
	return shellcmd.RunAll(load(
		fmt.Sprintf("symphony-api:%s", getDockerTag()),
		fmt.Sprintf("symphony-k8s:%s", getDockerTag()))...)
}

// Deletes the Minikube cluster from you dev box.
func (Minikube) Delete() error {
	return shellcmd.Command("minikube delete").Run()
}

// Brings the cluster up with all images loaded
func (Cluster) Load() error {
	if err := ensureMinikubeUp(); err != nil {
		return err
	}

	mk := &Minikube{}
	if err := mk.Load(); err != nil {
		return err
	}

	return nil
}

// Brings the cluster up, loads the image and deploys
func (Cluster) Up() error {
	defer logTime(time.Now(), "cluster:up")

	// Install minikube
	c := &Cluster{}
	if err := c.Load(); err != nil {
		return err
	}

	if err := c.Deploy(); err != nil {
		return err
	}

	return nil
}

// Stop the cluster
func (Cluster) Down() error {
	mk := &Minikube{}
	return mk.Stop()
}

// Deploys the symphony ecosystem to minikube and waits for all pods to be ready.
// This is intended for use with the automated integration tests.
// Dev workflows can use more optimized commands.
func (Test) Up() error {
	defer logTime(time.Now(), "test:up")

	// Show the state of the cluster for CI scenarios
	// This should be shown even when an error occurs
	c := &Cluster{}
	defer c.Status()

	// Delete if a minikube cluster already exists
	mk := &Minikube{}
	_ = mk.Delete()

	// Build and load images without deploying
	// tests will run the deployment
	return c.Up()
}

// Show the state of the cluster for CI scenarios
func (Cluster) Status() {
	fmt.Println("*******************[Cluster]**********************")
	shellcmd.Command("helm list --all").Run()
	shellcmd.Command("kubectl get pods -A -o wide").Run()
	shellcmd.Command("kubectl get deployments -A -o wide").Run()
	shellcmd.Command("kubectl get services -A -o wide").Run()
	shellcmd.Command("kubectl top pod -A").Run()
	shellcmd.Command("kubectl get events -A").Run()

	fmt.Println("Describing failed pods")
	dumpShellOutput(fmt.Sprintf("kubectl get pods --all-namespaces | grep -E 'CrashLoopBackOff|Error|ImagePullBackOff|InvalidImageName|Pending' | awk '{print $2}' | xargs -I {} kubectl describe pod {} -n %s", getChartNamespace()))
	dumpShellOutput(fmt.Sprintf("kubectl get pods --all-namespaces | grep -E 'CrashLoopBackOff|Error|ImagePullBackOff|InvalidImageName|Pending' | awk '{print $2}' | xargs -I {} kubectl logs {} -n %s", getChartNamespace()))
	fmt.Println("**************************************************")
}

// Launch the Minikube Kubernetes dashboard.
func (Minikube) Dashboard() error {
	return shellcmd.Command("minikube dashboard").Run()
}

// Pulls all docker images for symphony
func (Pull) All() error {
	defer logTime(time.Now(), "pull:all")

	// Pull directly from ACR
	return shellcmd.RunAll(pull(
		"symphony-k8s",
		"symphony-api",
	)...)
}

// Pull symphony-k8s
func (Pull) K8s() error {
	// Pull directly from CR
	return shellcmd.RunAll(pull(
		"symphony-k8s",
	)...)
}

// Pull symphony-api
func (Pull) Api() error {
	// Pull directly from CR
	return shellcmd.RunAll(pull(
		"symphony-api",
	)...)
}

// Log into ghcr, prompts if login failed.
func GhcrLogin() error {
	for i := 0; i < 3; i++ {
		github_pat := os.Getenv(GITHUB_PAT)
		if github_pat == "" {
			fmt.Println("Please input your GitHub PAT token:")
			fmt.Scanln(&github_pat)
			os.Setenv(GITHUB_PAT, github_pat)
		}
		err := shellcmd.RunAll(shellcmd.Command(fmt.Sprintf("docker login ghcr.io -u USERNAME --password %s", github_pat)))
		if err != nil {
			if i == 3 {
				return err
			}
		} else {
			return nil
		}
	}

	return nil
}

// Remove Symphony resource
func Remove(resourceType, resourceName string) error {
	fmt.Printf("Deleting resource %s %s\n", resourceType, resourceName)
	err := shellcmd.RunAll(shellcmd.Command(fmt.Sprintf("kubectl delete %s %s", resourceType, resourceName)))
	if err != nil {
		return err
	}

	return nil
}

/******************** Helpers ********************/
func browserOpen(url string) error {
	openBrowser := fmt.Sprintf("xdg-open %s", url)
	return shellcmd.Command(openBrowser).Run()
}

// runParallel parallelizes running the commands
// this will print out all errors and return only the last error
func runParallel(commands ...shellcmd.Command) error {
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(commands))

	// latest error seen
	var finalErr error
	for _, cmd := range commands {
		go func(cmd shellcmd.Command) {
			defer waitGroup.Done()
			start := time.Now()

			fmt.Printf("[START] '%s'\n", cmd)

			if err := cmd.Run(); err != nil {
				finalErr = err
				fmt.Println(err)
			}

			fmt.Printf("[DONE] (%s) '%s'\n", time.Since(start), cmd)
		}(cmd)
	}

	waitGroup.Wait()
	return finalErr
}

func load(names ...string) []shellcmd.Command {
	loads := make([]shellcmd.Command, len(names))
	for i, name := range names {
		loads[i] = shellcmd.Command(fmt.Sprintf(
			"minikube image load %s/%s",
			getContainerRegistry(),
			name,
		))
	}

	return loads
}

func pull(names ...string) []shellcmd.Command {
	loads := make([]shellcmd.Command, len(names))

	for i, name := range names {
		loads[i] = shellcmd.Command(fmt.Sprintf(
			"docker pull %s/%s",
			getContainerRegistry(),
			name,
		))
	}

	return loads
}

// Run a command with | or other things that do not work in shellcmd
func dumpShellOutput(cmd string) error {
	fmt.Println("> ", cmd)

	b, err := exec.Command("sh", "-c", cmd).CombinedOutput()
	if err != nil {
		fmt.Println("failed to run command", err)
		return err
	} else {
		fmt.Println(string(b))
	}

	return nil
}

// Wait for symphony pods to be cleaned up
func waitForSymphonyPodsCleanup() error {
	var startTime = time.Now()
	c := &Cluster{}

	fmt.Println("Waiting for all pods to go away...")

	loopCount := 0

	for {
		loopCount++
		if loopCount == 600 {
			return fmt.Errorf("Failed to clean up all the resources!")
		}

		o, err := shellcmd.Command.Output(`kubectl get pods -A --output=jsonpath='{range .items[*]}{@.metadata.namespace}{"|"}{@.metadata.name}{"\n"}{end}'`)
		if err != nil {
			return err
		}

		pods := strings.Split(strings.TrimSpace(string(o)), "\n")
		notReady := make([]string, 0)

		for _, pod := range pods {
			parts := strings.Split(pod, "|")
			pod = parts[1]
			namespace := parts[0]
			if namespace != "kube-system" && namespace != "cert-manager" {
				if strings.Contains(pod, "symphony") {
					notReady = append(notReady, pod)
				}
			}
		}

		if len(notReady) > 0 {
			// Show pods that aren't ready
			if loopCount%30 == 0 {
				fmt.Printf("waiting for pod removal. Try: %d Not ready: %s\n", loopCount, strings.Join(notReady, ", "))
			}

			// Show complete status every 5 minutes to help debug
			if loopCount%300 == 0 {
				c.Status()
			}

			time.Sleep(time.Second)
		} else {
			fmt.Println("All pods are cleaned up: ", time.Since(startTime).String())
			return nil
		}

		time.Sleep(time.Second)
	}
}

// Wait for cleanup to finish
func waitForServiceCleanup() error {
	var startTime = time.Now()
	c := &Cluster{}

	fmt.Println("Waiting for all pods to go away...")

	loopCount := 0

	for {
		loopCount++
		if loopCount == 600 {
			return fmt.Errorf("Failed to clean up all the resources!")
		}

		o, err := shellcmd.Command.Output(`kubectl get pods -A --output=jsonpath='{range .items[*]}{@.metadata.namespace}{"|"}{@.metadata.name}{"\n"}{end}'`)
		if err != nil {
			return err
		}

		pods := strings.Split(strings.TrimSpace(string(o)), "\n")
		notReady := make([]string, 0)

		for _, pod := range pods {
			parts := strings.Split(pod, "|")
			pod = parts[1]
			namespace := parts[0]
			if namespace != "kube-system" && namespace != "cert-manager" {
				notReady = append(notReady, pod)
			}
		}

		if len(notReady) > 0 {
			// Show pods that aren't ready
			if loopCount%30 == 0 {
				fmt.Printf("waiting for pod removal. Try: %d Not ready: %s\n", loopCount, strings.Join(notReady, ", "))
			}

			// Show complete status every 5 minutes to help debug
			if loopCount%300 == 0 {
				c.Status()
			}

			time.Sleep(time.Second)
		} else {
			fmt.Println("All pods are cleaned up: ", time.Since(startTime).String())
			return nil
		}

		time.Sleep(time.Second)
	}
}

// Run a command in the background
func runBg(f func() error) <-chan error {
	errChan := make(chan error, 1)

	go func() {
		defer close(errChan)

		if err := f(); err != nil {
			errChan <- err
		}
	}()

	return errChan
}

// Wait for an error or the channel to close
func runBgResult(errChan <-chan error) error {
	if errChan != nil {
		err, ok := <-errChan
		if !ok {
			return nil
		}
		if err != nil {
			return err
		}
	}
	return nil
}

// Delete and recreate minikube
func recreateMinikube() error {
	defer logTime(time.Now(), "recreate minikube")

	mk := &Minikube{}
	_ = mk.Delete()

	return ensureMinikubeUp()
}

func ensureSecureOtelCollectorPrereqs() error {
	fmt.Println("Deploying OSS cert-manager for otel-collector")
	err := shellcmd.Command("kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/v1.15.3/cert-manager.yaml --wait").Run()
	if err != nil {
		return err
	}

	// Path to the wait script
	fmt.Println("Waiting for cert-manager webhook to be ready")
	waitCmds := []shellcmd.Command{
		shellcmd.Command("kubectl wait --for=condition=ready pod -l app.kubernetes.io/component=webhook -n cert-manager --timeout=90s"),
	}
	err = shellcmd.RunAll(waitCmds...)
	if err != nil {
		fmt.Println("Try second time after 30 seconds...")
		time.Sleep(30 * time.Second)
		err = shellcmd.RunAll(waitCmds...)
		if err != nil {
			return err
		}
	}

	fmt.Println("Deploying OSS trust-manager for otel-collector")
	err = shellcmd.Command("helm repo add jetstack https://charts.jetstack.io --force-update").Run()
	if err != nil {
		return err
	}

	err = shellcmd.Command("helm upgrade trust-manager jetstack/trust-manager --install --namespace cert-manager --wait").Run()
	if err != nil {
		return err
	}

	fmt.Println("Preparing certificates for otel-collector")

	// replace the dns name and common name in 3.tls-cert.yaml
	fmt.Println("Replacing the dns name and common name in 3.tls-cert.yaml")
	err = shellcmd.Command(fmt.Sprintf("sed -i 's/symphony-otel-collector-service\\..*\\.svc\\.cluster\\.local/symphony-otel-collector-service.%s.svc.cluster.local/g' ./otel-certificates/3.tls-cert.yaml", getChartNamespace())).Run()

	if err != nil {
		return err
	}

	fmt.Printf("Creating namespace %s\n", getChartNamespace())
	shellcmd.Command(fmt.Sprintf("kubectl create ns %s", getChartNamespace())).Run()

	cmds := []shellcmd.Command{
		shellcmd.Command(fmt.Sprintf("kubectl apply -f ./otel-certificates/0.selfsigned-issuer.yaml -n %s", getChartNamespace())),
		shellcmd.Command(fmt.Sprintf("kubectl apply -f ./otel-certificates/1.root-ca.yaml")),
		shellcmd.Command(fmt.Sprintf("kubectl apply -f ./otel-certificates/2.root-ca-issuer.yaml -n %s", getChartNamespace())),
		shellcmd.Command(fmt.Sprintf("kubectl apply -f ./otel-certificates/3.tls-cert.yaml -n %s", getChartNamespace())),
		shellcmd.Command(fmt.Sprintf("kubectl apply -f ./otel-certificates/4.trust-bundle.yaml -n %s", getChartNamespace())),
	}

	return shellcmd.RunAll(cmds...)
}

// Ensure minikube is running, otherwise install and start it
func ensureMinikubeUp() error {
	defer logTime(time.Now(), "start minikube")

	if !minikubeRunning() {
		mk := &Minikube{}
		if err := mk.Install(); err != nil {
			return err
		}

		if err := mk.Start(); err != nil {
			return err
		}
	}

	if err := ensureMinikubeContext(); err != nil {
		return err
	}

	return nil
}

// True if minikube is active and running
func minikubeRunning() bool {
	b, err := shellcmd.Command.Output(`minikube status -f="{{.Host}}"`)
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(b)) == "Running"
}

// Set the kubectl context to minikube
func ensureMinikubeContext() error {
	return shellcmd.Command(`kubectl config use-context minikube`).Run()
}

func logTime(start time.Time, name string) {
	fmt.Printf("[DONE] (%s) '%s'\n", time.Since(start), name)
}

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
