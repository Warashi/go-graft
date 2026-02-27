package main

import (
	"context"
	"flag"
	"fmt"
	"go/ast"
	"go/token"
	"os"
	"time"

	"github.com/Warashi/go-graft"
)

func main() {
	var workers int
	var timeout time.Duration
	var baseTempDir string
	var keepTemp bool
	var builtinRule bool

	flag.IntVar(&workers, "workers", 1, "number of concurrent go test worker processes")
	flag.DurationVar(&timeout, "timeout", 30*time.Second, "timeout per mutant")
	flag.StringVar(&baseTempDir, "base-temp-dir", "", "base directory for mutant temp dirs")
	flag.BoolVar(&keepTemp, "keep-temp", false, "keep mutant temp dirs for debugging")
	flag.BoolVar(&builtinRule, "builtin-add-to-sub", true, "enable builtin + to - mutation rule")
	flag.Parse()

	engine := graft.New(graft.Config{
		Workers:       workers,
		MutantTimeout: timeout,
		BaseTempDir:   baseTempDir,
		KeepTemp:      keepTemp,
	})
	if builtinRule {
		graft.Register[*ast.BinaryExpr](engine, func(_ *graft.Context, n *ast.BinaryExpr) (*ast.BinaryExpr, bool) {
			if n.Op != token.ADD {
				return nil, false
			}
			n.Op = token.SUB
			return n, true
		}, graft.WithName("builtin-add-to-sub"))
	}

	patterns := flag.Args()
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	report, err := engine.Run(context.Background(), patterns...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "go-graft: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("total=%d killed=%d survived=%d unsupported=%d errored=%d score=%.2f\n",
		report.Total, report.Killed, report.Survived, report.Unsupported, report.Errored, report.MutationScore())
}
