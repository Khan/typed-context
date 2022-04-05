package linters

// This file defines the linter that typed context interfaces aren't
// unnecessarily large, and that any interface-component that's used explicitly
// is mentioned explicitly.
//
// The rules for this are somewhat complex.  In particular, for each ctx
// argument, we define some list of interfaces that it "explicitly" mentions
// (further defined in _explicitInterfaces, below).  Then the rules are that
// for each variable v of type I:
// - for each interface J recursively embedded in I, some use of v must use J
// - for each use of v that explicitly mentions J, I must explicitly mention J,
//   or must explicitly mention J's explicit mentions (or recursively)
//
//

import (
	"go/ast"
	"go/token"
	"go/types"
	"sort"
	"strings"

	"golang.org/x/tools/go/analysis"

	lintutil "github.com/aberkan/typed_context/linter/util"
)

var TypedContextInterfaceAnalyzer = &analysis.Analyzer{
	Name: "typedcontextinterface",
	Doc:  "enforces that typed context interfaces aren't unnecessarily large",
	Run:  _runInterface,
}

// isContextType returns true if the input is a context-type (either Go-style
// context.Context or a typed-context style interface embedding it).
func isContextType(typ types.Type) bool {
	if lintutil.TypeIs(typ, "context", "Context") {
		return true
	}
	iface, ok := typ.Underlying().(*types.Interface)
	if !ok {
		return false
	}
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		if isContextType(iface.EmbeddedType(i)) {
			return true
		}
	}
	return false
}

// _explicitInterfaces returns the Typed-Context interfaces explicitly
// included in the given type.  (This may include the type itself.)
//
// Specifically, these are the interfaces that you are treated as having
// requested if you use the given interface; and they are the interfaces that
// you are treated as having used (and thus that you need to have requested) if
// you call some function which wants the given interface.
//
// Defining that in a way that makes sense is somewhat subtle.  We use package
// boundaries:
// - we do not include, and recurse on on all unnamed or unexported interfaces
//   within the package
// - we include, but also recurse on, all named exported interfaces within the
//   package
// - we include, and do not recurse on, all named interfaces defined in other
//   packages
//
// In context, this means if you request some context from another package
//	type I interface { C }
// it's fine to use that to call some function `otherpkg.F(ctx otherpkg.I)`,
// but you can't use `C` yourself.  But if `I` were defined in your package, it
// would be fine to use `C` -- you are the one wrapping things up and maybe the
// whole reason to define `I` is so your callers can use it.  (But if `C`
// itself contains other contexts, you still can't use those.)
//
// For example, given:
//	type A interface { other.B; c; M() }
//	type c interface { other.D }
//	func(ctx interface { A; other.E })
// then calling _explicitInterfaces on the type of ctx will return `A`,
// `other.B`, `other.D`, and `other.E`, but not `c` (it's not exported),
// `interface { A; other.F }` (it's not named), nor `M()` (it's not itself an
// interface).
func _explicitInterfaces(typ types.Type, currentPackage *types.Package) []types.Type {
	iface, ok := typ.Underlying().(*types.Interface)
	if !ok {
		return nil
	}

	retval := make([]types.Type, 0, iface.NumEmbeddeds())
	named, ok := typ.(*types.Named)
	if ok && named.Obj().Pkg() != currentPackage {
		return []types.Type{typ}
	} else if ok && named.Obj().Exported() {
		retval = append(retval, typ)
	}

	for i := 0; i < iface.NumEmbeddeds(); i++ {
		retval = append(retval, _explicitInterfaces(iface.EmbeddedType(i), currentPackage)...)
	}
	return retval
}

