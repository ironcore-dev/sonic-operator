// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ztp

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	networkingv1alpha1 "github.com/ironcore-dev/sonic-operator/api/v1alpha1"
)

const matchingSwitchRemoteAddr = "192.0.2.10:1234"

func TestConfigMapHandlerServesMatchingSwitchScript(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := networkingv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&networkingv1alpha1.Switch{
			ObjectMeta: metav1.ObjectMeta{Name: "leaf-01"},
			Spec: networkingv1alpha1.SwitchSpec{
				ZTP: &networkingv1alpha1.ZTP{
					SourceAddress: "192.0.2.10",
					ScriptRef:     &networkingv1alpha1.ZTPConfigMapReference{Namespace: "provisioning", Name: "leaf-01-ztp", Key: "ztp.sh"},
				},
				Containers: []networkingv1alpha1.Container{{
					Name:  "ignored-in-configmap-mode",
					Image: "example.invalid/ignored:latest",
				}},
			},
		},
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Namespace: "provisioning", Name: "leaf-01-ztp"},
			Data:       map[string]string{"ztp.sh": "#!/bin/bash\necho configured\n"},
		},
	).Build()

	mux := http.NewServeMux()
	RegisterConfigMap(mux, c)
	req := httptest.NewRequest(http.MethodGet, "http://provisioning.example/ztp", nil)
	req.RemoteAddr = matchingSwitchRemoteAddr
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if got, want := response.Body.String(), "#!/bin/bash\necho configured\n"; got != want {
		t.Errorf("body = %q, want %q", got, want)
	}
	if got, want := response.Header().Get("Content-Type"), "text/x-shellscript; charset=utf-8"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

func TestConfigMapHandlerDoesNotFallBackForUnknownSwitch(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := networkingv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	RegisterConfigMap(mux, fake.NewClientBuilder().WithScheme(scheme).Build())
	req := httptest.NewRequest(http.MethodGet, "http://provisioning.example/ztp", nil)
	req.RemoteAddr = "192.0.2.99:1234"
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, req)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if got := response.Body.String(); got == "" {
		t.Error("expected an error body for an unknown switch")
	}
}

