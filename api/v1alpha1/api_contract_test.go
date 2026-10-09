// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"reflect"
	"sort"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
)

func TestSchemeExposesOnlySupportedCustomResources(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := AddToScheme(scheme); err != nil {
		t.Fatalf("register API types: %v", err)
	}

	apiPackage := reflect.TypeOf(Switch{}).PkgPath()
	var got []string
	for gvk, objectType := range scheme.AllKnownTypes() {
		if gvk.GroupVersion() != SchemeGroupVersion {
			continue
		}
		if objectType.PkgPath() == apiPackage {
			got = append(got, gvk.Kind)
		}
	}
	sort.Strings(got)

	want := []string{
		"Switch",
		"SwitchInterface",
		"SwitchInterfaceList",
		"SwitchList",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected registered API kinds: got %v, want %v", got, want)
	}
}

func TestManagementExposesOnlyHostAndPort(t *testing.T) {
	managementType := reflect.TypeOf(Management{})
	want := []struct {
		name    string
		jsonTag string
	}{
		{name: "Host", jsonTag: "host"},
		{name: "Port", jsonTag: "port"},
	}

	if managementType.NumField() != len(want) {
		t.Fatalf("unexpected number of Management fields: got %d, want %d", managementType.NumField(), len(want))
	}
	for i, expected := range want {
		field := managementType.Field(i)
		if field.Name != expected.name || field.Tag.Get("json") != expected.jsonTag {
			t.Errorf("unexpected Management field %d: got %s with JSON tag %q, want %s with JSON tag %q", i, field.Name, field.Tag.Get("json"), expected.name, expected.jsonTag)
		}
	}
}
