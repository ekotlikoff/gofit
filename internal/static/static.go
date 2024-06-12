package static

import (
	"embed"
	_ "embed"
)

var (
	//go:embed movements.json
	MovementsFS []byte
	//go:embed webpage
	WebpageStaticFS embed.FS
)
