// Package query This package implements a super basic parser that takes a string, converts it to json and
// constructs an AndExpression of "key == value" expressions of all the fields in the json object
// this is just a stand-in until we have defined a proper query language
package query

import (
	"encoding/json"

	. "github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/pkg/errors"
)

// Parse parses strings of the form { "attribute1":value1,"attribute2":value2} into an expression of the form "attribute1=value1 and attribute2=value2"
// returns the expression "true" if empty
func Parse(exp *string) (Expression, error) {
	if exp == nil || len(*exp) == 0 {
		return Literal(true), nil
	}
	var unmarshalled map[string]interface{}
	err := json.Unmarshal([]byte(*exp), &unmarshalled)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var result *Expression
	if len(unmarshalled) > 0 {
		for key, value := range unmarshalled {
			current := Equals(Field(key), Literal(value))
			if result == nil {
				result = &current
			} else {
				current = And(*result, current)
				result = &current
			}
		}
		return *result, nil
	}
	return Literal(true), nil
}
