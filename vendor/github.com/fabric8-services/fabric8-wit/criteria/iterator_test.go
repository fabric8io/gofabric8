package criteria

import (
	"reflect"
	"testing"

	"github.com/fabric8-services/fabric8-wit/resource"
)

func TestIterator(t *testing.T) {
	resource.Require(t, resource.UnitTest)

	// test left-to-right, depth first iteration
	visited := []Expression{}
	l := Field("a")
	r := Literal(5)
	expr := Equals(l, r)
	expected := []Expression{l, r, expr}
	recorder := func(expr Expression) bool {
		visited = append(visited, expr)
		return true
	}
	IteratePostOrder(expr, recorder)
	if !reflect.DeepEqual(expected, visited) {
		t.Errorf("Visited should be %v, but is %v", expected, visited)
	}

	// test early iteration cutoff with false return from iterator function
	visited = []Expression{}
	recorder = func(expr Expression) bool {
		visited = append(visited, expr)
		return expr != r
	}
	IteratePostOrder(expr, recorder)
	expected = []Expression{l, r}
	if !reflect.DeepEqual(expected, visited) {
		t.Errorf("Visited should be %v, but is %v", expected, visited)
	}

}
