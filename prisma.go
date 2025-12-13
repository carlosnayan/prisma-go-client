// Package prisma provides a Prisma-like ORM for Go with PostgreSQL.
//
// Prisma for Go includes:
//   - Query builder with Prisma-style API
//   - Type-safe database operations
//   - Schema migrations management
//   - Code generation from schema
//   - Support for common Prisma CLI commands
//
// Example usage:
//
//	import "github.com/carlosnayan/prisma-go-client/builder"
//
//	// Create a query builder for a table
//	qb := builder.NewTableQueryBuilder(db, "users", columns)
//
//	// Find many records with conditions
//	users, err := qb.FindMany(ctx, builder.QueryOptions{
//	    Where: builder.Where{
//	        "active": true,
//	        "role": builder.In("admin", "user"),
//	    },
//	    OrderBy: []builder.OrderBy{{Field: "created_at", Order: "DESC"}},
//	    Take: builder.Ptr(10),
//	})
//
// CLI Commands:
//
//	prisma generate              # Generate Go types and query builders
//	prisma migrate dev           # Create and apply migrations in development
//	prisma migrate reset         # Reset database and reapply migrations
//	prisma db seed              # Seed database with initial data
//
// For more examples and documentation, visit:
// https://github.com/carlosnayan/prisma-go-client
package prisma

const Version = "0.2.2"
