package filestore

import (
	"embed"
	"io/fs"
)

//go:embed migrations/*.sql
var migrationsEmbed embed.FS

var MigrationsFS fs.FS

func init() {
	sub, err := fs.Sub(migrationsEmbed, "migrations")
	if err != nil {
		panic(err)
	}
	MigrationsFS = sub
}
