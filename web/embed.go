package web

import "embed"

// Files contains the embedded static assets for GoodAI Monitor web dashboard.
//
//go:embed index.html style.css app.js logo.svg
var Files embed.FS