// _leafInterfaces returns a list of all interfaces embedded by this
// interface, including the interface itself, stopping at interfaces with
// methods.
//
// For example, if you do
//	type A interface { B; C }
//	type B interface { M() }
//	type C interface { D; N() }
//	type D interface { O() }
// then:
//	_leafInterfaces(A) => B, C
//	_leafInterfaces(B) => B
//	_leafInterfaces(C) => C
//
// NOTE: Stopping at interfaces with methods is sort of a heuristic.
// It doesn't work very well in cases where caller or callee embed their own
// explicit method, rather than another context.  For example, if caller has
// `interface { A; B; M() }` and one callee wants A and the other callee wants
// `interface { B; M() }`, we'll see both as unused: the caller is seen as
// having a single context-interface `{ A; B; M() }`, which is not equal to
// either A or `{ B; M() }`.  One way to solve for this would be to have
// some base interface included in each context, but that would require adding
// new packages, and doesn't seem to have many benefits other than in this linter.
func _leafInterfaces(typ types.Type) []types.Type {
	iface, ok := typ.Underlying().(*types.Interface)
	if !ok {
		return nil
	}

	if iface.NumExplicitMethods() > 0 {
		return []types.Type{typ}
	}

	retval := make([]types.Type, 0, iface.NumEmbeddeds())
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		retval = append(retval, _leafInterfaces(iface.EmbeddedType(i))...)
	}
	return retval
}

// _embedsExplicitlyContaining returns the interface recursively embedded in
// this interface(s), if any, which explicitly contains a method with the given
// name.
//
// If the method is an explicit method of the interface, returns the input
// interface.  If the method is not a method of the input interface at all,
// returns nil.  If the method is an explicit method of several recursively
// embedded interfaces (rare), returns all of them.
//
// Note the returned value contains the types as used (e.g. named types), not
// the underlying interface types.  This is all used to calculate which
// contexts you must explicitly request to use a method.
func _embedsExplicitlyContaining(typ types.Type, methodName string) []types.Type {
	iface, ok := typ.Underlying().(*types.Interface)
	if !ok {
		return nil
	}

	embeds := map[types.Type]bool{}
	// If the method is an explicit method of the interface, return the
	// interface.
	for i := 0; i < iface.NumExplicitMethods(); i++ {
		if iface.ExplicitMethod(i).Name() == methodName {
			embeds[typ] = true
			break // early-out: interfaces can't have explicit dupe methods
		}
	}

	// Otherwise, check the embeds.
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		for _, embed := range _embedsExplicitlyContaining(iface.EmbeddedType(i), methodName) {
			embeds[embed] = true
		}
		// (no early-out: we can have the same method via two embeds, in 1.14+)
	}

	// uniquify, for the case of diamond-deps
	retval := make([]types.Type, 0, len(embeds))
	for embed := range embeds {
		retval = append(retval, embed)
	}
	return retval
}

// _embedNamed takes an interface type and returns the interface type, if any,
// recursively embedded in it with the given name.  The names are as with
// lintutil.TypeIs.
//
// This is sort of a hack to get a reference to the types.Type for
// context.Context; we don't have a convenient way to look it up a priori, but
// we do have a reference to kacontext.Base, so we can grab the former from the
// latter.
func _embedNamed(typ types.Type, pkgName, typeName string) types.Type {
	if lintutil.TypeIs(typ, pkgName, typeName) {
		return typ
	}

	iface, ok := typ.Underlying().(*types.Interface)
	if !ok {
		return nil
	}

	// Check the embeds
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		embed := _embedNamed(iface.EmbeddedType(i), pkgName, typeName)
		if embed != nil {
			return embed
		}
	}

	return nil
}

// getParamAt gets the parameter to which the i'th argument of funcType will
// be assigned.
//
// You might think this would be just funcType.Params().At(i), but for variadic
// functions it may instead be the final parameter.
//
// Returns nil if there is no such parameter, which can happen for the function
// make() due to a bug: https://github.com/golang/go/issues/37349.  After
// that's fixed, this should never return nil.
func getParamAt(funcType *types.Signature, i int) *types.Var {
	params := funcType.Params()
	nParams := params.Len()
	switch {
	case i < nParams:
		return params.At(i)
	case funcType.Variadic():
		return params.At(nParams - 1)
	default:
		// Should never happen, except see the bug in the function docstring.
		return nil
	}
}

// _shortTypeName returns typ.String(), or a less verbose form if possible.
//
// For example, if typ is a named type, typ.String() includes the full package
// path; ShortTypeName(typ) just includes the package name.  pkg may be set to
// the current package, in which case types from that package will be printed
// unqualified.
func _shortTypeName(typ types.Type, pkg *types.Package) string {
	name := typ.String()
	if typ, ok := typ.(*types.Named); ok {
		obj := typ.Obj()
		switch obj.Pkg() {
		case nil:
			// typ.String() will have to be good enough!
		case pkg:
			return obj.Name() // unqualified name
		default:
			return obj.Pkg().Name() + "." + obj.Name()
		}
	}
	return name
}

