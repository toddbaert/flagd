package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	gen "github.com/open-feature/flagd/pkg/generated"
	provider "github.com/open-feature/flagd/pkg/provider"
)

type HttpServiceConfiguration struct {
	Port int32
}

type HttpService struct {
	HttpServiceConfiguration *HttpServiceConfiguration
}

var defaultReason = "DEFAULT"
var errorReason = "ERROR"

// implement the generated ServerInterface.
type Server struct {
	provider provider.IProvider
}

// TODO: might be able to simplify some of this with generics.
func (s Server) ResolveBoolean(w http.ResponseWriter, r *http.Request, flagKey gen.FlagKey, params gen.ResolveBooleanParams) {
	result, err := s.provider.ResolveBooleanValue(flagKey, params.DefaultValue)
	if (err != nil) {
		fmt.Println(err)
		json.NewEncoder(w).Encode(gen.ResolutionDetailsBoolean{
			Value: &params.DefaultValue,
			Reason: &errorReason,
		})
		return;
	}
	json.NewEncoder(w).Encode(gen.ResolutionDetailsBoolean{
		Value: &result,
		Reason: &defaultReason,
	})
}

func (s Server) ResolveString(w http.ResponseWriter, r *http.Request, flagKey gen.FlagKey, params gen.ResolveStringParams) {
	result, err := s.provider.ResolveStringValue(flagKey, params.DefaultValue)
	if (err != nil) {
		fmt.Println(err)
		json.NewEncoder(w).Encode(gen.ResolutionDetailsString{
			Value: &params.DefaultValue,
			Reason: &errorReason,
		})
		return;
	}
	json.NewEncoder(w).Encode(gen.ResolutionDetailsString{
		Value: &result,
		Reason: &defaultReason,
	})
}

func (s Server) ResolveNumber(w http.ResponseWriter, r *http.Request, flagKey gen.FlagKey, params gen.ResolveNumberParams) {
	result, err := s.provider.ResolveNumberValue(flagKey, params.DefaultValue)
	if (err != nil) {
		fmt.Println(err)
		json.NewEncoder(w).Encode(gen.ResolutionDetailsNumber{
			Value: &params.DefaultValue,
			Reason: &errorReason,
		})
		return;
	}
	json.NewEncoder(w).Encode(gen.ResolutionDetailsNumber{
		Value: &result,
		Reason: &defaultReason,
	})
}

func (s Server) ResolveObject(w http.ResponseWriter, r *http.Request, flagKey gen.FlagKey, params gen.ResolveObjectParams) {
	result, err := s.provider.ResolveObjectValue(flagKey, params.DefaultValue.AdditionalProperties)
	if (err != nil) {
		fmt.Println(err)
		json.NewEncoder(w).Encode(gen.ResolutionDetailsObject{
			Value: &gen.ResolutionDetailsObject_Value{
				AdditionalProperties: params.DefaultValue.AdditionalProperties,
			},
			Reason: &defaultReason,
		})
		return;
	}
	json.NewEncoder(w).Encode(gen.ResolutionDetailsObject{
		Value: &gen.ResolutionDetailsObject_Value{
			AdditionalProperties: result,
		},
		Reason: &defaultReason,
	})


}

func (h *HttpService) Serve(provider provider.IProvider) error {
	if h.HttpServiceConfiguration == nil {
		return errors.New("http service configuration has not been initialised")
	}

	// start with the configured provider.
	provider.Initialize()
	http.Handle("/", gen.Handler(Server{ provider }))
	http.ListenAndServe(fmt.Sprintf(":%d", h.HttpServiceConfiguration.Port), nil)

	return nil
}
