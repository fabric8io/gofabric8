package remoteworkitem

import (
	"fmt"
	"reflect"
)

// Flatten Takes the nested map and returns a non nested one with dot delimited keys
func Flatten(source map[string]interface{}) map[string]interface{} {
	target := make(map[string]interface{})
	flatten(target, source, nil)
	return target
}

func flatten(target map[string]interface{}, source map[string]interface{}, parent *string) {
	for k, v := range source {
		var key string
		if parent == nil {
			key = k
		} else {
			key = *parent + "." + k
		}

		if v != nil && reflect.TypeOf(v).Kind() == reflect.Map {
			flatten(target, v.(map[string]interface{}), &key)
		} else if v != nil && reflect.TypeOf(v).Kind() == reflect.Slice {
			arrayAsMap := convertArrayToMap(v.([]interface{}))
			flatten(target, arrayAsMap, &key)
		} else {
			target[key] = v
		}
	}
}

func convertArrayToMap(arrayOfObjects []interface{}) map[string]interface{} {
	arrayAsMap := make(map[string]interface{})
	for k, v := range arrayOfObjects {
		arrayAsMap[fmt.Sprintf("%d", k)] = v
	}
	return arrayAsMap
}