// _expandUnexportedNames takes a list of types, and for any type that is not
// visible to `pkg` -- because it is an unexported type in a different package
// -- it replaces that type with its list of embeds, recursing until the embeds
// are all visible.
//
// The idea is to return a list of interfaces that can actually be referenced
// in `pkg`.
//
// There's one complication, which is if the unexported type embeds a method
// directly (rather than another interface).  In that case, we return an
// unnamed interface with just that method.
//
// For example, if we have in some package mypkg
//	type i interface { j; k }
//	type j interface { L }
//	type k interface { M(); N }
// then we get
//	_expandUnexportedNames(i, otherpkg) => L, N, interface { M() }
//	_expandUnexportedNames(L, otherpkg) => L
//	_expandUnexportedNames(i, mypkg)    => i
func _expandUnexportedNames(typ types.Type, pkg *types.Package) []types.Type {
	iface, ok := typ.Underlying().(*types.Interface)
	if !ok {
		// probably shouldn't happen? But we may as well return the input.
		return []types.Type{typ}
	}

	named, ok := typ.(*types.Named)
	if ok && (named.Obj().Exported() || named.Obj().Pkg() == pkg) {
		// not not exported, or a named type in this package: safe to use.
		return []types.Type{typ}
	}

	// else, we have to expand the interface into its components.
	retval := make([]types.Type, 0, iface.NumEmbeddeds())
	for i := 0; i < iface.NumEmbeddeds(); i++ {
		// add all of this interfaces embeds (and recursively).
		retval = append(retval, _expandUnexportedNames(iface.EmbeddedType(i), pkg)...)
	}
	if iface.NumExplicitMethods() > 0 {
		// construct an unnamed interface with just the explicit methods.
		methods := make([]*types.Func, iface.NumExplicitMethods())
		for i := 0; i < iface.NumExplicitMethods(); i++ {
			methods[i] = iface.ExplicitMethod(i)
		}
		methodIface := types.NewInterfaceType(methods, nil /* embeds */)
		methodIface = methodIface.Complete()
		retval = append(retval, methodIface)
	}

	return retval
}

// _formatTypeList pretty-prints a list of types, using _shortTypeName.
func _formatTypeList(types []types.Type, pkg *types.Package) string {
	names := make([]string, 0, len(types))
	for _, typ := range types {
		for _, innerTyp := range _expandUnexportedNames(typ, pkg) {
			names = append(names, _shortTypeName(innerTyp, pkg))
		}
	}
	sort.Strings(names)
	// uniquify -- duplicates can happen if you needed a context both via a
	// method and a function-argument, or suchlike, and didn't request it.
	uniqueNames := make([]string, 0, len(types))
	for i, name := range names {
		if i == 0 || names[i-1] != name {
			uniqueNames = append(uniqueNames, name)
		}
	}
	return strings.Join(uniqueNames, ", ")
}

// _hasExplicitMethod returns true if iface has an explicit method with the
// given name (i.e. it's defined on that interface, not some embedded
// interface).
func _hasExplicitMethod(iface *types.Interface, name string) bool {
	for i := 0; i < iface.NumExplicitMethods(); i++ {
		if iface.ExplicitMethod(i).Name() == name {
			return true
		}
	}
	return false
}

// _interfaceTracker is the object we use to manage our process of marking
// which interfaces requested by an object are used.
type _interfaceTracker struct {
	// Map goes: object we want to check -> interfaces it uses -> whether we've
	// found a use.  The types are those returned by _explicitInterfaces.
	trackedIdents map[types.Object]*_objInfo

	typesInfo *types.Info
	pkg       *types.Package
}

