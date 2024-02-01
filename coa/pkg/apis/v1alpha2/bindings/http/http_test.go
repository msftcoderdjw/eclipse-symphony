/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

package http

import (
	"crypto/tls"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"

	v1alpha2 "github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2"
	autogen "github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/providers/certs/autogen"
	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2/providers/pubsub/memory"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestHTTPEcho(t *testing.T) {
	config := HttpBindingConfig{
		Port: 8080,
		TLS:  false,
	}
	binding := HttpBinding{}
	endpoints := []v1alpha2.Endpoint{
		{
			Methods: []string{"GET"},
			Route:   "greetings",
			Version: "v1",
			Handler: func(c v1alpha2.COARequest) v1alpha2.COAResponse {
				return v1alpha2.COAResponse{
					Body: []byte("Hi there!!"),
				}
			},
		},
		{
			Methods: []string{"GET"},
			Route:   "greetings2",
			Version: "v1",
			Handler: func(c v1alpha2.COARequest) v1alpha2.COAResponse {
				return v1alpha2.COAResponse{
					Body: []byte("Hi " + c.Parameters["name"] + "!!"),
				}
			},
		},
		{
			Methods:    []string{"GET"},
			Route:      "greetings3",
			Version:    "v1",
			Parameters: []string{"name"},
			Handler: func(c v1alpha2.COARequest) v1alpha2.COAResponse {
				return v1alpha2.COAResponse{
					Body: []byte("Hi " + c.Parameters["__name"] + "!!!"),
				}
			},
		},
		{
			Methods: []string{"POST"},
			Route:   "greetingsWithMetadata",
			Version: "v1",
			Handler: func(c v1alpha2.COARequest) v1alpha2.COAResponse {
				metadata := c.Metadata
				value := metadata["key"]
				return v1alpha2.COAResponse{
					Metadata: map[string]string{
						"key": value,
					},
					Body: []byte("Hi " + value + "!!!!"),
				}
			},
		},
	}
	err := binding.Launch(config, endpoints, nil)
	assert.Nil(t, err)

	client := &http.Client{}
	req, err := http.NewRequest(fasthttp.MethodGet, "http://localhost:8080/v1/greetings", nil)
	assert.Nil(t, err)
	resp, err := client.Do(req)
	assert.Nil(t, err)
	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, 200)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)
	assert.Equal(t, string(bodyBytes), "Hi there!!")

	// query args
	req2, err := http.NewRequest(fasthttp.MethodGet, "http://localhost:8080/v1/greetings2?name=John", nil)
	assert.Nil(t, err)
	resp2, err := client.Do(req2)
	assert.Nil(t, err)

	defer resp2.Body.Close()
	assert.Equal(t, resp2.StatusCode, 200)
	bodyBytes2, err := ioutil.ReadAll(resp2.Body)
	assert.Nil(t, err)

	assert.Equal(t, string(bodyBytes2), "Hi John!!")

	// path parameters
	req3, err := http.NewRequest(fasthttp.MethodGet, "http://localhost:8080/v1/greetings3/John", nil)
	assert.Nil(t, err)
	resp3, err := client.Do(req3)
	assert.Nil(t, err)

	defer resp3.Body.Close()
	assert.Equal(t, resp3.StatusCode, 200)
	bodyBytes3, err := ioutil.ReadAll(resp3.Body)
	assert.Nil(t, err)

	assert.Equal(t, string(bodyBytes3), "Hi John!!!")

	// req metadata and resp metadata
	req4, err := http.NewRequest(fasthttp.MethodPost, "http://localhost:8080/v1/greetingsWithMetadata", nil)
	assert.Nil(t, err)
	req4Metadata := map[string]string{
		"key": "Alice",
	}
	b, _ := json.Marshal(req4Metadata)
	req4.Header.Add(v1alpha2.COAMetaHeader, string(b))
	resp4, err := client.Do(req4)
	assert.Nil(t, err)

	defer resp4.Body.Close()
	assert.Equal(t, resp4.StatusCode, 200)
	bodyBytes4, err := ioutil.ReadAll(resp4.Body)
	assert.Nil(t, err)

	assert.Equal(t, string(bodyBytes4), "Hi Alice!!!!")
	resp4Metadata := resp4.Header.Get(v1alpha2.COAMetaHeader)
	var resp4MetadataMap map[string]string
	json.Unmarshal([]byte(resp4Metadata), &resp4MetadataMap)
	assert.Equal(t, resp4MetadataMap["key"], "Alice")
}

