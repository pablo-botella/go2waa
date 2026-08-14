package go2waa

import "errors"

// ErrNoWaaTargetConfigured means the destination resolved to nothing at
// all: no target was ever configured.
var ErrNoWaaTargetConfigured = errors.New("no WAA target configured")

// ErrShouldBeDispatchedElsewhere means this request is not WAA's: the
// package is mapped to "!" or no default target exists. Elsewhere is
// anything that is not WAA — typically the package's replacement in
// another technology, but whatever the caller decides. Call answers it
// with dispatched == false.
var ErrShouldBeDispatchedElsewhere = errors.New("should be dispatched elsewhere")

// ErrTargetNameAlreadyExists is AddTarget refusing a duplicate name.
var ErrTargetNameAlreadyExists = errors.New("target name already exists")

// ErrInvalidTargetName is AddTarget refusing the reserved name "!".
var ErrInvalidTargetName = errors.New("invalid target name")

// ErrTargetIndexOutOfRange means a package mapping points past the
// target list — a router wired by hand into an inconsistent state.
var ErrTargetIndexOutOfRange = errors.New("target index out of range")

// ErrTargetNameNotFound means no target carries the requested name.
var ErrTargetNameNotFound = errors.New("target name not found")
