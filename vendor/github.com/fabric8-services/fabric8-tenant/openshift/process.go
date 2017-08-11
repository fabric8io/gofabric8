package openshift

import "regexp"

// Process takes a K8/Openshift Template as input and resolves the variable expresions
func Process(source string, variables map[string]string) (string, error) {
	reg := regexp.MustCompile(`\${([A-Z_]+)}`)
	return string(reg.ReplaceAllFunc([]byte(source), func(found []byte) []byte {
		variableName := toVariableName(string(found))
		if variable, ok := variables[variableName]; ok {
			return []byte(variable)
		}
		return found
	})), nil
}

func toVariableName(exp string) string {
	return exp[:len(exp)-1][2:]
}

func replaceTemplateExpression(template string) string {
	reg := regexp.MustCompile(`\${([A-Z_]+)}`)
	return reg.ReplaceAllString(template, "{{.$1}}")
}
