// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package ztp

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"

	networkingv1alpha1 "github.com/ironcore-dev/sonic-operator/api/v1alpha1"
)

var generatedModeTemplate = template.Must(
	template.New("generated-mode.sh.gotmpl").
		Funcs(template.FuncMap{"shellQuote": shellQuote}).
		ParseFS(templateFS, "templates/generated-mode.sh.gotmpl"),
)

type generatedModeTemplateData struct {
	SwitchName    string
	Hostname      string
	ConfigureONIE bool
	Containers    []generatedModeContainerData
}

type generatedModeContainerData struct {
	Name                     string
	Image                    string
	DockerUser               string
	VolumeMounts             []string
	InjectControlKubeconfig  bool
	EncodedControlKubeconfig string
	ControlKubeconfigPath    string
	Entrypoint               string
	Command                  []string
	Args                     []string
	FailureMessage           string
}

func renderGeneratedScript(switchObject *networkingv1alpha1.Switch, controlKubeconfigFilePath string) (string, error) {
	data, err := buildGeneratedModeTemplateData(switchObject, controlKubeconfigFilePath)
	if err != nil {
		return "", err
	}

	var output bytes.Buffer
	if err := generatedModeTemplate.ExecuteTemplate(&output, "generated-mode.sh.gotmpl", data); err != nil {
		return "", fmt.Errorf("executing generated mode template: %w", err)
	}
	return output.String(), nil
}

func buildGeneratedModeTemplateData(
	switchObject *networkingv1alpha1.Switch,
	controlKubeconfigFilePath string,
) (*generatedModeTemplateData, error) {
	hostname := switchObject.Spec.Hostname
	if hostname == "" {
		hostname = switchObject.Name
	}
	if hostname == "" {
		return nil, fmt.Errorf("hostname is empty")
	}

	data := &generatedModeTemplateData{
		SwitchName: switchObject.Name,
		Hostname:   hostname,
		Containers: make([]generatedModeContainerData, 0, len(switchObject.Spec.Containers)),
	}

	// Boot lifecycle intent is strict and appears before fallible switch or
	// workload configuration in the rendered script.
	switch switchObject.Spec.NextBootMode {
	case "", networkingv1alpha1.NextBootModeNone:
	case networkingv1alpha1.NextBootModeInstallOS:
		data.ConfigureONIE = true
	default:
		return nil, fmt.Errorf("unsupported next boot mode %q", switchObject.Spec.NextBootMode)
	}

	var volumes map[string]string
	if len(switchObject.Spec.Containers) > 0 {
		var err error
		volumes, err = hostPathVolumes(switchObject.Spec.Volumes)
		if err != nil {
			return nil, err
		}
	}

	var encodedControlKubeconfig string
	for _, container := range switchObject.Spec.Containers {
		if !container.InjectControlKubeconfig || encodedControlKubeconfig != "" {
			continue
		}
		if controlKubeconfigFilePath == "" {
			return nil, fmt.Errorf("injectControlKubeconfig requires --bootstrap-control-kubeconfig-file")
		}
		kubeconfig, err := os.ReadFile(controlKubeconfigFilePath)
		if err != nil {
			return nil, fmt.Errorf("reading control kubeconfig file %q: %w", controlKubeconfigFilePath, err)
		}
		encodedControlKubeconfig = base64.StdEncoding.EncodeToString(kubeconfig)
	}

	for _, container := range switchObject.Spec.Containers {
		dockerUser, err := containerDockerUser(container)
		if err != nil {
			return nil, err
		}
		volumeMounts, err := containerVolumeMounts(container, volumes)
		if err != nil {
			return nil, err
		}

		containerData := generatedModeContainerData{
			Name:                     container.Name,
			Image:                    container.Image,
			DockerUser:               dockerUser,
			VolumeMounts:             volumeMounts,
			InjectControlKubeconfig:  container.InjectControlKubeconfig,
			EncodedControlKubeconfig: encodedControlKubeconfig,
			Args:                     container.Args,
			FailureMessage:           "sonic-operator: container " + container.Name + " failed; continuing",
		}
		if len(container.Command) > 0 {
			containerData.Entrypoint = container.Command[0]
			containerData.Command = container.Command[1:]
		}
		if container.InjectControlKubeconfig {
			if container.SecurityContext == nil || container.SecurityContext.RunAsUser == nil {
				return nil, fmt.Errorf("container %q injects the control kubeconfig but has no securityContext.runAsUser", container.Name)
			}
			containerData.ControlKubeconfigPath = "/etc/sonic-operator/credentials/" + container.Name + "/control-kubeconfig"
		}
		data.Containers = append(data.Containers, containerData)
	}

	return data, nil
}

func containerDockerUser(container networkingv1alpha1.Container) (string, error) {
	if container.SecurityContext == nil {
		return "", nil
	}

	securityContext := container.SecurityContext
	if securityContext.RunAsGroup != nil && securityContext.RunAsUser == nil {
		return "", fmt.Errorf("container %q sets securityContext.runAsGroup without securityContext.runAsUser", container.Name)
	}
	if securityContext.RunAsUser == nil {
		return "", nil
	}

	user := strconv.FormatInt(*securityContext.RunAsUser, 10)
	if securityContext.RunAsGroup == nil {
		return user, nil
	}
	return user + ":" + strconv.FormatInt(*securityContext.RunAsGroup, 10), nil
}

func hostPathVolumes(volumes []networkingv1alpha1.Volume) (map[string]string, error) {
	result := make(map[string]string, len(volumes))
	for _, volume := range volumes {
		if volume.Name == "" {
			return nil, fmt.Errorf("volume name is empty")
		}
		if _, exists := result[volume.Name]; exists {
			return nil, fmt.Errorf("duplicate volume %q", volume.Name)
		}
		if volume.HostPath == nil {
			return nil, fmt.Errorf("volume %q must specify hostPath", volume.Name)
		}
		if !strings.HasPrefix(volume.HostPath.Path, "/") {
			return nil, fmt.Errorf("volume %q hostPath %q must be absolute", volume.Name, volume.HostPath.Path)
		}
		result[volume.Name] = volume.HostPath.Path
	}
	return result, nil
}

func containerVolumeMounts(container networkingv1alpha1.Container, volumes map[string]string) ([]string, error) {
	result := make([]string, 0, len(container.VolumeMounts))
	mountPaths := make(map[string]struct{}, len(container.VolumeMounts))
	for _, mount := range container.VolumeMounts {
		hostPath, exists := volumes[mount.Name]
		if !exists {
			return nil, fmt.Errorf("container %q mounts unknown volume %q", container.Name, mount.Name)
		}
		if !strings.HasPrefix(mount.MountPath, "/") {
			return nil, fmt.Errorf("container %q mountPath %q must be absolute", container.Name, mount.MountPath)
		}
		if _, exists := mountPaths[mount.MountPath]; exists {
			return nil, fmt.Errorf("container %q mounts multiple volumes at %q", container.Name, mount.MountPath)
		}
		mountPaths[mount.MountPath] = struct{}{}
		mode := "rw"
		if mount.ReadOnly {
			mode = "ro"
		}
		result = append(result, hostPath+":"+mount.MountPath+":"+mode)
	}
	return result, nil
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\\"'\\\"'") + "'"
}
