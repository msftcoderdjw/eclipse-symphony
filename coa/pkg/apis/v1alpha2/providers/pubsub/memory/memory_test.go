/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

package memory

import (
	"testing"
	"time"

	"github.com/eclipse-symphony/symphony/coa/pkg/apis/v1alpha2"
	"github.com/stretchr/testify/assert"
)

func TestBasicPubSub(t *testing.T) {
	sig := make(chan int)
	msg := ""
	provider := InMemoryPubSubProvider{}
	provider.Init(InMemoryPubSubConfig{Name: "test"})
	provider.Subscribe("test", v1alpha2.EventHandler{
		Handler: func(topic string, event v1alpha2.Event) error {
			msg = event.Body.(string)
			sig <- 1
			return nil
		},
	})
	provider.Publish("test", v1alpha2.Event{Body: "TEST"})
	<-sig
	assert.Equal(t, "TEST", msg)
}

func TestMultipleSubscriber(t *testing.T) {
	sig1 := make(chan int)
	sig2 := make(chan int)
	msg1 := ""
	msg2 := ""
	provider := InMemoryPubSubProvider{}
	provider.Init(InMemoryPubSubConfig{Name: "test"})
	provider.Subscribe("test", v1alpha2.EventHandler{
		Handler: func(topic string, event v1alpha2.Event) error {
			msg1 = event.Body.(string)
			sig1 <- 1
			return nil
		},
	})
	provider.Subscribe("test", v1alpha2.EventHandler{
		Handler: func(topic string, event v1alpha2.Event) error {
			msg2 = event.Body.(string)
			sig2 <- 1
			return nil
		},
	})
	provider.Publish("test", v1alpha2.Event{Body: "TEST"})
	<-sig1
	<-sig2
	assert.Equal(t, "TEST", msg1)
	assert.Equal(t, "TEST", msg2)
}

func TestMultipleTopics(t *testing.T) {
	sig1 := make(chan int)
	sig2 := make(chan int)
	msg1 := ""
	msg2 := ""
	provider := InMemoryPubSubProvider{}
	provider.Init(InMemoryPubSubConfig{Name: "test"})
	provider.Subscribe("test1", v1alpha2.EventHandler{
		Handler: func(topic string, event v1alpha2.Event) error {
			msg1 = event.Body.(string)
			sig1 <- 1
			return nil
		},
	})
	provider.Subscribe("test2", v1alpha2.EventHandler{
		Handler: func(topic string, event v1alpha2.Event) error {
			msg2 = event.Body.(string)
			sig2 <- 1
			return nil
		},
	})
	provider.Publish("test1", v1alpha2.Event{Body: "TEST1"})
	provider.Publish("test2", v1alpha2.Event{Body: "TEST2"})
	<-sig1
	<-sig2
	assert.Equal(t, "TEST1", msg1)
	assert.Equal(t, "TEST2", msg2)
}
func TestMemoryPubsubProviderConfigFromMapNil(t *testing.T) {
	_, err := InMemoryPubSubConfigFromMap(nil)
	assert.Nil(t, err)
}

func TestMemoryPubsubProviderConfigFromMapEmpty(t *testing.T) {
	_, err := InMemoryPubSubConfigFromMap(map[string]string{})
	assert.Nil(t, err)
}
func TestMemoryPubsubProviderConfigFromMap(t *testing.T) {
	config, err := InMemoryPubSubConfigFromMap(map[string]string{
		"name": "my-name",
	})
	assert.Nil(t, err)
	assert.Equal(t, "my-name", config.Name)
}

func TestClone(t *testing.T) {
	provider := InMemoryPubSubProvider{}
	provider.Init(InMemoryPubSubConfig{Name: "test"})
	assert.Equal(t, "test", provider.ID())

	p, err := provider.Clone(InMemoryPubSubConfig{
		Name: "",
	})
	assert.NotNil(t, p)
	assert.Nil(t, err)

	pc, err := provider.Clone(nil)
	assert.NotNil(t, pc)
	assert.Nil(t, err)
}

// TestInitWithMap tests the InitWithMap function
func TestInitWithMap(t *testing.T) {
	provider := InMemoryPubSubProvider{}
	err := provider.InitWithMap(map[string]string{
		"name": "my-name",
	})
	assert.Nil(t, err)
	assert.Equal(t, "my-name", provider.Config.Name)
}

// TestCloneWithEmptyConfig tests the Clone function with an empty config
func TestCloneWithEmptyConfig(t *testing.T) {
	provider := InMemoryPubSubProvider{}
	_, err := provider.Clone(InMemoryPubSubConfig{})
	assert.Nil(t, err)
}

// TestCloneWithConfig tests the Clone function with a config
func TestCloneWithConfig(t *testing.T) {
	provider := InMemoryPubSubProvider{}
	_, err := provider.Clone(InMemoryPubSubConfig{
		Name: "my-name",
	})
	assert.Nil(t, err)
}

func TestMemoryPubsubProviderBadRequest(t *testing.T) {
	provider := InMemoryPubSubProvider{}
	provider.Init(InMemoryPubSubConfig{
		Name:                      "test",
		SubscriberRetryCount:      5,
		SubscriberRetryWaitSecond: 1,
	})

	ch := make(chan struct{})
	count := 0

	err := provider.Subscribe("test", v1alpha2.EventHandler{
		Handler: func(topic string, event v1alpha2.Event) error {
			count += 1
			ch <- struct{}{}
			return v1alpha2.NewCOAError(nil, "insert bad request", v1alpha2.BadRequest)
		},
	})
	assert.Nil(t, err)
	err = provider.Publish("test", v1alpha2.Event{
		Body: "test",
	})
	assert.Nil(t, err)

	signal := 0
	for signal < 1 {
		select {
		case <-ch:
			close(ch)
		case <-time.After(5 * time.Second):
			// Timeout, function was not called
			t.Fatal("Function was not called within the timeout period")
		}
		signal += 1
	}
	time.Sleep(5 * time.Second) // Wait to ensure no further calls are made
	assert.Equal(t, 1, count)
}

func TestMemoryPubsubProviderInternalError(t *testing.T) {
	provider := InMemoryPubSubProvider{}
	provider.Init(InMemoryPubSubConfig{
		Name:                      "test",
		SubscriberRetryCount:      4,
		SubscriberRetryWaitSecond: 1,
	})

	ch := make(chan struct{})
	count := 0

	err := provider.Subscribe("test", v1alpha2.EventHandler{
		Handler: func(topic string, event v1alpha2.Event) error {
			count += 1
			ch <- struct{}{}
			return v1alpha2.NewCOAError(nil, "insert internal error", v1alpha2.InternalError)
		},
	})
	assert.Nil(t, err)
	err = provider.Publish("test", v1alpha2.Event{
		Body: "test",
	})
	assert.Nil(t, err)

	signal := 0
	for signal < 5 {
		select {
		case <-ch:
		case <-time.After(5 * time.Second):
			// Timeout, function was not called
			t.Fatal("Function was not called within the timeout period")
		}
		signal += 1
	}
	close(ch)
	time.Sleep(2 * time.Second) // Wait to ensure no further calls are made
	assert.Equal(t, 5, count)
}
