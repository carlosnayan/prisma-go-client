package builder

// QueryOptions defines options for FindMany queries, similar to Prisma's findMany options
type QueryOptions struct {
	// Where conditions to filter records
	Where Where

	// OrderBy defines sorting order
	OrderBy []OrderBy

	// Take restricts the number of records returned
	Take *int

	// Skip skips a number of records
	Skip *int
}

// OrderBy defines sorting for a single field
type OrderBy struct {
	// Field name to sort by
	Field string

	// Order direction: "ASC" or "DESC"
	Order string
}

// Ptr is a helper function to create a pointer to an int
func Ptr(i int) *int {
	return &i
}

// BatchPayload represents the result of batch operations (CreateMany, UpdateMany, DeleteMany)
type BatchPayload struct {
	// Count is the number of records affected
	Count int
}