// track adds the given identifier to have its interface usage tracked.
//
// If the identifier is named _, or is not a context type, it is ignored.
func (tracker *_interfaceTracker) track(ident *ast.Ident) {
	obj := tracker.typesInfo.Defs[ident]
	// obj is only nil in edge cases we don't care about (like struct fields)
	if obj == nil || obj.Name() == "_" || !isContextType(obj.Type()) {
		return
	}

	ifaces := _leafInterfaces(obj.Type())
	if len(ifaces) == 0 {
		return // this isn't a ctx.
	}

	// If you _just_ requested context.Context, and don't use it, that's
	// probably to match an interface or for future expansion, and anyway
	// is a job for an unused-argument linter, not us.  We just skip
	// checking this case.
	if len(ifaces) == 1 && lintutil.TypeIs(ifaces[0], "context", "Context") {
		return
	}

	// Otherwise, get ready to track this interface.
	tracker.trackedIdents[obj] = &_objInfo{
		obj:           obj,
		interfaceUses: map[types.Type]bool{},
		methodUses:    map[string]bool{},
	}
}

// _markArgsUsed marks used any context-interfaces which are required as
// parameters to the given call.
//
// For example, if you call database.Read(ctx), this will mark the
// database.Context interface of ctx as used.
func (tracker *_interfaceTracker) _markArgsUsed(call *ast.CallExpr) {
	funcType, ok := tracker.typesInfo.TypeOf(call.Fun).Underlying().(*types.Signature)
	if !ok {
		panic("Bad Signature?")
	}
	for i := 0; i < len(call.Args); i++ {
		argIdent, ok := call.Args[i].(*ast.Ident)
		if !ok {
			continue
		}
		param := getParamAt(funcType, i)
		if param == nil {
			continue
		}
		info := tracker.trackedIdents[tracker.typesInfo.ObjectOf(argIdent)]
		if info != nil {
			info.interfaceUses[param.Type()] = true
		}
	}
}

// _markCastUsed marks used any context-interfaces used via a cast.
//
// This is sorta a hack: you're doing a cast and all bets are off!  But in
// practice it makes sense that we mark the overlap between the type you are
// and the type you're casting to as used.  For example, if you cast from
// interface{ A; B } to interface{ B; C } we'll count that as a use of B.
func (tracker *_interfaceTracker) _markCastUsed(cast *ast.TypeAssertExpr) {
	ident, ok := cast.X.(*ast.Ident)
	if !ok {
		return
	}

	info := tracker.trackedIdents[tracker.typesInfo.ObjectOf(ident)]
	if info != nil {
		info.interfaceUses[tracker.typesInfo.TypeOf(cast.Type)] = true
	}
}

// _markReceiverUsed marks used any context-interfaces which are required to
// make this receiver-method call.
//
// For example, if you call ctx.Datastore(), this will mark the
// datastore.KAContext interface of ctx as used.
func (tracker *_interfaceTracker) _markReceiverUsed(call *ast.CallExpr) {
	// We want the case where the function is <ident>.<method>.
	selector, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return
	}
	recv, ok := selector.X.(*ast.Ident)
	if !ok {
		return
	}
	info := tracker.trackedIdents[tracker.typesInfo.ObjectOf(recv)]
	if info != nil {
		info.methodUses[selector.Sel.Name] = true
	}
}

// _markCachedFunctionUsed marks any context-interfaces that might be needed
// for our caching library (pkg/lib/cache), as a special-case.  This is a case
// it's common in our codebase, and hard to handle other ways, so we just put
// in a special hack.
func (tracker *_interfaceTracker) _markCachedFunctionUsed(call *ast.CallExpr) {
	funcName := lintutil.NameOf(lintutil.ObjectFor(call.Fun, tracker.typesInfo))
	if funcName != "github.com/Khan/webapp/pkg/lib/cache.Cache" ||
		len(call.Args) == 0 { // len == 0 never happens (cache arg is required)
		return
	}

	cachedFunctionSig, ok := tracker.typesInfo.TypeOf(call.Args[0]).(*types.Signature)
	if !ok || cachedFunctionSig.Params().Len() == 0 {
		// should also never happen (if init-time validation passes): first arg
		// of cache is always a function, and it must have a context arg
		return
	}

	ctxArg := cachedFunctionSig.Params().At(0)
	info := tracker.trackedIdents[ctxArg]
	if info != nil {
		info.isCached = true
	}
}

