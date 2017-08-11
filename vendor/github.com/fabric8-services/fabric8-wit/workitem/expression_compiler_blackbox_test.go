package workitem_test

import (
	"reflect"
	"runtime/debug"
	"testing"

	. "github.com/fabric8-services/fabric8-wit/criteria"
	"github.com/fabric8-services/fabric8-wit/resource"
	. "github.com/fabric8-services/fabric8-wit/workitem"
	"github.com/stretchr/testify/assert"
)

func TestField(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	expect(t, Equals(Field("foo"), Literal(23)), "(Fields@>'{\"foo\" : 23}')", []interface{}{})
	expect(t, Equals(Field("Type"), Literal("abcd")), "(type = ?)", []interface{}{"abcd"})
	expect(t, Not(Field("Type"), Literal("abcd")), "(type != ?)", []interface{}{"abcd"})
	expect(t, Not(Field("Version"), Literal("abcd")), "(version != ?)", []interface{}{"abcd"})
	expect(t, Not(Field("Number"), Literal("abcd")), "(number != ?)", []interface{}{"abcd"})
	expect(t, Not(Field("SpaceID"), Literal("abcd")), "(space_id != ?)", []interface{}{"abcd"})
}

func TestAndOr(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	expect(t, Or(Literal(true), Literal(false)), "(? or ?)", []interface{}{true, false})

	expect(t, And(Not(Field("foo"), Literal("abcd")), Not(Literal(true), Literal(false))), "(NOT (Fields@>'{\"foo\" : \"abcd\"}') and (? != ?))", []interface{}{true, false})
	expect(t, And(Equals(Field("foo"), Literal("abcd")), Equals(Literal(true), Literal(false))), "((Fields@>'{\"foo\" : \"abcd\"}') and (? = ?))", []interface{}{true, false})
	expect(t, Or(Equals(Field("foo"), Literal("abcd")), Equals(Literal(true), Literal(false))), "((Fields@>'{\"foo\" : \"abcd\"}') or (? = ?))", []interface{}{true, false})
}

func TestIsNull(t *testing.T) {
	t.Parallel()
	resource.Require(t, resource.UnitTest)
	expect(t, IsNull("system.assignees"), "(Fields->>'system.assignees' IS NULL)", []interface{}{})
	expect(t, IsNull("ID"), "(id IS NULL)", []interface{}{})
	expect(t, IsNull("Type"), "(type IS NULL)", []interface{}{})
	expect(t, IsNull("Version"), "(version IS NULL)", []interface{}{})
	expect(t, IsNull("Number"), "(number IS NULL)", []interface{}{})
	expect(t, IsNull("SpaceID"), "(space_id IS NULL)", []interface{}{})
}

func expect(t *testing.T, expr Expression, expectedClause string, expectedParameters []interface{}) {
	clause, parameters, err := Compile(expr)
	if len(err) > 0 {
		debug.PrintStack()
		t.Fatal(err[0].Error())
	}
	if clause != expectedClause {
		debug.PrintStack()
		t.Fatalf("clause should be %s but is %s", expectedClause, clause)
	}

	if !reflect.DeepEqual(expectedParameters, parameters) {
		debug.PrintStack()
		t.Fatalf("parameters should be %v but is %v", expectedParameters, parameters)
	}
}

func TestArray(t *testing.T) {
	assignees := []string{"1", "2", "3"}

	exp := Equals(Field("system.assignees"), Literal(assignees))
	where, _, _ := Compile(exp)

	assert.Equal(t, "(Fields@>'{\"system.assignees\" : [\"1\",\"2\",\"3\"]}')", where)
}
