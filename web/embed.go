package web

import "embed"

//go:embed template/*
var TemplateFS embed.FS

//go:embed static/*
var StaticFS embed.FS
