package builder

// Where represents a map of field conditions for queries, similar to Prisma's where clause.
// Each key is a field name, and the value can be either:
//   - A direct value for equality comparison
//   - A WhereOperator for complex comparisons
//   - nil for IS NULL checks
//
// Example:
//
//	where := builder.Where{
//	    "email": "user@example.com",
//	    "age": builder.Gte(18),
//	    "status": builder.In("active", "pending"),
//	    "deleted_at": nil,
//	}
type Where map[string]interface{}

// WhereOperator represents a conditional operator with its value
type WhereOperator struct {
	op    string
	value interface{}
}

// Comparison operators for building WHERE clauses

// Equals creates an equality operator (=)
func Equals(value interface{}) WhereOperator {
	return WhereOperator{op: "=", value: value}
}

// NotEquals creates a not equal operator (!=)
func NotEquals(value interface{}) WhereOperator {
	return WhereOperator{op: "!=", value: value}
}

// Gt creates a greater than operator (>)
func Gt(value interface{}) WhereOperator {
	return WhereOperator{op: ">", value: value}
}

// Gte creates a greater than or equal operator (>=)
func Gte(value interface{}) WhereOperator {
	return WhereOperator{op: ">=", value: value}
}

// Lt creates a less than operator (<)
func Lt(value interface{}) WhereOperator {
	return WhereOperator{op: "<", value: value}
}

// Lte creates a less than or equal operator (<=)
func Lte(value interface{}) WhereOperator {
	return WhereOperator{op: "<=", value: value}
}

// Like creates a LIKE operator (case-sensitive pattern matching)
func Like(value string) WhereOperator {
	return WhereOperator{op: "LIKE", value: value}
}

// ILike creates an ILIKE operator (case-insensitive pattern matching)
func ILike(value string) WhereOperator {
	return WhereOperator{op: "ILIKE", value: value}
}

// In creates an IN operator for matching any value in a list
func In(values ...interface{}) WhereOperator {
	return WhereOperator{op: "IN", value: values}
}

// NotIn creates a NOT IN operator
func NotIn(values ...interface{}) WhereOperator {
	return WhereOperator{op: "NOT IN", value: values}
}

// IsNull creates an IS NULL operator
func IsNull() WhereOperator {
	return WhereOperator{op: "IS NULL", value: nil}
}

// IsNotNull creates an IS NOT NULL operator
func IsNotNull() WhereOperator {
	return WhereOperator{op: "IS NOT NULL", value: nil}
}

// Contains creates a LIKE operator with %value% pattern (case-sensitive)
func Contains(value string) WhereOperator {
	return WhereOperator{op: "LIKE", value: "%" + value + "%"}
}

// StartsWith creates a LIKE operator with value% pattern (case-sensitive)
func StartsWith(value string) WhereOperator {
	return WhereOperator{op: "LIKE", value: value + "%"}
}

// EndsWith creates a LIKE operator with %value pattern (case-sensitive)
func EndsWith(value string) WhereOperator {
	return WhereOperator{op: "LIKE", value: "%" + value}
}

// ContainsInsensitive creates an ILIKE operator with %value% pattern (case-insensitive)
func ContainsInsensitive(value string) WhereOperator {
	return WhereOperator{op: "ILIKE", value: "%" + value + "%"}
}

// StartsWithInsensitive creates an ILIKE operator with value% pattern (case-insensitive)
func StartsWithInsensitive(value string) WhereOperator {
	return WhereOperator{op: "ILIKE", value: value + "%"}
}

// EndsWithInsensitive creates an ILIKE operator with %value pattern (case-insensitive)
func EndsWithInsensitive(value string) WhereOperator {
	return WhereOperator{op: "ILIKE", value: "%" + value}
}

// Has checks if an array/JSON field contains a value
func Has(value interface{}) WhereOperator {
	return WhereOperator{op: "HAS", value: value}
}

// HasEvery checks if an array/JSON field contains all values
func HasEvery(values ...interface{}) WhereOperator {
	return WhereOperator{op: "HAS_EVERY", value: values}
}

// HasSome checks if an array/JSON field contains any value
func HasSome(values ...interface{}) WhereOperator {
	return WhereOperator{op: "HAS_SOME", value: values}
}

// IsEmpty checks if an array/JSON field is empty
func IsEmpty() WhereOperator {
	return WhereOperator{op: "IS_EMPTY", value: nil}
}

// GetOp returns the operator string (exported for internal use)
func (wo WhereOperator) GetOp() string {
	return wo.op
}

// GetValue returns the operator value (exported for internal use)
func (wo WhereOperator) GetValue() interface{} {
	return wo.value
}
