package dist

import "embed"

//go:embed *.html all:layui
var WebDist embed.FS
