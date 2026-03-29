# go-graft Mutation Feature

This document is the source of truth for mutation-point collection and mutant artifact building.

## Mutation-point collection

`mutation.Collect`:

- traverses non-test Go files only
- matches nodes by concrete `ast.Node` type
- records package identity, file path, AST path, source position, and enclosing function
- prefers compiled Go file paths when building overlay replacement targets

One generated mutant always corresponds to one mutation point and one node replacement.

## AST clone infrastructure

`astclone` provides:

- shallow-copy support for supported nodes
- deep-copy support with clone-to-original tracking
- path cloning plus single-node replacement for building mutated files

Generated copy/replace support is maintained by:

```bash
go generate ./internal/astclone
```

## Mutant artifact building

`mutation.Builder`:

- creates one temp directory per mutant
- writes the formatted mutated file under `overlay/`
- writes `overlay.json` mapping the original compiled file to the mutated file
- creates a mutant-local `tmp/` directory used as `TMPDIR` during execution
