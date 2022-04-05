package main

import (
	contextLinter "github.com/khan/typed-context/linter"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(contextLinter.TypedContextInterfaceAnalyzer)
}
