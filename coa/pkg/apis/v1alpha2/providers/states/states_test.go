/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

package states

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type CustomType struct {
	Name string                 `json:"name"`
	Map  map[string]interface{} `json:"map"` // map of string to InnerType
}

type InnerType struct {
	InnerName string `json:"innerName"`
}

func (i InnerType) DeepEquals(other InnerType) bool {
	return i.InnerName == other.InnerName
}

func (c CustomType) DeepEquals(other CustomType) (bool, error) {
	if c.Name != other.Name {
		return false, nil
	}

	for k, v := range c.Map {
		if otherVal, ok := other.Map[k]; !ok {
			return false, errors.New("other map does not contain key " + k)
		} else {
			var i1 InnerType
			var i2 InnerType
			d1, _ := json.Marshal(v)
			d2, _ := json.Marshal(otherVal)
			e1 := json.Unmarshal(d1, &i1)
			e2 := json.Unmarshal(d2, &i2)
			if e1 != nil || e2 != nil {
				return false, errors.New("cannot unmarshal value to inner type for key " + k)
			}
			if !i1.DeepEquals(i2) {
				return false, nil
			}
		}
	}

	return true, nil
}

func GetCustomTypeObject(body interface{}) (CustomType, error) {
	j, _ := json.Marshal(body)
	var customType CustomType
	err := json.Unmarshal(j, &customType)
	if err != nil {
		return CustomType{}, err
	}
	return customType, nil
}

func TestStateEntryDeepCopy(t *testing.T) {
	stateEntry := StateEntry{
		ID:   "id",
		ETag: "etag",
		Body: CustomType{
			Name: "name",
			Map: map[string]interface{}{
				"key1": InnerType{
					InnerName: "innerName",
				},
			},
		},
	}
	stateEntryCopy := stateEntry.DeepCopy()

	assert.Equal(t, stateEntry.ID, stateEntryCopy.ID)
	assert.Equal(t, stateEntry.ETag, stateEntryCopy.ETag)
	c1, err := GetCustomTypeObject(stateEntry.Body)
	assert.Nil(t, err)
	c2, err := GetCustomTypeObject(stateEntryCopy.Body)
	assert.Nil(t, err)
	equals, err := c1.DeepEquals(c2)
	assert.True(t, equals)
	assert.Nil(t, err)
}

func TestStateEntryJsonMatch(t *testing.T) {
	stateEntry := StateEntry{
		ID:   "id",
		ETag: "etag",
		Body: CustomType{
			Name: "name",
			Map: map[string]interface{}{
				"key1": InnerType{
					InnerName: "innerName",
				},
			},
		},
	}

	stateEntryCopy := stateEntry.DeepCopy()

	b := JsonPathMatch(stateEntryCopy.Body, "$.name", "name")
	assert.True(t, b)
	b = JsonPathMatch(stateEntryCopy.Body, "$.map.key1.innerName", "innerName")
	assert.True(t, b)
	b = JsonPathMatch(stateEntryCopy.Body, "$.map.key1.innerName", "innerName2")
	assert.False(t, b)
	b = JsonPathMatch(stateEntryCopy.Body, "$.map.key2.innerName", "innerName")
	assert.False(t, b)
}