// _markKeyParamsFunctionUsed marks any context-interfaces that might be needed
// for a key-params function in our caching library (pkg/lib/cache), as a
// special-case.  This is a case it's common in our codebase, and hard to
// handle other ways, so we just put in a special hack.
func (tracker *_interfaceTracker) _markKeyParamsFunctionUsed(call *ast.CallExpr) {
	funcName := lintutil.NameOf(lintutil.ObjectFor(call.Fun, tracker.typesInfo))
	if funcName != "github.com/Khan/webapp/pkg/lib/cache.KeyParamsFxn" ||
		len(call.Args) == 0 { // len == 0 never happens (cache arg is required)
		return
	}

	cachedFunctionSig, ok := tracker.typesInfo.TypeOf(call.Args[0]).(*types.Signature)
	if !ok || cachedFunctionSig.Params().Len() == 0 {
		// should also never happen (if init-time validation passes): first arg
		// of cache is always a function, and it must have a context arg
		return
	}

	// If it's used as a key-params fxn, its argument types must match exactly
	// those of the cached function, so we just ignore it.
	ctxArg := cachedFunctionSig.Params().At(0)
	delete(tracker.trackedIdents, ctxArg)
}

func (tracker *_interfaceTracker) _markSingleStructValueUsed(typ types.Type, val ast.Expr) {
	ident, ok := val.(*ast.Ident)
	if !ok {
		return
	}

	info := tracker.trackedIdents[tracker.typesInfo.ObjectOf(ident)]
	if info != nil {
		info.interfaceUses[typ] = true
	}
}

// _markCompositeLitValuesUsed marks used any context-interfaces which are
// required to use the context in this struct-, map-, slice-, or
// array-literal.
//
// At this time, we only look at struct-literals, because it's not common to
// have a map, slice, or array containing a context.
func (tracker *_interfaceTracker) _markCompositeLitValuesUsed(compLit *ast.CompositeLit) {
	if len(compLit.Elts) == 0 {
		return
	}

	typ := tracker.typesInfo.TypeOf(compLit)
	if typ == nil { // should never happen
		return
	}

	underlying, ok := typ.Underlying().(*types.Struct)
	if !ok { // map, slice, or array
		return
	}

	// It's guaranteed that either all fields are keyed, or none of them are,
	// but we just check each, it's easier that way.
	for i, element := range compLit.Elts {
		switch element := element.(type) {
		case *ast.KeyValueExpr:
			// Keyed field; the type of the key is the type of the
			// struct-field.
			tracker._markSingleStructValueUsed(
				tracker.typesInfo.TypeOf(element.Key), element.Value)
		default:
			// Unkeyed field; we just look at the i'th field of the struct.
			tracker._markSingleStructValueUsed(
				underlying.Field(i).Type(), element)
		}
	}
}

// markUses traverses marks as used all interfaces required by the code in the
// given node and all its descendants.
func (tracker *_interfaceTracker) markUses(startNode ast.Node) {
	ast.Inspect(startNode, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.TypeAssertExpr:
			if node.Type != nil { // nil means a type-switch x.(type)
				tracker._markCastUsed(node)
			}
		case *ast.CallExpr:
			tracker._markArgsUsed(node)
			tracker._markReceiverUsed(node)
			tracker._markCachedFunctionUsed(node)
			tracker._markKeyParamsFunctionUsed(node)
		case *ast.CompositeLit: // struct, map, or array
			tracker._markCompositeLitValuesUsed(node)
			// There are a bunch of other ways to use a
			// value: for example you could assign it to a variable/field,
			// use it in a struct literal, etc., so more may be needed here.
		}
		return true // otherwise, recurse
	})
}

// trackIdents registers all identifiers (function parameters, variables, etc.)
// in the given node and all its descendants if we want to ensure they have no
// more ka-contexts than they need.
//
// If includeFuncType is set, we will recurse within the first funcType we see
// (typically the input node).  Otherwise, we don't (see comments inline).
func (tracker *_interfaceTracker) trackIdents(node ast.Node, includeFuncType bool) {
	ast.Inspect(node, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.Ident:
			tracker.track(node)
			return false // nothing to recurse
		case *ast.GenDecl:
			// Don't recurse within typedefs -- we'll lint at their
			// use-sites if relevant.
			return node.Tok != token.TYPE
		case *ast.FuncType:
			// We don't look at FuncTypes unless they're a child of a
			// FuncLit or a FuncDecl.  In those cases (immediately following)
			// we set the flag includeFuncType to signal that; otherwise we
			// just ignore the FuncType.  (We also set the flag back to false
			// before recursing; we only want to handle the immediate child
			// FuncType, not, say, a FuncType within the FuncDecl's body.)
			//
			// This is all because we want to analyze the named parameters in,
			// e.g.,
			//	helper := func(ctx ...) { ... }
			// (which is a FuncLit) but not in
			//	var myFunc = cache.Cache(_uncachedMyFunc).(func(ctx ...))
			// (where the FuncType is nested within a TypeAssertExpr
			// instead) as the latter don't really have uses as such.
			ret := includeFuncType
			includeFuncType = false
			return ret
		case *ast.FuncDecl:
			// In this case, we want to recurse into our child FuncType.  But
			// the normal recursion we do via `return true` won't do that,
			// since normally FuncType's are ignored (in the case right before
			// this one).  So we explicitly recurse on the FuncType, setting
			// the flag such that it won't be ignored.
			tracker.trackIdents(node.Type, true)
			return true
		case *ast.FuncLit:
			// Same as FuncDecl.
			tracker.trackIdents(node.Type, true)
			return true
		default:
			return true // recurse everywhere else
		}
	})
}

