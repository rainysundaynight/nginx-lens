package hub

import "embed"

// ---------- Встроенные статические файлы ----------
// Dashboard UI для hub (go:embed).

//go:embed static/*
var staticFS embed.FS

//go:embed templates/*
var templatesFS embed.FS
