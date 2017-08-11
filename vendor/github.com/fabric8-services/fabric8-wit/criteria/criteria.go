package criteria

// Expression is used to express conditions for selecting an entity
type Expression interface {
	// Accept calls the visitor callback of the appropriate type
	Accept(visitor ExpressionVisitor) interface{}
	// SetAnnotation puts the given annotation on the expression
	SetAnnotation(key string, value interface{})
	// Annotation reads back values set with SetAnnotation
	Annotation(key string) interface{}
	// Returns the parent expression or nil
	Parent() Expression
	setParent(parent Expression)
}

// IterateParents calls f for every member of the parent chain
// Stops iterating if f returns false
func IterateParents(exp Expression, f func(Expression) bool) {
	if exp != nil {
		exp = exp.Parent()
	}
	for exp != nil {
		if !f(exp) {
			return
		}
		exp = exp.Parent()
	}
}

// BinaryExpression represents expressions with 2 children
// This could be generalized to n-ary expressions, but that is not necessary right now
type BinaryExpression interface {
	Expression
	Left() Expression
	Right() Expression
}

// ExpressionVisitor is an implementation of the visitor pattern for expressions
type ExpressionVisitor interface {
	Field(t *FieldExpression) interface{}
	And(a *AndExpression) interface{}
	Or(a *OrExpression) interface{}
	Equals(e *EqualsExpression) interface{}
	Parameter(v *ParameterExpression) interface{}
	Literal(c *LiteralExpression) interface{}
	Not(e *NotExpression) interface{}
	IsNull(e *IsNullExpression) interface{}
}

type expression struct {
	parent      Expression
	annotations map[string]interface{}
}

func (exp *expression) SetAnnotation(key string, value interface{}) {
	if exp.annotations == nil {
		exp.annotations = map[string]interface{}{}
	}
	exp.annotations[key] = value
}

func (exp *expression) Annotation(key string) interface{} {
	return exp.annotations[key]
}

func (exp *expression) Parent() Expression {
	result := exp.parent
	return result
}

func (exp *expression) setParent(parent Expression) {
	exp.parent = parent
}

// access a Field

// FieldExpression represents access to a field of the tested object
type FieldExpression struct {
	expression
	FieldName string
}

// Accept implements ExpressionVisitor
func (t *FieldExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Field(t)
}

// Field constructs a FieldExpression
func Field(id string) Expression {
	return &FieldExpression{expression{}, id}
}

// Parameter (free variable of the expression)

// A ParameterExpression represents a parameter to be passed upon evaluation of the expression
type ParameterExpression struct {
	expression
}

// Accept implements ExpressionVisitor
func (t *ParameterExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Parameter(t)
}

// Parameter constructs a value expression.
func Parameter() Expression {
	return &ParameterExpression{}
}

// literal value

// A LiteralExpression represents a single constant value in the expression, think "5" or "asdf"
// the type of literals is not restricted at this level, but compilers or interpreters will have limitations on what they handle
type LiteralExpression struct {
	expression
	Value interface{}
}

// Accept implements ExpressionVisitor
func (t *LiteralExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Literal(t)
}

// Literal constructs a literal expression
func Literal(value interface{}) Expression {
	return &LiteralExpression{expression{}, value}
}

// binaryExpression is an "abstract" type for binary expressions.
type binaryExpression struct {
	expression
	left  Expression
	right Expression
}

// Left implements BinaryExpression
func (exp *binaryExpression) Left() Expression {
	return exp.left
}

// Right implements BinaryExpression
func (exp *binaryExpression) Right() Expression {
	return exp.right
}

// make sure the children have the correct parent
func reparent(parent BinaryExpression) Expression {
	parent.Left().setParent(parent)
	parent.Right().setParent(parent)
	return parent
}

// And

// AndExpression represents the conjunction operation of two terms
type AndExpression struct {
	binaryExpression
}

// Accept implements ExpressionVisitor
func (t *AndExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.And(t)
}

// And constructs an AndExpression
func And(left Expression, right Expression) Expression {
	return reparent(&AndExpression{binaryExpression{expression{}, left, right}})
}

// Or

// OrExpression represents the disjunction operation of two terms
type OrExpression struct {
	binaryExpression
}

// Accept implements ExpressionVisitor
func (t *OrExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Or(t)
}

// Or constructs an OrExpression
func Or(left Expression, right Expression) Expression {
	return reparent(&OrExpression{binaryExpression{expression{}, left, right}})
}

// ==

// EqualsExpression represents the equality operator
type EqualsExpression struct {
	binaryExpression
}

// Accept implements ExpressionVisitor
func (t *EqualsExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Equals(t)
}

// Equals constructs an EqualsExpression
func Equals(left Expression, right Expression) Expression {
	return reparent(&EqualsExpression{binaryExpression{expression{}, left, right}})
}

// IS NULL

// IsNullExpression represents the IS operator with NULL value
type IsNullExpression struct {
	expression
	FieldName string
}

// IsNull constructs an NullExpression
func IsNull(name string) Expression {
	return &IsNullExpression{expression{}, name}
}

// Accept implements ExpressionVisitor
func (t *IsNullExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.IsNull(t)
}

// Not

// NotExpression represents the negation operator
type NotExpression struct {
	binaryExpression
}

// Accept implements ExpressionVisitor
func (t *NotExpression) Accept(visitor ExpressionVisitor) interface{} {
	return visitor.Not(t)
}

// Not constructs a NotExpression
func Not(left Expression, right Expression) Expression {
	return reparent(&NotExpression{binaryExpression{expression{}, left, right}})
}