// identifyInterfaceMethods modifies trackedIdents so that its maps are shared
// between implementations of the same interface method.
//
// If you want to implement an interface, the types of your methods must match
// exactly; this means sometimes you have to ask for a type with more
// typed context interfaces than you really wanted.  For example, if you have
// several implementations T, U, and V of an interface I { M(ctx ...) }, you
// might require that the context be some context-type K because T needs that
// type, but U and V might need only a subset of it.  We don't want to complain
// about that.
//
// So, we update tracker.trackedIdents, such that the entries corresponding to
// the 'ctx' arguments of T, U, and V are all the same.  That way, when we mark
// a type as used by T, we'll cover U and V as well.  In fact, we'll allow it
// even if T, U, and V each use different subsets of K, which add up to the
// whole thing!  (See tests for examples.)
//
// NOTE: We might also wish to check for the case where the interface
// being implemented is in another package; we could look for the standard
//	var _ I = (*T)(nil) // ensure T implements I
// to avoid looking at all interfaces ever.
//
// NOTE: Another thing we should check with interfaces is that the
// interface explicitly requests all the contexts that its implementations do.
// If you use named types, that's already guaranteed -- an interface-method
// `M(MyContext)` is only matched by an implementation-method `M(MyContext)` --
// but if you did `M(interface { ... })` on the interface, then the
// implementation can use any other interface with the same method-set.  We
// should ideally to say they have to be structurally the same, or at least
// have the same explicit members, in the sense used elsewhere in this linter.
func (tracker *_interfaceTracker) identifyInterfaceMethods(files []*ast.File) {
	recvs := lintutil.ReceiversByType(files, tracker.typesInfo)

	// First, find all the named interfaces in the package.
	for _, def := range tracker.typesInfo.Defs {
		typeDef, ok := def.(*types.TypeName)
		if !ok {
			continue // not a type-definition
		}
		iface, ok := typeDef.Type().Underlying().(*types.Interface)
		if !ok {
			continue // not an interface
		}
		if iface.Empty() {
			// early-out; the rest would be a no-op anyway because the empty
			// interface has no methods.
			continue
		}

		// We have a (non-empty) interface; find its methods.
		//
		// The methods are identified by their "ID" as used by the go/types
		// package, which is the unqualified-name for an exported method, and
		// the package + unqualified name for unexported methods.  This matches
		// how go does interface method name-matching.
		mapsByMethod := map[string]*_objInfo{}
		for i := 0; i < iface.NumMethods(); i++ {
			// Id() returns package + local-name if the method is unexported,
			// or just the local-name if it's exported; this is the key on
			// which Go matches interface method-names.
			mapsByMethod[iface.Method(i).Id()] = nil
		}

		// Now, go through all the receivers for types which implement this
		// interface, and do the map-sharing.
		for recvTyp, recvDefs := range recvs {
			// We identify the methods as long as the pointer implements the
			// interface.  (This includes the case where the value implements
			// the interface.)
			if !types.Implements(types.NewPointer(recvTyp), iface) {
				continue
			}

			for _, recvDef := range recvDefs {
				recvObj := tracker.typesInfo.Defs[recvDef.Name]
				if recvObj == nil { // should never happen
					continue
				}
				id := recvObj.Id()
				mapForMethod, ok := mapsByMethod[id]
				if !ok { // not a method of this interface
					continue
				}

				paramsList := recvDef.Type.Params.List
				if len(paramsList) == 0 || len(paramsList[0].Names) == 0 {
					// we're only interested in functions with at least one
					// named parameter
					continue
				}

				// Get the first parameter, that's where the ctx should be.
				paramObj := tracker.typesInfo.Defs[paramsList[0].Names[0]]
				if tracker.trackedIdents[paramObj] == nil {
					// not a parameter we are interested in
					continue
				}

				// We found one!  Set up the sharing.  If this was the first
				// implementation we've found, save this map so we can use it
				// for later methods.  Otherwise, re-use that saved map.
				if mapForMethod == nil {
					mapsByMethod[id] = tracker.trackedIdents[paramObj]
				} else {
					tracker.trackedIdents[paramObj] = mapForMethod
				}
			}
		}
	}
}

