// +build !ignore_autogenerated

// Code generated by openapi-gen. DO NOT EDIT.

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1alpha1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"ring-operator/pkg/apis/rings/v1alpha1.Ring":       schema_pkg_apis_rings_v1alpha1_Ring(ref),
		"ring-operator/pkg/apis/rings/v1alpha1.RingSpec":   schema_pkg_apis_rings_v1alpha1_RingSpec(ref),
		"ring-operator/pkg/apis/rings/v1alpha1.RingStatus": schema_pkg_apis_rings_v1alpha1_RingStatus(ref),
	}
}

func schema_pkg_apis_rings_v1alpha1_Ring(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "Ring is the Schema for the rings API",
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("ring-operator/pkg/apis/rings/v1alpha1.RingSpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("ring-operator/pkg/apis/rings/v1alpha1.RingStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta", "ring-operator/pkg/apis/rings/v1alpha1.RingSpec", "ring-operator/pkg/apis/rings/v1alpha1.RingStatus"},
	}
}

func schema_pkg_apis_rings_v1alpha1_RingSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "RingSpec defines the desired state of Ring",
				Properties: map[string]spec.Schema{
					"deploy": {
						SchemaProps: spec.SchemaProps{
							Description: "Deploy marks whether this ring will be deployed to the live environment",
							Type:        []string{"boolean"},
							Format:      "",
						},
					},
					"routing": {
						SchemaProps: spec.SchemaProps{
							Description: "Routing describes the service, group and users to be included in the ring",
							Ref:         ref("ring-operator/pkg/apis/rings/v1alpha1.RingRouting"),
						},
					},
				},
				Required: []string{"deploy", "routing"},
			},
		},
		Dependencies: []string{
			"ring-operator/pkg/apis/rings/v1alpha1.RingRouting"},
	}
}

func schema_pkg_apis_rings_v1alpha1_RingStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "RingStatus defines the observed state of Ring",
				Properties:  map[string]spec.Schema{},
			},
		},
		Dependencies: []string{},
	}
}
