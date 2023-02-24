/*
   MIT License

   Copyright (c) Microsoft Corporation.

   Permission is hereby granted, free of charge, to any person obtaining a copy
   of this software and associated documentation files (the "Software"), to deal
   in the Software without restriction, including without limitation the rights
   to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
   copies of the Software, and to permit persons to whom the Software is
   furnished to do so, subject to the following conditions:

   The above copyright notice and this permission notice shall be included in all
   copies or substantial portions of the Software.

   THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
   IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
   FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
   AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
   LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
   OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
   SOFTWARE

*/

package k8s

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/azure/symphony/coa/pkg/apis/v1alpha2"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/contexts"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/observability"
	observ_utils "github.com/azure/symphony/coa/pkg/apis/v1alpha2/observability/utils"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers"
	"github.com/azure/symphony/coa/pkg/logger"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/model"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/target/k8s/projectors"
	utils "github.com/azure/symphony/api/pkg/apis/v1alpha1/utils"
	"go.opentelemetry.io/otel/trace"
	v1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var log = logger.NewLogger("coa.runtime")

const (
	ENV_NAME     string = "SYMPHONY_AGENT_ADDRESS"
	SINGLE_POD   string = "single-pod"
	SERVICES     string = "services"
	SERVICES_NS  string = "ns-services"
	SERVICES_HNS string = "hns-services" //TODO: future versions
)

type K8sTargetProviderConfig struct {
	Name               string `json:"name"`
	ConfigType         string `json:"configType,omitempty"`
	ConfigData         string `json:"configData,omitempty"`
	Context            string `json:"context,omitempty"`
	InCluster          bool   `json:"inCluster"`
	Projector          string `json:"projector,omitempty"`
	DeploymentStrategy string `json:"deploymentStrategy,omitempty"`
}

type K8sTargetProvider struct {
	Config        K8sTargetProviderConfig
	Context       *contexts.ManagerContext
	Client        *kubernetes.Clientset
	DynamicClient dynamic.Interface
}

func K8sTargetProviderConfigFromMap(properties map[string]string) (K8sTargetProviderConfig, error) {
	ret := K8sTargetProviderConfig{}
	if v, ok := properties["name"]; ok {
		ret.Name = v
	}
	if v, ok := properties["configType"]; ok {
		ret.ConfigType = v
	}
	if v, ok := properties["configData"]; ok {
		ret.ConfigData = v
	}
	if v, ok := properties["context"]; ok {
		ret.Context = v
	}
	if v, ok := properties["inCluster"]; ok {
		val := v
		if val != "" {
			bVal, err := strconv.ParseBool(val)
			if err != nil {
				return ret, v1alpha2.NewCOAError(err, "invalid bool value in the 'inCluster' setting of K8s reference provider", v1alpha2.BadConfig)
			}
			ret.InCluster = bVal
		}
	}
	if v, ok := properties["deploymentStrategy"]; ok && v != "" {
		if v != SERVICES && v != SINGLE_POD && v != SERVICES_NS {
			return ret, v1alpha2.NewCOAError(nil, fmt.Sprintf("invalid deployment strategy. Expected: %s (default), %s or %s", SINGLE_POD, SERVICES, SERVICES_NS), v1alpha2.BadConfig)
		}
		ret.DeploymentStrategy = v
	} else {
		ret.DeploymentStrategy = SINGLE_POD
	}
	return ret, nil
}
func (i *K8sTargetProvider) InitWithMap(properties map[string]string) error {
	config, err := K8sTargetProviderConfigFromMap(properties)
	if err != nil {
		return err
	}
	return i.Init(config)
}
func (i *K8sTargetProvider) Init(config providers.IProviderConfig) error {
	updateConfig, err := toK8sTargetProviderConfig(config)
	if err != nil {
		return errors.New("expected K8sTargetProviderConfig")
	}
	i.Config = updateConfig
	var kConfig *rest.Config
	if i.Config.InCluster {
		kConfig, err = rest.InClusterConfig()
	} else {
		switch i.Config.ConfigType {
		case "path":
			if i.Config.ConfigData == "" {
				if home := homedir.HomeDir(); home != "" {
					i.Config.ConfigData = filepath.Join(home, ".kube", "config")
				} else {
					return v1alpha2.NewCOAError(nil, "can't locate home direction to read default kubernetes config file, to run in cluster, set inCluster config setting to true", v1alpha2.BadConfig)
				}
			}
			kConfig, err = clientcmd.BuildConfigFromFlags("", i.Config.ConfigData)
		case "bytes":
			if i.Config.ConfigData != "" {
				kConfig, err = clientcmd.RESTConfigFromKubeConfig([]byte(i.Config.ConfigData))
				if err != nil {
					return err
				}
			} else {
				return v1alpha2.NewCOAError(nil, "config data is not supplied", v1alpha2.BadConfig)
			}
		default:
			return v1alpha2.NewCOAError(nil, "unrecognized config type, accepted values are: path and bytes", v1alpha2.BadConfig)
		}
	}
	if err != nil {
		return err
	}
	i.Client, err = kubernetes.NewForConfig(kConfig)
	if err != nil {
		return err
	}
	i.DynamicClient, err = dynamic.NewForConfig(kConfig)
	if err != nil {
		return err
	}
	return nil
}
func toK8sTargetProviderConfig(config providers.IProviderConfig) (K8sTargetProviderConfig, error) {
	ret := K8sTargetProviderConfig{}
	data, err := json.Marshal(config)
	if err != nil {
		return ret, err
	}
	err = json.Unmarshal(data, &ret)
	//ret.Name = providers.LoadEnv(ret.Name)
	//ret.ConfigPath = providers.LoadEnv(ret.ConfigPath)
	return ret, err
}

