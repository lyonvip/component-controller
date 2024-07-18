package utils

import (
	"bytes"
	compv1 "component-controller/api/v1"
	"component-controller/controllers/static"
	"encoding/base64"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
	"path/filepath"
	"text/template"
)

func ParseTemplate(resourceType string, comp *compv1.Component) []byte {
	filename := filepath.Join(comp.Spec.Type, resourceType+".yaml")
	fileObj, err := static.TemplatesFS.ReadFile(filename)
	if err != nil {
		return nil
	}
	tmpl := template.Must(template.New("").Funcs(template.FuncMap{
		"b64enc": func(s string) string {
			return base64.StdEncoding.EncodeToString([]byte(s))
		},
	}).Parse(string(fileObj)))
	buffer := new(bytes.Buffer)
	if err = tmpl.Execute(buffer, comp); err != nil {
		return nil
	}
	return buffer.Bytes()
}

func NewDeployment(comp *compv1.Component) *appsv1.Deployment {
	deploymentObj := &appsv1.Deployment{}
	if err := yaml.Unmarshal(ParseTemplate("deployment", comp), deploymentObj); err != nil {
		return nil
	}
	return deploymentObj
}

func NewConfigmap(comp *compv1.Component) *corev1.ConfigMap {
	configmapObj := &corev1.ConfigMap{}
	if err := yaml.Unmarshal(ParseTemplate("configmap", comp), configmapObj); err != nil {
		return nil
	}
	return configmapObj
}

func NewSecret(comp *compv1.Component) *corev1.Secret {
	secretObj := &corev1.Secret{}
	if err := yaml.Unmarshal(ParseTemplate("secret", comp), secretObj); err != nil {
		return nil
	}
	return secretObj
}

func NewService(comp *compv1.Component) *corev1.Service {
	serviceObj := &corev1.Service{}
	if err := yaml.Unmarshal(ParseTemplate("service", comp), serviceObj); err != nil {
		return nil
	}
	return serviceObj
}
