package limits

// Memory safety limits to prevent unbounded growth and OOM

const (
	// MaxScanRows is the maximum number of rows that can be scanned into memory
	// This prevents OOM when querying very large datasets
	MaxScanRows = 100000

	// MaxQueryConditions is the maximum number of WHERE conditions in a single query
	// This prevents unbounded growth of whereConditions slice
	MaxQueryConditions = 1000

	// MaxJoins is the maximum number of JOINs in a single query
	// This prevents unbounded growth of joins slice
	MaxJoins = 50

	// MaxOrderByFields is the maximum number of ORDER BY fields
	MaxOrderByFields = 20

	// MaxGroupByFields is the maximum number of GROUP BY fields
	MaxGroupByFields = 20

	// MaxSelectFields is the maximum number of SELECT fields
	MaxSelectFields = 100
)