// _objInfo represents what we know about how a particular variable is used.
type _objInfo struct {
	// obj is the object representing the variable (most importantly,
	// obj.Type() is its type)
	obj types.Object
	// interfaceUses contains the places where the variable is used as an
	// interface value, most commonly by passing it to a function expecting
	// some typed context-interface.  (Specifically it contains the interface types
	// as which the variable is used.)
	interfaceUses map[types.Type]bool
	// methodUses is the places where the variable is used by calling a method
	// with the variable as a receiver.  (Specifically it contains the method
	// names.)
	methodUses map[string]bool
	// isCached is set if this variable is the argument to a cached function;
	// see _maybeNeededForCache.
	isCached bool
}

// _interfaceWasUsed returns true if the given interface -- a leaf-interface of
// info.obj.Type() -- was in fact used.
//
// The main cases are if we passed it to a function requiring that interface,
// or if that interface defines a method we called, but there are some others,
// discussed inline.
func (info *_objInfo) _interfaceWasUsed(typ types.Type) bool {
	iface, ok := typ.Underlying().(*types.Interface)
	if !ok { // should never happen, assume it's used
		return true
	}

	// We used the variable as this interface (or some interface which
	// contains, i.e. implements, this one)
	for used := range info.interfaceUses {
		if types.Implements(used, iface) {
			return true
		}
	}

	// We called a method defined explicitly in this interface on the variable.
	for methodName := range info.methodUses {
		if _hasExplicitMethod(iface, methodName) {
			return true
		}
	}

	return false
}

// _interfaceWasRequested returns true if the given interface was
// explicitly-requested in the type of the variable.
//
// Mainly, this means that it was one of the explicitly-requested interfaces of
// the type of the variable.  But again, there are some other cases, discussed
// inline.
func (info *_objInfo) _interfaceWasRequested(typ types.Type) bool {
	// If we used the given interface via a cast (see _markCastUsed), the type
	// of the variable may not even implement it!  We shouldn't have to request
	// it; that's the whole point of a cast.
	iface, ok := typ.Underlying().(*types.Interface)
	if ok && !types.Implements(info.obj.Type(), iface) {
		return true
	}

	// If the interface is an inline interface, but has an explicit method,
	// things get very confusing and we just give up on this check.
	inlineIface, ok := typ.(*types.Interface)
	if ok && inlineIface.NumExplicitMethods() > 0 {
		return true
	}

	// This is the main check: if we used the given type, then we have to have
	// requested it explicitly.
	for _, embed := range _explicitInterfaces(info.obj.Type(), info.obj.Pkg()) {
		if typ == embed {
			return true
		}
	}

	// Alternately, it's okay if we requested all the constituent interfaces of
	// the given type (e.g. our caller asked for `type C interface { A; B }`
	// and we asked for `A; B`).
	if named, ok := typ.(*types.Named); ok {
		// Note we calculate said "constitutent interfaces" with respect to the
		// *caller*'s package; otherwise we'd likely just get C itself.
		typMentions := _explicitInterfaces(typ, named.Obj().Pkg())
		// It only counts if "all" was at least one!  (And we don't count the
		// type itself, which we skip to avoid infinite recursion.)
		if len(typMentions) > 1 || len(typMentions) > 0 && typMentions[0] != typ {
			for _, mention := range typMentions {
				if mention != typ && !info._interfaceWasRequested(mention) {
					return false
				}
			}
			return true
		}
	}

	return false
}

