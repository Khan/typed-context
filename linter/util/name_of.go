// Package lintutil contains utilities for writing linters.
//
// For general documentation on linting, see dev/linters/README.md.
//
// TODO(benkraft): We may want to see if there's interest to contribute some of
// these back to astutil or typeutil.
package lintutil

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/ast/astutil"
)

// ObjectFor takes an AST node, and returns the corresponding types.Object, if
// there is one.
func ObjectFor(node ast.Node, typesInfo *types.Info) types.Object {
	// Sadly, there are a bunch of different ways that types.Info might store
	// this mapping, depending on the type of node and the context in which
	// it's used, so we have to do a few checks.  This is mostly cribbed from
	// https://github.com/golang/tools/blob/master/go/types/typeutil/callee.go#L16
	// which does a very similar thing, but only for functions.
	exprNode, ok := node.(ast.Expr)
	if !ok {
		return nil
	}
	exprNode = astutil.Unparen(exprNode)

	switch node := exprNode.(type) {
	case *ast.Ident:
		return typesInfo.ObjectOf(node)
	case *ast.SelectorExpr:
		if sel, ok := typesInfo.Selections[node]; ok {
			return sel.Obj()
		}
		return typesInfo.Uses[node.Sel]
	// TODO(benkraft): This is incomplete; we should check typesInfo.Types and
	// perhaps typesInfo.Implicits.  Nothing we do needs those yet.
	default:
		return nil
	}
}

// NameOf takes a node and returns the name of the symbol to which it refers,
// if any, in the form "package/path.UnqualifiedName". For built-in calls (such
// as `println()`) it uses a package name of "builtin" (so `builtin.println`).
//
// This will return a name for functions (including builtin), types,
// package-vars, consts, and not necessarily other nodes like struct fields.
// If it can't determine the name, it returns "".
//
// TODO(benkraft): Write tests for the const case, if we ever make use of that
// behavior.
//
// Note that methods have names like "(package/path.Interface).Method" or
// "(*package/path.Struct).Method".
func NameOf(obj types.Object) string {
	qualifiedName := func(obj types.Object) string {
		pkg := obj.Pkg()
		if pkg == nil {
			return obj.Name()
		}
		return pkg.Path() + "." + obj.Name()
	}

	switch obj := obj.(type) {
	case nil:
		return "nil"
	case *types.TypeName, *types.Const:
		return qualifiedName(obj)
	case *types.Var:
		if obj.IsField() {
			// TODO(benkraft): Handle struct fields.
			return ""
		}
		return qualifiedName(obj)
	case *types.Func:
		return obj.FullName()
	case *types.Builtin:
		return "builtin." + obj.Name()
	default:
		return ""
	}
}

// TypeIs takes a type object, and returns true if it is the given named type.
//
// Returns false if pkgPath.name is not a type, or if it is not this type.
// Note that this includes cases where this type wraps pkgPath.name, or where
// they share an underlying type: this will only return true if the types are
// the same.  Predeclared types will match the empty path.
//
// TODO(benkraft): Should we just check `typ.String() == "<pkgPath>.<name>"
// which seems to be the same?
func TypeIs(typ types.Type, pkgPath string, name string) bool {
	named, ok := typ.(*types.Named)
	if !ok {
		return false
	}

	if named.Obj().Name() != name {
		return false
	}

	if named.Obj().Pkg() == nil {
		return pkgPath == ""
	}
	return named.Obj().Pkg().Path() == pkgPath
}
