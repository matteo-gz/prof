package web

import "embed"

//go:embed template/*
var TemplateFS embed.FS

//go:embed all:static
var StaticFS embed.FS
