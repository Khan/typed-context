package lintutil

// This file defines utilities relating to function-declarations.

import (
	"go/ast"
	"go/types"
	"strings"
)

// FilterFuncs returns the toplevel functions in the given files that match
// the predicate.
func FilterFuncs(files []*ast.File, predicate func(*ast.FuncDecl) bool) []*ast.FuncDecl {
	retval := []*ast.FuncDecl{}
	for _, file := range files {
		for _, decl := range file.Decls {
			funcDecl, ok := decl.(*ast.FuncDecl)
			if ok && predicate(funcDecl) {
				retval = append(retval, funcDecl)
			}
		}
	}
	return retval
}

// ReceiversByType returns all the receivers in the file, in a map by type.
//
// TODO(benkraft): May be more efficient to export this as an analyzer-result.
func ReceiversByType(files []*ast.File, typesInfo *types.Info) map[types.Type][]*ast.FuncDecl {
	allReceivers := FilterFuncs(files,
		func(decl *ast.FuncDecl) bool { return decl.Recv != nil })

	retval := map[types.Type][]*ast.FuncDecl{}
	for _, recv := range allReceivers {
		typ := UnwrapMaybePointer(typesInfo.TypeOf(recv.Recv.List[0].Type))
		retval[typ] = append(retval[typ], recv)
	}
	return retval
}

// CallsSuper returns whether the given function body (which must be a
// receiver) calls its "super" -- that is,
// <receiver-var>.<superclass-name>.<receiver-name>().
//
// TODO(benkraft): At present, we don't validate what the superclass name is;
// in the general case it's not clear what the correct rule would be as it's
// not necessarily the name of any field of the receiver-type (it may be a
// field of an intermediate embedded type).  We just validate that you call
// some method of a field of your receiver, which has the same name; in
// practice you probably only have one option.
func CallsSuper(funcDecl *ast.FuncDecl, typesInfo *types.Info) bool {
	foundSuper := false
	receiver := typesInfo.Defs[funcDecl.Recv.List[0].Names[0]]
	ast.Inspect(funcDecl.Body, func(node ast.Node) bool {
		expr, ok := node.(*ast.SelectorExpr)
		if !ok || expr.Sel.Name != funcDecl.Name.Name {
			return true // recurse
		}
		subExpr, ok := expr.X.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := subExpr.X.(*ast.Ident)
		if !ok || typesInfo.Uses[ident] != receiver {
			return true
		}
		foundSuper = true
		return false // no need to recurse here
	})
	return foundSuper
}

// Says whether the given function is a graphql resolver.  A
// surprising number of linters want to special case graphql
// resolvers, which have a format dictated by gqlgen and thus may not
// follow our linting rules.  This helps with that.
//
// The conditions we use are:
// 1) having receiver whose name ends with "Resolver"
// 2) is exported
// 3a) either has a `context.Context` as the first argument (for resolvers)
// 3b) or returns an object whose name ends with Resolver (for federation)
func IsResolverFunc(funcDecl *ast.FuncDecl, typesInfo *types.Info) bool {
	if funcDecl.Recv == nil {
		return false
	}
	if !funcDecl.Name.IsExported() {
		return false
	}
	t := funcDecl.Recv.List[0].Type
	// Unwrap (r *someResolver) into (r someResolver)
	if sid, ok := t.(*ast.StarExpr); ok {
		t = sid.X
	}
	tid, ok := t.(*ast.Ident)
	if !ok {
		return false
	}
	if !strings.HasSuffix(tid.Name, "Resolver") {
		return false
	}

	// ctx context.Context should be the first argument...
	if len(funcDecl.Type.Params.List) > 0 {
		firstArg := funcDecl.Type.Params.List[0]
		if TypeIs(typesInfo.TypeOf(firstArg.Type), "context", "Context") {
			return true
		}
	}
	// ...or the return type should have a name ending in resolver.
	if funcDecl.Type.Results != nil && len(funcDecl.Type.Results.List) > 0 {
		firstRet := funcDecl.Type.Results.List[0].Type
		if sid, ok := firstRet.(*ast.StarExpr); ok {
			firstRet = sid.X
		}
		ident, ok := firstRet.(*ast.Ident) // return type is Foo
		if ok && strings.HasSuffix(ident.Name, "Resolver") {
			return true
		}
		sel, ok := firstRet.(*ast.SelectorExpr) // return type is module.Foo
		if ok && strings.HasSuffix(sel.Sel.Name, "Resolver") {
			return true
		}
	}
	return false
}
