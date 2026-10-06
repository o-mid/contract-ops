// Package migrations embeds the goose SQL files into the migrate binary.
package migrations

import "embed"

// FS holds the SQL migrations. The binary carries them so a container
// does not need the source tree mounted in.
//
//go:embed *.sql
var FS embed.FS
