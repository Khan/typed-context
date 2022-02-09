package main

import (
	contextLinter "github.com/aberkan/typed_context/linter"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(contextLinter.TypedContextInterfaceAnalyzer)
}