func TestGeneratedHandlerRendersCompleteSwitchScript(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := networkingv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	kubeconfigPath := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(kubeconfigPath, []byte("apiVersion: v1\nkind: Config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	wireletUID := int64(65532)
	agentUID := int64(0)

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&networkingv1alpha1.Switch{
			ObjectMeta: metav1.ObjectMeta{Name: "switch-1"},
			Spec: networkingv1alpha1.SwitchSpec{
				Hostname: "switch-1.lab.example",
				ZTP: &networkingv1alpha1.ZTP{
					SourceAddress: "192.0.2.10",
				},
				Volumes: []networkingv1alpha1.Volume{
					{Name: "dbus", HostPath: &networkingv1alpha1.HostPathVolumeSource{Path: "/var/run/dbus"}},
					{Name: "sonic-version", HostPath: &networkingv1alpha1.HostPathVolumeSource{Path: "/etc/sonic/sonic_version.yml"}},
				},
				Containers: []networkingv1alpha1.Container{
					{
						Name:                    "wirelet",
						Image:                   "ghcr.io/hardikdr/wirelet:fixed-1",
						SecurityContext:         &networkingv1alpha1.ContainerSecurityContext{RunAsUser: &wireletUID, RunAsGroup: &wireletUID},
						Args:                    []string{"--name=switch-1", "--interface=Ethernet0"},
						InjectControlKubeconfig: true,
					},
					{
						Name:    "sonic-agent",
						Image:   "ghcr.io/giluerre/sonic-agent:latest",
						Command: []string{"/switch-agent-server"},
						Args:    []string{"-port", "57400"},
						SecurityContext: &networkingv1alpha1.ContainerSecurityContext{
							RunAsUser: &agentUID,
						},
						VolumeMounts: []networkingv1alpha1.VolumeMount{
							{Name: "dbus", MountPath: "/var/run/dbus"},
							{Name: "sonic-version", MountPath: "/etc/sonic/sonic_version.yml", ReadOnly: true},
						},
					},
				},
				NextBootMode: networkingv1alpha1.NextBootModeInstallOS,
			},
		},
	).Build()

	mux := http.NewServeMux()
	RegisterGenerated(mux, c, GeneratedOptions{ControlKubeconfigFile: kubeconfigPath})
	req := httptest.NewRequest(http.MethodGet, "http://provisioning.example/ztp", nil)
	req.RemoteAddr = matchingSwitchRemoteAddr
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, req)

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	body := response.Body.String()
	wantBody, err := os.ReadFile("testdata/generated-mode.golden.sh")
	if err != nil {
		t.Fatal(err)
	}
	if body != string(wantBody) {
		t.Errorf("rendered script differs from testdata/generated-mode.golden.sh\ngot:\n%s\nwant:\n%s", body, wantBody)
	}
	for _, want := range []string{
		"#!/bin/bash",
		"set -euo pipefail",
		"config hostname 'switch-1.lab.example'",
		"config save -y",
		"printf '%s' 'YXBpVmVyc2lvbjogdjEKa2luZDogQ29uZmlnCg==' | base64 -d >'/etc/sonic-operator/credentials/wirelet/control-kubeconfig'",
		"chown 65532:65532 '/etc/sonic-operator/credentials/wirelet/control-kubeconfig'",
		"chmod 0600 '/etc/sonic-operator/credentials/wirelet/control-kubeconfig'",
		"docker pull 'ghcr.io/hardikdr/wirelet:fixed-1'",
		"docker run -d --name 'wirelet' --network host --restart unless-stopped --user '65532:65532'",
		"-e KUBECONFIG=/var/run/sonic-operator/control-kubeconfig",
		"-v '/etc/sonic-operator/credentials/wirelet/control-kubeconfig:/var/run/sonic-operator/control-kubeconfig:ro'",
		"'--name=switch-1' '--interface=Ethernet0'",
		"docker run -d --name 'sonic-agent' --network host --restart unless-stopped --user '0' -v '/var/run/dbus:/var/run/dbus:rw' -v '/etc/sonic/sonic_version.yml:/etc/sonic/sonic_version.yml:ro' --entrypoint '/switch-agent-server' 'ghcr.io/giluerre/sonic-agent:latest' '-port' '57400'",
		"What=LABEL=ONIE-BOOT",
		"sonic-operator-onie-install.service",
		"set next_entry=ONIE",
		"set onie_mode=install",
	} {
		if !strings.Contains(body, want) {
			t.Errorf("response does not contain %q:\n%s", want, body)
		}
	}
	if strings.Index(body, "# Configure ONIE install discovery") > strings.Index(body, "config hostname") {
		t.Error("ONIE boot lifecycle configuration must be rendered before hostname configuration")
	}
	if !strings.Contains(body, "sonic-operator: container wirelet failed; continuing") {
		t.Errorf("response does not isolate bootstrap-container failures:\n%s", body)
	}

	scriptPath := filepath.Join(t.TempDir(), "generated-ztp.sh")
	if err := os.WriteFile(scriptPath, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("bash", "-n", scriptPath).CombinedOutput(); err != nil {
		t.Fatalf("generated script is not valid Bash: %v: %s", err, output)
	}
}

func TestGeneratedHandlerRejectsMissingControlKubeconfig(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := networkingv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}

	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(
		&networkingv1alpha1.Switch{
			ObjectMeta: metav1.ObjectMeta{Name: "switch-1"},
			Spec: networkingv1alpha1.SwitchSpec{
				ZTP: &networkingv1alpha1.ZTP{
					SourceAddress: "192.0.2.10",
				},
				Containers: []networkingv1alpha1.Container{{
					Name:                    "wirelet",
					Image:                   "example.invalid/wirelet:latest",
					InjectControlKubeconfig: true,
				}},
			},
		},
	).Build()

	mux := http.NewServeMux()
	RegisterGenerated(mux, c, GeneratedOptions{})
	req := httptest.NewRequest(http.MethodGet, "http://provisioning.example/ztp", nil)
	req.RemoteAddr = matchingSwitchRemoteAddr
	response := httptest.NewRecorder()

	mux.ServeHTTP(response, req)

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d: %s", response.Code, http.StatusInternalServerError, response.Body.String())
	}
	if !strings.Contains(response.Body.String(), "bootstrap-control-kubeconfig-file") {
		t.Errorf("response = %q, want missing kubeconfig error", response.Body.String())
	}
}

func TestRenderGeneratedScriptDefaultsHostnameToSwitchName(t *testing.T) {
	switchObject := &networkingv1alpha1.Switch{
		ObjectMeta: metav1.ObjectMeta{Name: "leaf-1"},
	}

	script, err := renderGeneratedScript(switchObject, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(script, "config hostname 'leaf-1'") {
		t.Errorf("generated script does not default hostname to switch name:\n%s", script)
	}
}