func (i *K8sTargetProvider) getDeployment(ctx context.Context, scope string, name string) ([]model.ComponentSpec, error) {
	deployment, err := i.Client.AppsV1().Deployments(scope).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	components, err := deploymentToComponents(*deployment)
	if err != nil {
		log.Infof("  P (K8s Target Provider): getDeployment failed - %s", err.Error())
		return nil, err
	}
	return components, nil
}
func (i *K8sTargetProvider) fillServiceMeta(ctx context.Context, scope string, name string, component model.ComponentSpec) error {
	svc, err := i.Client.CoreV1().Services(scope).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return nil
		}
		return err
	}
	if component.Metadata == nil {
		component.Metadata = make(map[string]string)
	}
	portData, _ := json.Marshal(svc.Spec.Ports)
	component.Metadata["service.ports"] = string(portData)
	component.Metadata["service.type"] = string(svc.Spec.Type)
	if svc.ObjectMeta.Name != name {
		component.Metadata["service.name"] = svc.ObjectMeta.Name
	}
	if component.Metadata["service.type"] == "LoadBalancer" {
		component.Metadata["service.loadBalancerIP"] = svc.Spec.LoadBalancerIP
	}
	for k, v := range svc.ObjectMeta.Annotations {
		component.Metadata["service.annotation."+k] = v
	}
	return nil
}
func (i *K8sTargetProvider) Get(ctx context.Context, dep model.DeploymentSpec) ([]model.ComponentSpec, error) {
	ctx, span := observability.StartSpan("K8s Target Provider", ctx, &map[string]string{
		"method": "Get",
	})
	log.Infof("  P (K8s Target Provider): getting artifacts: %s - %s", dep.Instance.Scope, dep.Instance.Name)

	var components []model.ComponentSpec
	var err error

	switch i.Config.DeploymentStrategy {
	case "", SINGLE_POD:
		components, err = i.getDeployment(ctx, dep.Instance.Scope, dep.Instance.Name)
		if err != nil {
			observ_utils.CloseSpanWithError(span, err)
			log.Debugf("  P (K8s Target Provider): failed to get - %s", err.Error())
			return nil, err
		}
	case SERVICES, SERVICES_NS:
		components = make([]model.ComponentSpec, 0)
		scope := dep.Instance.Scope
		if i.Config.DeploymentStrategy == SERVICES_NS {
			scope = dep.Instance.Name
		}
		slice := dep.GetComponentSlice()
		for _, component := range slice {
			cComponents, err := i.getDeployment(ctx, scope, component.Name)
			if err != nil {
				observ_utils.CloseSpanWithError(span, err)
				log.Debugf("  P (K8s Target Provider) - failed to get: %s", err.Error())
				return nil, err
			}
			if len(cComponents) > 1 {
				return nil, v1alpha2.NewCOAError(nil, fmt.Sprintf("can't read multiple components when %s strategy or %s strategy is used", SERVICES, SERVICES_NS), v1alpha2.InternalError)
			}
			if len(cComponents) == 1 {
				serviceName := cComponents[0].Name

				if cComponents[0].Metadata != nil {
					if v, ok := cComponents[0].Metadata["service.name"]; ok && v != "" {
						serviceName = v
					}
				}
				if cComponents[0].Metadata == nil {
					cComponents[0].Metadata = make(map[string]string)
				}

				err = i.fillServiceMeta(ctx, scope, serviceName, cComponents[0])
				if err != nil {
					observ_utils.CloseSpanWithError(span, err)
					log.Debugf("failed to get: %s", err.Error())
					return nil, err
				}
				components = append(components, cComponents...)
			}
		}
	}

	observ_utils.CloseSpanWithError(span, nil)
	return components, nil
}
func (i *K8sTargetProvider) removeService(ctx context.Context, scope string, serviceName string) error {
	svc, err := i.Client.CoreV1().Services(scope).Get(ctx, serviceName, metav1.GetOptions{})
	if err == nil && svc != nil {
		err = i.Client.CoreV1().Services(scope).Delete(ctx, serviceName, metav1.DeleteOptions{})
		if err != nil {
			if !k8s_errors.IsNotFound(err) {
				return err
			}
		}
	}
	return nil
}
func (i *K8sTargetProvider) removeDeployment(ctx context.Context, scope string, name string) error {
	err := i.Client.AppsV1().Deployments(scope).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {
		if !k8s_errors.IsNotFound(err) {
			return err
		}
	}
	return nil
}
func (i *K8sTargetProvider) Remove(ctx context.Context, dep model.DeploymentSpec, currentRef []model.ComponentSpec) error {
	ctx, span := observability.StartSpan("K8s Target Provider", ctx, &map[string]string{
		"method": "Remove",
	})
	log.Infof("  P (K8s Target Provider): deleting artifacts: %s - %s", dep.Instance.Scope, dep.Instance.Name)

	switch i.Config.DeploymentStrategy {
	case "", SINGLE_POD:
		serviceName := dep.Instance.Name
		if v, ok := dep.Instance.Metadata["service.name"]; ok && v != "" {
			serviceName = v
		}
		err := i.removeService(ctx, dep.Instance.Scope, serviceName)
		if err != nil {
			observ_utils.CloseSpanWithError(span, err)
			log.Debugf("failed to remove service: %s", err.Error())
			return err
		}
		err = i.removeDeployment(ctx, dep.Instance.Scope, dep.Instance.Name)
		if err != nil {
			observ_utils.CloseSpanWithError(span, err)
			log.Debugf("failed to remove deployment: %s", err.Error())
			return err
		}
	case SERVICES, SERVICES_NS:
		scope := dep.Instance.Scope
		if i.Config.DeploymentStrategy == SERVICES_NS {
			scope = dep.Instance.Name
		}
		slice := dep.GetComponentSlice()
		for _, component := range slice {
			serviceName := component.Name
			if component.Metadata != nil {
				if v, ok := component.Metadata["service.name"]; ok {
					serviceName = v
				}
			}
			err := i.removeService(ctx, scope, serviceName)
			if err != nil {
				observ_utils.CloseSpanWithError(span, err)
				log.Debugf("failed to remove service: %s", err.Error())
				return err
			}
			err = i.removeDeployment(ctx, scope, component.Name)
			if err != nil {
				observ_utils.CloseSpanWithError(span, err)
				log.Debugf("failed to remove deployment: %s", err.Error())
				return err
			}
		}
	}

	//TODO: Should we remove empty namespaces?
	observ_utils.CloseSpanWithError(span, nil)
	return nil
}
func (i *K8sTargetProvider) NeedsUpdate(ctx context.Context, desired []model.ComponentSpec, current []model.ComponentSpec) bool {
	ctx, span := observability.StartSpan("K8s Target Provider", ctx, &map[string]string{
		"method": "NeedsUpdate",
	})
	log.Infof("  P (K8s Target Provider): NeedsUpdate: %d - %d", len(desired), len(current))

	for _, d := range desired {
		found := false
		for _, c := range current {
			if c.Name == d.Name && c.Properties["container.image"] == d.Properties["container.image"] {
				if model.EnvMapsEqual(c.Properties, d.Properties) {
					found = true
					break
				}
			}
		}
		if !found {
			log.Info("  P (K8s Target Provider): NeedsUpdate: returning true")
			observ_utils.CloseSpanWithError(span, nil)
			return true
		}
	}
	observ_utils.CloseSpanWithError(span, nil)
	log.Info("  P (K8s Target Provider): NeedsUpdate: returning false")
	return false
}
func (i *K8sTargetProvider) NeedsRemove(ctx context.Context, desired []model.ComponentSpec, current []model.ComponentSpec) bool {
	ctx, span := observability.StartSpan("K8s Target Provider", ctx, &map[string]string{
		"method": "NeedsRemove",
	})
	log.Infof("  P (K8s Target Provider): NeedsRemove: %d - %d", len(desired), len(current))

	for _, d := range desired {
		for _, c := range current {
			if c.Name == d.Name && c.Properties["container.image"] == d.Properties["container.image"] {
				return true
			}
		}
	}
	observ_utils.CloseSpanWithError(span, nil)
	log.Info("  P (K8s Target Provider): NeedsRemove: returning false")
	return false
}

