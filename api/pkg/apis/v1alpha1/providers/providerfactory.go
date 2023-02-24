/*
Copyright 2022 The COA Authors
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package providers

import (
	"fmt"

	"github.com/azure/symphony/coa/pkg/apis/v1alpha2"
	cp "github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers/probe/rtsp"
	cvref "github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers/reference/customvision"
	httpref "github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers/reference/http"
	k8sref "github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers/reference/k8s"
	httpreporter "github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers/reporter/http"
	k8sreporter "github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers/reporter/k8s"
	k8sstate "github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/states/k8s"

	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers/states/httpstate"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers/states/memorystate"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers/uploader/azure/blob"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/vendors"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/model"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/target/adb"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/target/azure/adu"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/target/azure/iotedge"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/target/helm"
	targethttp "github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/target/http"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/target/k8s"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/target/kubectl"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/target/mqtt"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/target/proxy"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/target/script"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/target/staging"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/providers/target/win10/sideload"
)

type SymphonyProviderFactory struct {
}

func (c SymphonyProviderFactory) CreateProviders(config vendors.VendorConfig) (map[string]map[string]cp.IProvider, error) {
	ret := make(map[string]map[string]cp.IProvider)
	for _, m := range config.Managers {
		ret[m.Name] = make(map[string]cp.IProvider)
		for k, p := range m.Providers {
			switch p.Type {
			case "providers.state.memory":
				mProvider := &memorystate.MemoryStateProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.state.k8s":
				mProvider := &k8sstate.K8sStateProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.state.http":
				mProvider := &httpstate.HttpStateProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.reference.k8s":
				mProvider := &k8sref.K8sReferenceProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.reference.customvision":
				mProvider := &cvref.CustomVisionReferenceProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.reference.http":
				mProvider := &httpref.HTTPReferenceProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.reporter.k8s":
				mProvider := &k8sreporter.K8sReporter{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.reporter.http":
				mProvider := &httpreporter.HTTPReporter{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.probe.rtsp":
				mProvider := &rtsp.RTSPProbeProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.uploader.azure.blob":
				mProvider := &blob.AzureBlobUploader{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.target.azure.iotedge":
				mProvider := &iotedge.IoTEdgeTargetProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.target.azure.adu":
				mProvider := &adu.ADUTargetProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.target.k8s":
				mProvider := &k8s.K8sTargetProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.target.kubectl":
				mProvider := &kubectl.KubectlTargetProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.target.staging":
				mProvider := &staging.StagingTargetProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.target.script":
				mProvider := &script.ScriptProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.target.http":
				mProvider := &targethttp.HttpTargetProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.target.win10.sideload":
				mProvider := &sideload.Win10SideLoadProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.target.adb":
				mProvider := &adb.AdbProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.target.proxy":
				mProvider := &proxy.ProxyUpdateProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			case "providers.target.mqtt":
				mProvider := &mqtt.MQTTTargetProvider{}
				err := mProvider.Init(p.Config)
				if err != nil {
					return ret, err
				}
				ret[m.Name][k] = mProvider
			}
		}
	}
	return ret, nil
}

func (s SymphonyProviderFactory) CreateProvider(providerType string, config cp.IProviderConfig) (cp.IProvider, error) {
	switch providerType {
	case "providers.target.iotedge":

	}
	return nil, nil
}
func CreateProviderForTargetRole(role string, target model.TargetSpec, override cp.IProvider) (cp.IProvider, error) {
	for _, topology := range target.Topologies {
		for _, binding := range topology.Bindings {
			testRole := role
			if role == "" || role == "container" {
				testRole = "instance"
			}
			if binding.Role == testRole {
				switch binding.Provider {
				case "providers.target.azure.iotedge":
					provider := &iotedge.IoTEdgeTargetProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.target.azure.adu":
					provider := &adu.ADUTargetProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.target.k8s":
					provider := &k8s.K8sTargetProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.target.kubectl":
					provider := &kubectl.KubectlTargetProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.target.staging":
					provider := &staging.StagingTargetProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.target.script":
					provider := &script.ScriptProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.target.http":
					provider := &targethttp.HttpTargetProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.target.win10.sideload":
					provider := &sideload.Win10SideLoadProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.target.adb":
					provider := &adb.AdbProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.target.proxy":
					if override == nil {
						provider := &proxy.ProxyUpdateProvider{}
						err := provider.InitWithMap(binding.Config)
						if err != nil {
							return nil, err
						}
						return provider, nil
					} else {
						return override, nil
					}
				case "providers.target.mqtt":
					if override == nil {
						provider := &mqtt.MQTTTargetProvider{}
						err := provider.InitWithMap(binding.Config)
						if err != nil {
							return nil, err
						}
						return provider, nil
					} else {
						return override, nil
					}
				case "providers.state.memory":
					provider := &memorystate.MemoryStateProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.state.k8s":
					provider := &k8sstate.K8sStateProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.state.http":
					provider := &httpstate.HttpStateProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.reference.k8s":
					provider := &k8sref.K8sReferenceProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.reference.customvision":
					provider := &cvref.CustomVisionReferenceProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.reference.http":
					provider := &httpref.HTTPReferenceProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.reporter.k8s":
					provider := &k8sreporter.K8sReporter{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.reporter.http":
					provider := &httpreporter.HTTPReporter{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				case "providers.target.helm":
					provider := &helm.HelmTargetProvider{}
					err := provider.InitWithMap(binding.Config)
					if err != nil {
						return nil, err
					}
					return provider, nil
				}
			}
		}
	}
	return nil, v1alpha2.NewCOAError(nil, fmt.Sprintf("target doesn't have a '%s' role defined", role), v1alpha2.BadConfig)
}