// _methodWasRequested returns true if interface that provides the given method
// was explicitly-requested in the type of the variable.
//
// The nontrivial part here is finding which interface that is!
func (info *_objInfo) _methodWasRequested(methodName string) bool {
	embeds := _embedsExplicitlyContaining(info.obj.Type(), methodName)
	for _, embed := range embeds {
		if info._interfaceWasRequested(embed) {
			return true
		}
	}
	return false
}

// problems computes whether there are any problems with this variable's
// context-interfaces.  Specifically:
// - allUnused is true if the variable appears totally unused
// - unused contains any context-interfaces the variable requested in its
//   type, but did not use
// - unrequested contains any context-interfaces the variable used, but did not
//   explicitly request in its type (perhaps it requested them indirectly)
func (info *_objInfo) problems() (allUnused bool, unused, unrequested []types.Type) {
	typ := info.obj.Type()

	allLeaves := _leafInterfaces(typ)
	for _, embed := range allLeaves {
		if !info._interfaceWasUsed(embed) {
			unused = append(unused, embed)
		}
	}

	for usedInterface := range info.interfaceUses {
		for _, usedEmbed := range _explicitInterfaces(usedInterface, info.obj.Pkg()) {
			if !info._interfaceWasRequested(usedEmbed) {
				unrequested = append(unrequested, usedEmbed)
			}
		}
	}

	for usedMethod := range info.methodUses {
		if !info._methodWasRequested(usedMethod) {
			// If there are multiple distinct types explicitly containing this
			// method, and none are requested, we'll just mention all of them.
			unrequested = append(unrequested,
				_embedsExplicitlyContaining(typ, usedMethod)...)
		}
	}

	return len(unused) == len(allLeaves), unused, unrequested
}

// _runInterface lints that you don't ask for typed context interfaces you don't
// need.
//
// It isn't perfect: if you do complicated things like putting a context inside
// another type or assigning a new name to a context it may get confused.  But
// it catches most of the common cases; and if any uncommon case becomes
// common, we can add support that.
func _runInterface(pass *analysis.Pass) (interface{}, error) {
	tracker := _interfaceTracker{
		map[types.Object]*_objInfo{},
		pass.TypesInfo,
		pass.Pkg,
	}

	// First, find the identifiers we want to look at.
	for _, file := range pass.Files {
		tracker.trackIdents(file, false)
	}

	// For interface-methods, share the trackedIdents-maps so we can tret a
	// use of a particular context in one implementation of the interface as a
	// use for all the implementations.  (See callee for details.)
	tracker.identifyInterfaceMethods(pass.Files)

	// Second, see where they're used.
	for _, file := range pass.Files {
		tracker.markUses(file)
	}

	// Finally, report any errors.
	for obj, info := range tracker.trackedIdents {
		filename := pass.Fset.File(obj.Pos()).Name()
		if strings.HasSuffix(filename, "_test.go") {
			// We allow tests to ask for more interfaces than they need.
			continue
		}

		// Figure out the errors.
		allUnused, unused, unrequested := info.problems()

		// Report!
		switch {
		case allUnused:
			// In the case where the entire var is unused, clearly say so.
			// (The main unused-variable linter won't complain about function
			// arguments.)
			pass.Reportf(obj.Pos(),
				"no interfaces requested by %s are used; "+
					"remove them or rename it to _ if it's unused",
				obj.Name())
		case len(unrequested) > 0:
			// report unrequested contexts first; they may clarify why a
			// context is unused (namely you are using some part of it, not the
			// actual interface).
			pass.Reportf(obj.Pos(),
				"%s uses but does not explicitly request interface(s) %s; "+
					"add it explicitly (see ADR-429)",
				obj.Name(), _formatTypeList(unrequested, pass.Pkg))
		case len(unused) > 0:
			// If the identifier's type is an inline interface
			// it would be nice to report on the line where each embedded
			// interface is included in it.  This is surprisingly tricky to
			// implement, so we just report at the identifier itself.
			pass.Reportf(obj.Pos(),
				"%s requests but does not use interface(s) %s; "+
					"remove to use the smallest possible interface",
				obj.Name(), _formatTypeList(unused, pass.Pkg))
		}
	}

	return nil, nil
}
