// Package migrations embeds the SQL migration files so the migrate command (and tests) run
// exactly the files in this folder, with no path lookup at run time.
package migrations

import "embed"

// FS holds every *.sql file in this directory.
//
//go:embed *.sql
var FS embed.FS
