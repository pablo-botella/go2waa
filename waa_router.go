package go2waa

import (
	"fmt"
	"strings"
)

// WaaRouter is the multibinding destination: it holds several targets
// and picks one per call by the package variable of the conversation
// (WAA_PACKAGE by default). Unmapped packages go to the default target
// (the first one added, unless SetDefaultTarget says otherwise), and a
// package mapped to "!" is served elsewhere — anything that is not WAA:
// the call answers ErrShouldBeDispatchedElsewhere and the caller serves
// it its own way. The typical use is the migration lever: an Xbase++/WAA
// program moves to Go package by package, each replacement mapped to
// "!", WAA serving the rest, no seams visible.
type WaaRouter struct {
	targetList     []*WaaTarget   // one per WAA server, each with its own host, port, timeouts
	targetDefault  int            // index of the default target; zero value = the first one; -1 (SetDefaultTarget "!") = no default, dispatch elsewhere
	packageMap     map[string]int // map package name to targetList index
	packageVarName string         // optional name of the variable to use for the switch, typically "WAA_PACKAGE" that is the default
}

var _ WaaTargetParam = (*WaaRouter)(nil)

func (r *WaaRouter) _init() {
	if r.packageMap == nil {
		r.packageMap = make(map[string]int)
	}
	if r.targetList == nil {
		r.targetList = make([]*WaaTarget, 0)
	}
	if r.packageVarName == "" {
		r.packageVarName = "WAA_PACKAGE"
	}
}
// SetPackageVarName changes the variable that picks the binding.
// Usually you migrate package by package (WAA_PACKAGE, the default),
// but inside one package another variable may play the pivot; "" resets
// to the default.
func (r *WaaRouter) SetPackageVarName(name string) {
	r._init()
	r.packageVarName = name
}

// GetTargetIndexByName finds a target by its name, case-insensitively;
// -1 when no target carries it.
func (r *WaaRouter) GetTargetIndexByName(name string) int {
	r._init()
	n := strings.ToLower(name)
	for idx, target := range r.targetList {
		if strings.ToLower(target.Name) == n {
			return idx
		}
	}
	return -1
}
// AddTarget registers one WAA server under a name ("" auto-names it
// "host:port"; "!" is reserved and refused) and returns the created
// target. The first target added is the default one. Timeouts are
// optional: conn, read, write, in seconds.
func (r *WaaRouter) AddTarget(name string, host string, port int, timeouts ...int) (error, *WaaTarget) {

	r._init()
	if name == "!" { // reserved name for "dispatch elsewhere"
		r.targetDefault = -1
		return ErrInvalidTargetName, nil
	}

	t := &WaaTarget{}
	t.Init(host, port, timeouts...)
	if name == "" {
		name = fmt.Sprintf("%s:%d", host, port)
	}
	i := r.GetTargetIndexByName(name)
	if i >= 0 {
		return ErrTargetNameAlreadyExists, nil
	}
	t.Name = name
	i = len(r.targetList)
	r.targetList = append(r.targetList, t)
	return nil, t
}

// SetDefaultTarget picks the target that serves the unmapped packages.
// "!" means no default at all: anything unmapped should be dispatched
// elsewhere.
func (r *WaaRouter) SetDefaultTarget(name string) error {
	r._init()
	if name == "!" {
		r.targetDefault = -1
		return nil
	}
	n := strings.ToLower(name)
	for idx, target := range r.targetList {
		if strings.ToLower(target.Name) == n {
			r.targetDefault = idx
			return nil
		}
	}
	return ErrTargetNameNotFound
}
// MapPackageToTarget binds one package value to a target by name.
// Target "!" marks the package as already migrated: it will be
// dispatched elsewhere (the migration lever, one package at a time).
func (r *WaaRouter) MapPackageToTarget(pkgName string, targetName string) error {
	r._init()
	if targetName == "!" {
		r.packageMap[pkgName] = -1 // should be dispatched elsewhere
		return nil
	}

	i := r.GetTargetIndexByName(targetName)
	if i >= 0 {
		r.packageMap[pkgName] = i
		return nil
	}
	return ErrTargetNameNotFound
}

// GetWaaTargetParam resolves the destination for this conversation: the
// package variable is asked to the ctx, mapped packages go to their
// target, unmapped ones to the default, and no default (or a "!"
// mapping) answers ErrShouldBeDispatchedElsewhere — not WAA's request.
func (w *WaaRouter) GetWaaTargetParam(ctx WaaCtx) (error, *WaaTarget) {
	w._init()
	i := -1
	if p, found := ctx.OnGetVar(w.packageVarName); found && len(p) > 0 {
		i, found = w.packageMap[p[0]]
		if !found {
			i = w.targetDefault
		}
	}
	if i < 0 {
		return ErrShouldBeDispatchedElsewhere, nil
	}
	if i >= len(w.targetList) {
		return ErrTargetIndexOutOfRange, nil
	}
	return w.targetList[i].GetWaaTargetParam(ctx)
}
