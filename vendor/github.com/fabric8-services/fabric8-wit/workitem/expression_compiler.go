package workitem

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/fabric8-services/fabric8-wit/criteria"
	uuid "github.com/satori/go.uuid"
)

const (
	jsonAnnotation = "JSON"
)

// Compile takes an expression and compiles it to a where clause for use with gorm.DB.Where()
// Returns the number of expected parameters for the query and a slice of errors if something goes wrong
func Compile(where criteria.Expression) (whereClause string, parameters []interface{}, err []error) {
	criteria.IteratePostOrder(where, bubbleUpJSONContext)

	compiler := newExpressionCompiler()
	compiled := where.Accept(&compiler)

	return compiled.(string), compiler.parameters, compiler.err
}

// mark expression tree nodes that reference json fields
func bubbleUpJSONContext(exp criteria.Expression) bool {
	switch t := exp.(type) {
	case *criteria.FieldExpression:
		_, isJSONField := getFieldName(t.FieldName)
		if isJSONField {
			t.SetAnnotation(jsonAnnotation, true)
		}
	case *criteria.EqualsExpression:
		if t.Left().Annotation(jsonAnnotation) == true || t.Right().Annotation(jsonAnnotation) == true {
			t.SetAnnotation(jsonAnnotation, true)
		}
	case *criteria.NotExpression:
		if t.Left().Annotation(jsonAnnotation) == true || t.Right().Annotation(jsonAnnotation) == true {
			t.SetAnnotation(jsonAnnotation, true)
		}
	}
	return true
}

// fieldMap tells how to resolve struct fields as SQL fields in the work_items
// SQL table.
// NOTE: anything not listed here will be treated as if it is nested inside the
// jsonb "fields" column.
var fieldMap = map[string]string{
	"ID":      "id",
	"Type":    "type",
	"Version": "version",
	"Number":  "number",
	"SpaceID": "space_id",
}

// getFieldName applies any potentially necessary mapping to field names (e.g.
// SpaceID -> space_id) and tells if the field is stored inside the jsonb column
// (last result is true then) or as a normal column.
func getFieldName(fieldName string) (mappedFieldName string, isJSONField bool) {
	mappedFieldName, isColumnField := fieldMap[fieldName]
	if isColumnField {
		return mappedFieldName, false
	}
	// leave field untouched
	return fieldName, true
}

func newExpressionCompiler() expressionCompiler {
	return expressionCompiler{parameters: []interface{}{}}
}

// expressionCompiler takes an expression and compiles it to a where clause for our gorm models
// implements criteria.ExpressionVisitor
type expressionCompiler struct {
	parameters []interface{} // records the number of parameter expressions encountered
	err        []error       // record any errors found in the expression
}

// visitor implementation
// the convention is to return nil when the expression cannot be compiled and to append an error to the err field

func (c *expressionCompiler) Field(f *criteria.FieldExpression) interface{} {
	mappedFieldName, isJSONField := getFieldName(f.FieldName)
	if !isJSONField {
		return mappedFieldName
	}
	if strings.Contains(mappedFieldName, "'") {
		// beware of injection, it's a reasonable restriction for field names,
		// make sure it's not allowed when creating wi types
		c.err = append(c.err, fmt.Errorf("single quote not allowed in field name"))
		return nil
	}
	return "Fields@>'{\"" + mappedFieldName + "\""
}

func (c *expressionCompiler) And(a *criteria.AndExpression) interface{} {
	return c.binary(a, "and")
}

func (c *expressionCompiler) binary(a criteria.BinaryExpression, op string) interface{} {
	left := a.Left().Accept(c)
	right := a.Right().Accept(c)
	if left != nil && right != nil {
		return "(" + left.(string) + " " + op + " " + right.(string) + ")"
	}
	// something went wrong in either compilation, errors have been accumulated
	return nil
}

func (c *expressionCompiler) Or(a *criteria.OrExpression) interface{} {
	return c.binary(a, "or")
}

func (c *expressionCompiler) Equals(e *criteria.EqualsExpression) interface{} {
	if isInJSONContext(e.Left()) {
		return c.binary(e, ":")
	}
	return c.binary(e, "=")
}

func (c *expressionCompiler) IsNull(e *criteria.IsNullExpression) interface{} {
	mappedFieldName, isJSONField := getFieldName(e.FieldName)
	if isJSONField {
		return "(Fields->>'" + mappedFieldName + "' IS NULL)"
	}
	return "(" + mappedFieldName + " IS NULL)"
}

func (c *expressionCompiler) Not(e *criteria.NotExpression) interface{} {
	if isInJSONContext(e.Left()) {
		condition := c.binary(e, ":")
		if condition != nil {
			return "NOT " + condition.(string)
		}
		return nil
	}
	return c.binary(e, "!=")
}

func (c *expressionCompiler) Parameter(v *criteria.ParameterExpression) interface{} {
	c.err = append(c.err, fmt.Errorf("Parameter expression not supported"))
	return nil
}

// iterate the parent chain to see if this expression references json fields
func isInJSONContext(exp criteria.Expression) bool {
	result := false
	criteria.IterateParents(exp, func(exp criteria.Expression) bool {
		if exp.Annotation(jsonAnnotation) == true {
			result = true
			return false
		}
		return true
	})
	return result
}

// literal values need to be converted differently depending on whether they are used in a JSON context or a regular SQL expression.
// JSON values are always strings (delimited with "'"), but operators can be used depending on the dynamic type. For example,
// you can write "a->'foo' < '5'" and it will return true for the json object { "a": 40 }.
func (c *expressionCompiler) Literal(v *criteria.LiteralExpression) interface{} {
	json := isInJSONContext(v)
	if json {
		stringVal, err := c.convertToString(v.Value)
		if err == nil {
			return stringVal + "}'"
		}
		if stringArr, ok := v.Value.([]string); ok {
			return "[" + c.wrapStrings(stringArr) + "]}'"
		}
		c.err = append(c.err, err)
		return nil
	}
	c.parameters = append(c.parameters, v.Value)
	return "?"
}

func (c *expressionCompiler) wrapStrings(value []string) string {
	wrapped := []string{}
	for i := 0; i < len(value); i++ {
		wrapped = append(wrapped, "\""+value[i]+"\"")
	}
	return strings.Join(wrapped, ",")
}

func (c *expressionCompiler) convertToString(value interface{}) (string, error) {
	var result string
	switch t := value.(type) {
	case float64:
		result = strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		result = strconv.Itoa(t)
	case int64:
		result = strconv.FormatInt(t, 10)
	case uint:
		result = strconv.FormatUint(uint64(t), 10)
	case uint64:
		result = strconv.FormatUint(t, 10)
	case string:
		result = "\"" + t + "\""
	case bool:
		result = strconv.FormatBool(t)
	case uuid.UUID:
		result = t.String()
	default:
		return "", fmt.Errorf("unknown value type of %v: %T", value, value)
	}
	return result, nil
}
