package openshift

import (
	"bytes"
	"html/template"
	"regexp"
)

// Process takes a K8/Openshift Template as input and resolves the variable expresions
func Process(source string, variables map[string]string) (string, error) {
	target, err := template.New("openshift").Parse(replaceTemplateExpression(source))
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	err = target.Execute(&buf, variables)
	if err != nil {
		return "", err
	}
	str := buf.String()
	return str, nil
}

func replaceTemplateExpression(template string) string {
	reg := regexp.MustCompile(`\${([A-Z_]+)}`)
	return reg.ReplaceAllString(template, "{{.$1}}")
}
