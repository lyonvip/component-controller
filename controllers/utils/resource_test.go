package utils

import (
	compv1 "component-controller/api/v1"
	"fmt"
	"testing"
)

func TestNewConfigmap(t *testing.T) {
	comp := &compv1.Component{
		Spec: compv1.ComponentSpec{
			Type:           "redis",
			EnableNodePort: false,
		},
	}
	comp.Namespace = "component"
	yamlBytes := ParseTemplate("service", comp)
	fmt.Println(string(yamlBytes))
}

func TestNewSecret(t *testing.T) {
	comp := &compv1.Component{
		Spec: compv1.ComponentSpec{
			Type:           "redis",
			EnableNodePort: false,
			LoginUser:      "prod",
			LoginPass:      "prod123456",
		},
	}
	comp.Namespace = "component"
	yamlBytes := ParseTemplate("secret", comp)
	fmt.Println(string(yamlBytes))
}