func TestHTTPEchoWithTLS(t *testing.T) {
	config := HttpBindingConfig{
		Port: 8888,
		TLS:  true,
		CertProvider: CertProviderConfig{
			Type: "certs.autogen",
			Config: autogen.AutoGenCertProviderConfig{
				Name: "test",
			},
		},
	}
	binding := HttpBinding{}
	endpoints := []v1alpha2.Endpoint{
		{
			Methods: []string{"GET"},
			Route:   "greetings",
			Version: "v1",
			Handler: func(c v1alpha2.COARequest) v1alpha2.COAResponse {
				return v1alpha2.COAResponse{
					Body: []byte("Hi there!!"),
				}
			},
		},
	}
	err := binding.Launch(config, endpoints, nil)
	assert.Nil(t, err)

	// need to wait for tls cert creation (it is in another go routine)
	time.Sleep(5 * time.Second)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	req, err := http.NewRequest(fasthttp.MethodGet, "https://localhost:8888/v1/greetings", nil)
	assert.Nil(t, err)
	resp, err := client.Do(req)
	assert.Nil(t, err)

	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, 200)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)

	assert.Equal(t, string(bodyBytes), "Hi there!!")
}

func TestHTTEchoWithPipeline(t *testing.T) {
	pubsub := &memory.InMemoryPubSubProvider{}
	err := pubsub.Init(memory.InMemoryPubSubConfig{})
	assert.Nil(t, err)

	os.Setenv("ENABLE_APP_INSIGHT", "true")
	os.Setenv("APP_INSIGHT_KEY", "a04ac91b-747a-4d40-bd86-d49397aaf1e8")
	config := HttpBindingConfig{
		Port: 8081,
		TLS:  false,
		Pipeline: []MiddlewareConfig{
			{
				Type: "middleware.http.cors",
				Properties: map[string]interface{}{
					"Any": "value",
				},
			},
			{
				Type:       "middleware.http.trail",
				Properties: map[string]interface{}{},
			},
			{
				Type: "middleware.http.telemetry",
				Properties: map[string]interface{}{
					"enabled":                 true,
					"maxBatchSize":            8192,
					"maxBatchIntervalSeconds": 2,
					"client":                  "coabinding-test", // will be override as uuid
				},
			},
		},
	}
	binding := HttpBinding{}
	endpoints := []v1alpha2.Endpoint{
		{
			Methods: []string{"GET"},
			Route:   "greetings",
			Version: "v1",
			Handler: func(c v1alpha2.COARequest) v1alpha2.COAResponse {
				return v1alpha2.COAResponse{
					Body: []byte("Hi there!!"),
				}
			},
		},
	}
	err = binding.Launch(config, endpoints, pubsub)
	assert.Nil(t, err)

	time.Sleep(5 * time.Second)

	client := &http.Client{}
	req, err := http.NewRequest(fasthttp.MethodGet, "http://localhost:8081/v1/greetings", nil)
	assert.Nil(t, err)
	resp, err := client.Do(req)
	assert.Nil(t, err)

	defer resp.Body.Close()
	assert.Equal(t, resp.StatusCode, 200)
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	assert.Nil(t, err)

	assert.Equal(t, string(bodyBytes), "Hi there!!")
	assert.Equal(t, resp.Header.Get("Access-Control-Allow-Origin"), corsAllowOrigin)
	assert.Equal(t, resp.Header.Get("Access-Control-Allow-Methods"), corsAllowMethods)
	assert.Equal(t, resp.Header.Get("Access-Control-Allow-Credentials"), corsAllowCredentials)
	assert.Equal(t, resp.Header.Get("Access-Control-Allow-Headers"), corsAllowHeaders)
	assert.Equal(t, resp.Header.Get("Any"), "value")

	time.Sleep(5 * time.Second) // wait for telemetry to send data
}
