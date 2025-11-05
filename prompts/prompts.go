package prompts

import (
	_ "embed"
)

//go:embed code-quality-analysis.txt
var CodeQualityAnalysis string

//go:embed hello.txt
var Hello string
