// Package migrations embeds the goose SQL migrations so services and tests can
// apply them without depending on files on disk.
package migrations

import "embed"

//go:embed migrate/*.sql
var FS embed.FS

// Dir is the directory (within FS) that holds the migration files.
const Dir = "migrate"
