# go-graft Rule Feature

This document is the source of truth for rule registration and callback behavior.

## Responsibilities

The `rule` feature owns:

- rule registry and snapshotting
- generic rule registration
- callback context creation semantics
- helper rules for function-call and method-call swapping

## Registration behavior

- Rule IDs are assigned in registration order and are used for default rule names.
- Registered rules are indexed by concrete `ast.Node` type.
- Rule panics are recovered and converted to immediate `Errored` mutants.
- Returning `changed=true` with a `nil` node is treated as an error.

## Callback input preparation

- Default mode: shallow-copy the matched node before invoking the callback.
- Deep-copy mode: deep-copy the subtree rooted at the matched node.
- Original-node mapping is recorded for callback-visible clones that need lookup support.

## Swap helper notes

- Function-call swap resolves the original callee through type information when available.
- Method-call swap validates receiver type identity and signature identity before mutating the selector.
- Helper parsing supports generic function runtime names and package paths whose last element contains a dot.
