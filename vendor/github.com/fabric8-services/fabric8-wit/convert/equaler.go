package convert

// The Equaler interface allows object that implement it, to be compared
type Equaler interface {
	// Returns true if the Equaler object is the same as this object; otherwise false is returned.
	// You need to convert the Equaler object using type assertions: https://golang.org/ref/spec#Type_assertions
	Equal(Equaler) bool
}

// DummyEqualer implements the Equaler interface and can be used by tests.
// Other than that it has not meaning.
type DummyEqualer struct {
}

// Ensure DummyEqualer implements the Equaler interface
var _ Equaler = DummyEqualer{}
var _ Equaler = (*DummyEqualer)(nil)

// Equal returns true if the argument is also an DummyEqualer; otherwise false is returned.
func (d DummyEqualer) Equal(u Equaler) bool {
	_, ok := u.(DummyEqualer)
	return ok
}
