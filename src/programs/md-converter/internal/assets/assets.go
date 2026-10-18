package assets

import _ "embed"

// Mermaid 11.12.1; licenses are in THIRD_PARTY_LICENSES.md.
//go:embed mermaid.min.js
var Mermaid string

//go:embed default.css
var CSS string
