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

package vendors

import (
	"encoding/json"

	"github.com/azure/symphony/coa/pkg/apis/v1alpha2"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/managers"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/observability"
	observ_utils "github.com/azure/symphony/coa/pkg/apis/v1alpha2/observability/utils"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/providers"
	"github.com/azure/symphony/coa/pkg/apis/v1alpha2/vendors"
	"github.com/azure/symphony/coa/pkg/logger"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/managers/solutions"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/model"
	"github.com/azure/symphony/api/pkg/apis/v1alpha1/utils"
	"github.com/valyala/fasthttp"
)

var uLog = logger.NewLogger("coa.runtime")

type SolutionsVendor struct {
	vendors.Vendor
	SolutionsManager *solutions.SolutionsManager
}

func (o *SolutionsVendor) GetInfo() vendors.VendorInfo {
	return vendors.VendorInfo{
		Version:  o.Vendor.Version,
		Name:     "Solutions",
		Producer: "Microsoft",
	}
}

func (e *SolutionsVendor) Init(config vendors.VendorConfig, factories []managers.IManagerFactroy, providers map[string]map[string]providers.IProvider) error {
	err := e.Vendor.Init(config, factories, providers)
	if err != nil {
		return err
	}
	for _, m := range e.Managers {
		if c, ok := m.(*solutions.SolutionsManager); ok {
			e.SolutionsManager = c
		}
	}
	if e.SolutionsManager == nil {
		return v1alpha2.NewCOAError(nil, "solutions manager is not supplied", v1alpha2.MissingConfig)
	}
	return nil
}

func (o *SolutionsVendor) HasLoop() bool {
	return false
}

func (o *SolutionsVendor) GetEndpoints() []v1alpha2.Endpoint {
	route := "solutions"
	if o.Route != "" {
		route = o.Route
	}
	return []v1alpha2.Endpoint{
		{
			Methods:    []string{fasthttp.MethodGet, fasthttp.MethodPost, fasthttp.MethodDelete},
			Route:      route,
			Version:    o.Version,
			Handler:    o.onSolutions,
			Parameters: []string{"name?"},
		},
	}
}

func (c *SolutionsVendor) onSolutions(request v1alpha2.COARequest) v1alpha2.COAResponse {
	pCtx, span := observability.StartSpan("Solutions Vendor", request.Context, &map[string]string{
		"method": "onSolutions",
	})
	tLog.Info("~ Solutions Manager ~ : onSolutions")

	switch request.Method {
	case fasthttp.MethodGet:
		ctx, span := observability.StartSpan("onSolutions-GET", pCtx, nil)
		id := request.Parameters["__name"]
		var err error
		var state interface{}
		isArray := false
		if id == "" {
			state, err = c.SolutionsManager.ListSpec(ctx)
			isArray = true
		} else {
			state, err = c.SolutionsManager.GetSpec(ctx, id)
		}
		if err != nil {
			return observ_utils.CloseSpanWithCOAResponse(span, v1alpha2.COAResponse{
				State: v1alpha2.InternalError,
				Body:  []byte(err.Error()),
			})
		}
		jData, _ := utils.FormatObject(state, isArray, request.Parameters["path"], request.Parameters["doc-type"])
		resp := observ_utils.CloseSpanWithCOAResponse(span, v1alpha2.COAResponse{
			State:       v1alpha2.OK,
			Body:        jData,
			ContentType: "application/json",
		})
		if request.Parameters["doc-type"] == "yaml" {
			resp.ContentType = "application/text"
		}
		return resp
	case fasthttp.MethodPost:
		ctx, span := observability.StartSpan("onSolutions-POST", pCtx, nil)
		id := request.Parameters["__name"]

		embed_type := request.Parameters["embed-type"]
		embed_component := request.Parameters["embed-component"]
		embed_property := request.Parameters["embed-property"]

		var solution model.SolutionSpec

		if embed_type != "" && embed_component != "" && embed_property != "" {
			solution = model.SolutionSpec{
				DisplayName: id,
				Components: []model.ComponentSpec{
					{
						Name: embed_component,
						Type: embed_type,
						Properties: map[string]string{
							embed_property: string(request.Body),
						},
					},
				},
			}
		} else {
			err := json.Unmarshal(request.Body, &solution)
			if err != nil {
				return observ_utils.CloseSpanWithCOAResponse(span, v1alpha2.COAResponse{
					State: v1alpha2.InternalError,
					Body:  []byte(err.Error()),
				})
			}
		}
		err := c.SolutionsManager.UpsertSpec(ctx, id, solution)
		if err != nil {
			return observ_utils.CloseSpanWithCOAResponse(span, v1alpha2.COAResponse{
				State: v1alpha2.InternalError,
				Body:  []byte(err.Error()),
			})
		}
		return observ_utils.CloseSpanWithCOAResponse(span, v1alpha2.COAResponse{
			State: v1alpha2.OK,
		})
	case fasthttp.MethodDelete:
		ctx, span := observability.StartSpan("onSolutions-DELETE", pCtx, nil)
		id := request.Parameters["__name"]
		err := c.SolutionsManager.DeleteSpec(ctx, id)
		if err != nil {
			return observ_utils.CloseSpanWithCOAResponse(span, v1alpha2.COAResponse{
				State: v1alpha2.InternalError,
				Body:  []byte(err.Error()),
			})
		}
		return observ_utils.CloseSpanWithCOAResponse(span, v1alpha2.COAResponse{
			State: v1alpha2.OK,
		})
	}
	resp := v1alpha2.COAResponse{
		State:       v1alpha2.MethodNotAllowed,
		Body:        []byte("{\"result\":\"405 - method not allowed\"}"),
		ContentType: "application/json",
	}
	observ_utils.UpdateSpanStatusFromCOAResponse(span, resp)
	span.End()
	return resp
}