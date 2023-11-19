package yubikey

import "embed"

// content holds our static web server content.
//
//go:embed all:templates
//go:embed all:static
var content embed.FS