func (i *K8sTargetProvider) createNamespace(ctx context.Context, scope string) error {
	_, err := i.Client.CoreV1().Namespaces().Get(ctx, scope, metav1.GetOptions{})
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			_, err = i.Client.CoreV1().Namespaces().Create(ctx, &apiv1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: scope,
				},
			}, metav1.CreateOptions{})
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}
func (i *K8sTargetProvider) upsertDeployment(ctx context.Context, scope string, name string, deployment *v1.Deployment) error {
	existing, err := i.Client.AppsV1().Deployments(scope).Get(ctx, name, metav1.GetOptions{})
	if err != nil && !k8s_errors.IsNotFound(err) {
		return err
	}
	if k8s_errors.IsNotFound(err) {
		_, err = i.Client.AppsV1().Deployments(scope).Create(ctx, deployment, metav1.CreateOptions{})
	} else {
		deployment.ResourceVersion = existing.ResourceVersion
		_, err = i.Client.AppsV1().Deployments(scope).Update(ctx, deployment, metav1.UpdateOptions{})
	}
	if err != nil {
		return err
	}
	return nil
}
func (i *K8sTargetProvider) upsertService(ctx context.Context, scope string, name string, service *apiv1.Service) error {
	existing, err := i.Client.CoreV1().Services(scope).Get(ctx, name, metav1.GetOptions{})
	if err != nil && !k8s_errors.IsNotFound(err) {
		return err
	}
	if k8s_errors.IsNotFound(err) {
		_, err = i.Client.CoreV1().Services(scope).Create(ctx, service, metav1.CreateOptions{})
	} else {
		service.ResourceVersion = existing.ResourceVersion
		_, err = i.Client.CoreV1().Services(scope).Update(ctx, service, metav1.UpdateOptions{})
	}
	if err != nil {
		return err
	}
	return nil
}
func (i *K8sTargetProvider) deployComponents(ctx context.Context, span trace.Span, scope string, name string, metadata map[string]string, components []model.ComponentSpec, projector IK8sProjector, instanceName string) error {
	deployment, err := componentsToDeployment(scope, name, metadata, components, instanceName)
	if projector != nil {
		err = projector.ProjectDeployment(scope, name, metadata, components, deployment)
		if err != nil {
			observ_utils.CloseSpanWithError(span, err)
			log.Debugf("  P (K8s Target Provider): failed to project deployment: %s", err.Error())
			return err
		}
	}
	if err != nil {
		observ_utils.CloseSpanWithError(span, err)
		log.Debugf("  P (K8s Target Provider): failed to apply: %s", err.Error())
		return err
	}
	service, err := metadataToService(scope, name, metadata)
	if err != nil {
		observ_utils.CloseSpanWithError(span, err)
		log.Debugf("  P (K8s Target Provider): failed to apply (convert): %s", err.Error())
		return err
	}
	if projector != nil {
		err = projector.ProjectService(scope, name, metadata, service)
		if err != nil {
			observ_utils.CloseSpanWithError(span, err)
			log.Debugf("  P (K8s Target Provider): failed to project service: %s", err.Error())
			return err
		}
	}

	log.Debug("  P (K8s Target Provider): checking namespace")
	err = i.createNamespace(ctx, scope)
	if err != nil {
		observ_utils.CloseSpanWithError(span, err)
		log.Debugf("failed to create namespace: %s", err.Error())
		return err
	}

	log.Debug("  P (K8s Target Provider): creating deployment")
	err = i.upsertDeployment(ctx, scope, name, deployment)
	if err != nil {
		observ_utils.CloseSpanWithError(span, err)
		log.Debugf("  P (K8s Target Provider): failed to apply (API): %s", err.Error())
		return err
	}

	if service != nil {
		log.Debug("  P (K8s Target Provider): creating service")
		err = i.upsertService(ctx, scope, service.Name, service)
		if err != nil {
			observ_utils.CloseSpanWithError(span, err)
			log.Debugf("  P (K8s Target Provider): failed to apply (service): %s", err.Error())
			return err
		}
	}
	return nil
}
func (i *K8sTargetProvider) Apply(ctx context.Context, dep model.DeploymentSpec) error {
	ctx, span := observability.StartSpan("K8s Target Provider", ctx, &map[string]string{
		"method": "Apply",
	})
	log.Infof("  P (K8s Target Provider): applying artifacts: %s - %s", dep.Instance.Scope, dep.Instance.Name)

	projector, err := createProjector(i.Config.Projector)
	if err != nil {
		observ_utils.CloseSpanWithError(span, err)
		log.Debugf("  P (K8s Target Provider): failed to create projector: %s", err.Error())
		return err
	}

	components := dep.GetComponentSlice()

	switch i.Config.DeploymentStrategy {
	case "", SINGLE_POD:
		err = i.deployComponents(ctx, span, dep.Instance.Scope, dep.Instance.Name, dep.Instance.Metadata, components, projector, dep.Instance.Name)
		if err != nil {
			observ_utils.CloseSpanWithError(span, err)
			log.Debugf("  P (K8s Target Provider): failed to apply components: %s", err.Error())
			return err
		}
	case SERVICES, SERVICES_NS:
		scope := dep.Instance.Scope
		if i.Config.DeploymentStrategy == SERVICES_NS {
			scope = dep.Instance.Name
		}
		for _, component := range components {
			if dep.Instance.Metadata != nil {
				if v, ok := dep.Instance.Metadata[ENV_NAME]; ok && v != "" {
					if component.Metadata == nil {
						component.Metadata = make(map[string]string)
					}
					component.Metadata[ENV_NAME] = v
				}
			}
			err = i.deployComponents(ctx, span, scope, component.Name, component.Metadata, []model.ComponentSpec{component}, projector, dep.Instance.Name)
			if err != nil {
				observ_utils.CloseSpanWithError(span, err)
				log.Debugf("  P (K8s Target Provider): failed to apply components: %s", err.Error())
				return err
			}
		}
	}
	observ_utils.CloseSpanWithError(span, nil)
	return nil
}
func deploymentToComponents(deployment v1.Deployment) ([]model.ComponentSpec, error) {
	components := make([]model.ComponentSpec, len(deployment.Spec.Template.Spec.Containers))
	for i, c := range deployment.Spec.Template.Spec.Containers {
		component := model.ComponentSpec{
			Name:       c.Name,
			Properties: make(map[string]string),
		}
		component.Properties["container.image"] = c.Image
		policy := string(c.ImagePullPolicy)
		if policy != "" {
			component.Properties["container.imagePullPolicy"] = policy
		}
		if len(c.Ports) > 0 {
			ports, _ := json.Marshal(c.Ports)
			component.Properties["container.ports"] = string(ports)
		}
		if len(c.Args) > 0 {
			args, _ := json.Marshal(c.Args)
			component.Properties["container.args"] = string(args)
		}
		if len(c.Command) > 0 {
			commands, _ := json.Marshal(c.Command)
			component.Properties["container.commands"] = string(commands)
		}
		resources, _ := json.Marshal(c.Resources)
		if string(resources) != "{}" {
			component.Properties["container.resources"] = string(resources)
		}
		if len(c.VolumeMounts) > 0 {
			volumeMounts, _ := json.Marshal(c.VolumeMounts)
			component.Properties["container.volumeMounts"] = string(volumeMounts)
		}
		if len(c.Env) > 0 {
			for _, e := range c.Env {
				component.Properties["env."+e.Name] = e.Value
			}
		}
		components[i] = component
	}
	return components, nil
}
func metadataToService(scope string, name string, metadata map[string]string) (*apiv1.Service, error) {
	if len(metadata) == 0 {
		return nil, nil
	}

	servicePorts := make([]apiv1.ServicePort, 0)
	if v, ok := metadata["service.ports"]; ok {
		e := json.Unmarshal([]byte(v), &servicePorts)
		if e != nil {
			return nil, e
		}
	} else {
		return nil, nil
	}
	service := apiv1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      utils.ReadString(metadata, "service.name", name),
			Namespace: scope,
			Labels: map[string]string{
				"app": name,
			},
		},
		Spec: apiv1.ServiceSpec{
			Type:  apiv1.ServiceType(utils.ReadString(metadata, "service.type", "ClusterIP")),
			Ports: servicePorts,
			Selector: map[string]string{
				"app": name,
			},
		},
	}
	if _, ok := metadata["service.loadBalancerIP"]; ok {
		service.Spec.LoadBalancerIP = utils.ReadString(metadata, "service.loadBalancerIP", "")
	}
	annotations := utils.CollectStringMap(metadata, "service.annotation.")
	if len(annotations) > 0 {
		service.ObjectMeta.Annotations = make(map[string]string)
		for k, v := range annotations {
			service.ObjectMeta.Annotations[k[19:]] = v
		}
	}
	return &service, nil
}
func int32Ptr(i int32) *int32 { return &i }
func componentsToDeployment(scope string, name string, metadata map[string]string, components []model.ComponentSpec, instanceName string) (*v1.Deployment, error) {
	deployment := v1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: v1.DeploymentSpec{
			Replicas: int32Ptr(utils.ReadInt32(metadata, "deployment.replicas", 1)),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": name,
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": name,
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{},
				},
			},
		},
	}
	for _, c := range components {
		ports := make([]apiv1.ContainerPort, 0)
		if v, ok := c.Properties["container.ports"]; ok && v != "" {
			e := json.Unmarshal([]byte(v), &ports)
			if e != nil {
				return nil, e
			}
		}
		container := apiv1.Container{
			Name:            c.Name,
			Image:           c.Properties["container.image"],
			Ports:           ports,
			ImagePullPolicy: apiv1.PullPolicy(utils.ReadString(c.Properties, "container.imagePullPolicy", "Always")),
		}
		if v, ok := c.Properties["container.args"]; ok && v != "" {
			args := make([]string, 0)
			e := json.Unmarshal([]byte(v), &args)
			if e != nil {
				return nil, e
			}
			container.Args = args
		}
		if v, ok := c.Properties["container.commands"]; ok && v != "" {
			cmds := make([]string, 0)
			e := json.Unmarshal([]byte(v), &cmds)
			if e != nil {
				return nil, e
			}
			container.Command = cmds
		}
		if v, ok := c.Properties["container.resources"]; ok && v != "" {
			res := apiv1.ResourceRequirements{}
			e := json.Unmarshal([]byte(v), &res)
			if e != nil {
				return nil, e
			}
			container.Resources = res
		}
		if v, ok := c.Properties["container.volumeMounts"]; ok && v != "" {
			mounts := make([]apiv1.VolumeMount, 0)
			e := json.Unmarshal([]byte(v), &mounts)
			if e != nil {
				return nil, e
			}
			container.VolumeMounts = mounts
		}
		for k, v := range c.Properties {
			tv := utils.ProjectValue(v, instanceName)
			if strings.HasPrefix(k, "env.") {
				if container.Env == nil {
					container.Env = make([]apiv1.EnvVar, 0)
				}
				container.Env = append(container.Env, apiv1.EnvVar{
					Name:  k[4:],
					Value: tv,
				})
			}
		}
		agentName := metadata[ENV_NAME]
		if agentName != "" {
			if container.Env == nil {
				container.Env = make([]apiv1.EnvVar, 0)
			}
			container.Env = append(container.Env, apiv1.EnvVar{
				Name:  ENV_NAME,
				Value: agentName + ".default.svc.cluster.local", //agent is currently always installed under deault
			})
		}
		deployment.Spec.Template.Spec.Containers = append(deployment.Spec.Template.Spec.Containers, container)
	}
	if v, ok := metadata["deployment.imagePullSecrets"]; ok && v != "" {
		secrets := make([]apiv1.LocalObjectReference, 0)
		e := json.Unmarshal([]byte(v), &secrets)
		if e != nil {
			return nil, e
		}
		deployment.Spec.Template.Spec.ImagePullSecrets = secrets
	}
	if v, ok := metadata["deployment.volumes"]; ok && v != "" {
		volumes := make([]apiv1.Volume, 0)
		e := json.Unmarshal([]byte(v), &volumes)
		if e != nil {
			return nil, e
		}
		deployment.Spec.Template.Spec.Volumes = volumes
	}
	if v, ok := metadata["deployment.nodeSelector"]; ok && v != "" {
		selector := make(map[string]string)
		e := json.Unmarshal([]byte(v), &selector)
		if e != nil {
			return nil, e
		}
		deployment.Spec.Template.Spec.NodeSelector = selector
	}
	return &deployment, nil
}

func createProjector(projector string) (IK8sProjector, error) {
	switch projector {
	case "noop":
		return &projectors.NoOpProjector{}, nil
	case "":
		return nil, nil
	}
	return nil, v1alpha2.NewCOAError(nil, fmt.Sprintf("project type '%s' is unsupported", projector), v1alpha2.BadConfig)
}

type IK8sProjector interface {
	ProjectDeployment(scope string, name string, metadata map[string]string, components []model.ComponentSpec, deployment *v1.Deployment) error
	ProjectService(scope string, name string, metadata map[string]string, service *apiv1.Service) error
}