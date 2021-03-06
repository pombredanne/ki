// Copyright (c) 2018, The GoKi Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package ki

import (
	"io"
	"log"
	"reflect"

	"github.com/goki/ki/kit"
)

// The Ki interface provides the core functionality for the GoKi tree --
// insipred by Qt QObject in specific and every other Tree everywhere in
// general.
//
// NOTE: The inability to have a field and a method of the same name makes it
// so you either have to use private fields in a struct that implements this
// interface (lowercase) or we have to use different names in the struct
// vs. interface.  We want to export and use the direct fields, which are easy
// to use, so we have different synonyms.
//
// Other key issues with the Ki design / Go: * All interfaces are implicitly
// pointers: this is why you have to pass args with & address of.
type Ki interface {
	// Init initializes the node -- automatically called during Add/Insert
	// Child -- sets the This pointer for this node as a Ki interface (pass
	// pointer to node as this arg) -- Go cannot always access the true
	// underlying type for structs using embedded Ki objects (when these objs
	// are receivers to methods) so we need a This interface pointer that
	// guarantees access to the Ki interface in a way that always reveals the
	// underlying type (e.g., in reflect calls).  Calls Init on Ki fields
	// within struct, sets their names to the field name, and sets us as their
	// parent.
	Init(this Ki)

	// InitName initializes this node and set its name -- used for root nodes
	// which don't otherwise have their This pointer set (otherwise typically
	// happens in Add, Insert Child).
	InitName(this Ki, name string)

	// This returns the Ki interface that guarantees access to the Ki
	// interface in a way that always reveals the underlying type
	// (e.g., in reflect calls).  Returns nil if node is nil,
	// has been destroyed, or is improperly constructed.
	This() Ki

	// ThisCheck checks that the This pointer is set and issues a warning to
	// log if not -- returns error if not set -- called when nodes are added
	// and inserted.
	ThisCheck() error

	// Type returns the underlying struct type of this node
	// (reflect.TypeOf(This).Elem()).
	Type() reflect.Type

	// TypeEmbeds tests whether this node is of the given type, or it embeds
	// that type at any level of anonymous embedding -- use Embed to get the
	// embedded struct of that type from this node.
	TypeEmbeds(t reflect.Type) bool

	// Embed returns the embedded struct of given type from this node (or nil
	// if it does not embed that type, or the type is not a Ki type -- see
	// kit.Embed for a generic interface{} version.
	Embed(t reflect.Type) Ki

	// BaseIface returns the 	base interface type for all elements
	// within this tree.  Use reflect.TypeOf((*<interface_type>)(nil)).Elem().
	// Used e.g., for determining what types of children
	// can be created (see kit.EmbedImplements for test method)
	BaseIface() reflect.Type

	// Name returns the user-defined name of the object (Node.Nm), for finding
	// elements, generating paths, IO, etc -- allows generic GUI / Text / Path
	// / etc representation of Trees.
	Name() string

	// UniqueName returns a name that is guaranteed to be non-empty and unique
	// within the children of this node (Node.UniqueNm), but starts with Name
	// or parents name if Name is empty -- important for generating unique
	// paths to definitively locate a given node in the tree (see PathUnique,
	// FindPathUnique).
	UniqueName() string

	// SetName sets the name of this node, and its unique name based on this
	// name, such that all names are unique within list of siblings of this
	// node (somewhat expensive but important, unless you definitely know that
	// the names are unique -- see SetNameRaw).  Does nothing if name is
	// already set to that value -- returns false in that case.  Does NOT
	// wrap in UpdateStart / End.
	SetName(name string) bool

	// SetNameRaw just sets the name and doesn't update the unique name --
	// only use if also/ setting unique names in some other way that is
	// guaranteed to be unique.
	SetNameRaw(name string)

	// SetUniqueName sets the unique name of this node based on given name
	// string -- does not do any further testing that the name is indeed
	// unique -- should generally only be used by UniquifyNames.
	SetUniqueName(name string)

	// UniquifyNames ensures all of my children have unique, non-empty names
	// -- duplicates are named sequentially _1, _2 etc, and empty names get a
	// name based on my name or my type name.
	UniquifyNames()

	//////////////////////////////////////////////////////////////////////////
	//  Parents

	// Parent returns the parent of this Ki (Node.Par) -- Ki has strict
	// one-parent, no-cycles structure -- see SetParent.
	Parent() Ki

	// SetParent just sets parent of node (and inherits update count from
	// parent, to keep consistent) -- does NOT remove from existing parent --
	// use Add / Insert / Delete Child functions properly move or delete nodes.
	SetParent(parent Ki)

	// IsRoot tests if this node is the root node -- checks Parent = nil.
	IsRoot() bool

	// Root returns the root object of this tree (the node with a nil parent).
	Root() Ki

	// FieldRoot returns the field root object for this node -- the node that
	// owns the branch of the tree rooted in one of its fields -- i.e., the
	// first non-Field parent node after the first Field parent node -- can be
	// nil if no such thing exists for this node.
	FieldRoot() Ki

	// IndexInParent returns our index within our parent object -- caches the
	// last value and uses that for an optimized search so subsequent calls
	// are typically quite fast.  Returns false if we don't have a parent.
	IndexInParent() (int, bool)

	// ParentLevel finds a given potential parent node recursively up the
	// hierarchy, returning level above current node that the parent was
	// found, and -1 if not found.
	ParentLevel(par Ki) int

	// HasParent checks if given node is a parent of this one (i.e.,
	// ParentLevel(par) != -1).
	HasParent(par Ki) bool

	// ParentByName finds first parent recursively up hierarchy that matches
	// given name.  Returns nil if not found.
	ParentByName(name string) Ki

	// ParentByNameTry finds first parent recursively up hierarchy that matches
	// given name -- Try version returns error on failure.
	ParentByNameTry(name string) (Ki, error)

	// ParentByType finds parent recursively up hierarchy, by type, and
	// returns nil if not found. If embeds is true, then it looks for any
	// type that embeds the given type at any level of anonymous embedding.
	ParentByType(t reflect.Type, embeds bool) Ki

	// ParentByTypeTry finds parent recursively up hierarchy, by type, and
	// returns error if not found. If embeds is true, then it looks for any
	// type that embeds the given type at any level of anonymous embedding.
	ParentByTypeTry(t reflect.Type, embeds bool) (Ki, error)

	// KiFieldByName returns field Ki element by name -- returns nil if not found.
	KiFieldByName(name string) Ki

	// KiFieldByNameTry returns field Ki element by name -- returns error if not found.
	KiFieldByNameTry(name string) (Ki, error)

	//////////////////////////////////////////////////////////////////////////
	//  Children

	// HasChildren tests whether this node has children (i.e., non-terminal).
	HasChildren() bool

	// Children returns a pointer to the slice of children (Node.Kids) -- use
	// methods on ki.Slice for further ways to access (ByName, ByType, etc).
	// Slice can be modified directly (e.g., sort, reorder) but Add* / Delete*
	// methods on parent node should be used to ensure proper tracking.
	Children() *Slice

	// Child returns the child at given index -- will panic if index is invalid.
	// See methods on ki.Slice for more ways to access.
	Child(idx int) Ki

	// ChildTry returns the child at given index -- Try version returns
	// error if index is invalid.
	// See methods on ki.Slice for more ways to access.
	ChildTry(idx int) (Ki, error)

	// ChildByName returns first element that has given name, nil if not found.
	// startIdx arg allows for optimized bidirectional find if you have
	// an idea where it might be -- can be key speedup for large lists -- pass
	// -1 to start in the middle (good default).
	ChildByName(name string, startIdx int) Ki

	// ChildByNameTry returns first element that has given name -- Try version
	// returns error message if not found.
	// startIdx arg allows for optimized bidirectional find if you have
	// an idea where it might be -- can be key speedup for large lists -- pass
	// -1 to start in the middle (good default).
	ChildByNameTry(name string, startIdx int) (Ki, error)

	// ChildByType returns first element that has given type, nil if not found.
	// If embeds is true, then it looks for any type that embeds the given type
	// at any level of anonymous embedding.
	// startIdx arg allows for optimized bidirectional find if you have
	// an idea where it might be -- can be key speedup for large lists -- pass
	// -1 to start in the middle (good default).
	ChildByType(t reflect.Type, embeds bool, startIdx int) Ki

	// ChildByTypeTry returns first element that has given name -- Try version
	// returns error message if not found.
	// If embeds is true, then it looks for any type that embeds the given type
	// at any level of anonymous embedding.
	// startIdx arg allows for optimized bidirectional find if you have
	// an idea where it might be -- can be key speedup for large lists -- pass
	// -1 to start in the middle (good default).
	ChildByTypeTry(t reflect.Type, embeds bool, startIdx int) (Ki, error)

	//////////////////////////////////////////////////////////////////////////
	//  Paths

	// Path returns path to this node from Root(), using regular user-given
	// Name's (may be empty or non-unique), with nodes separated by / and
	// fields by . -- only use for informational purposes.
	Path() string

	// PathUnique returns path to this node from Root(), using unique names,
	// with nodes separated by / and fields by . -- suitable for reliably
	// finding this node.
	PathUnique() string

	// PathFrom returns path to this node from given parent node, using
	// regular user-given Name's (may be empty or non-unique), with nodes
	// separated by / and fields by . -- only use for informational purposes.
	PathFrom(par Ki) string

	// PathFromUnique returns path to this node from given parent node, using
	// unique names, with nodes separated by / and fields by . -- suitable for
	// reliably finding this node.
	PathFromUnique(par Ki) string

	// FindPathUnique returns Ki object at given unique path, starting from
	// this node (e.g., Root()) -- if this node is not the root, then the path
	// to this node is subtracted from the start of the path if present there.
	// There is also support for [idx] index-based access for any given path
	// element, for cases when indexes are more useful than names.
	// Returns nil if not found.
	FindPathUnique(path string) Ki

	// FindPathUniqueTry returns Ki object at given unique path, starting from
	// this node (e.g., Root()) -- if this node is not the root, then the path
	// to this node is subtracted from the start of the path if present there.
	// There is also support for [idx] index-based access for any given path
	// element, for cases when indexes are more useful than names.
	// Returns error if not found.
	FindPathUniqueTry(path string) (Ki, error)

	//////////////////////////////////////////////////////////////////////////
	//  Adding, Inserting Children

	// SetChildType sets the ChildType used as a default type for creating new
	// children -- as a property called ChildType --ensures that the type is a
	// Ki type, and errors if not.
	SetChildType(t reflect.Type) error

	// AddChild adds a new child at end of children list -- if child is in an
	// existing tree, it is removed from that parent, and a NodeMoved signal
	// is emitted for the child -- UniquifyNames is called after adding to
	// ensure name is unique (assumed to already have a name).
	AddChild(kid Ki) error

	// AddChildFast adds a new child at end of children list in the fastest
	// way possible -- saves about 30% of time overall --
	// assumes InitName has already been run, and doesn't
	// ensure names are unique, or run other checks, including if child
	// already has a parent.  Only use if you really need the speed..
	AddChildFast(kid Ki)

	// InsertChild adds a new child at given position in children list -- if
	// child is in an existing tree, it is removed from that parent, and a
	// NodeMoved signal is emitted for the child -- UniquifyNames is called
	// after adding to ensure name is unique (assumed to already have a name).
	InsertChild(kid Ki, at int) error

	// NewOfType creates a new child of given type -- if nil, uses ChildType,
	// else uses the same type as this struct.
	NewOfType(typ reflect.Type) Ki

	// AddNewChild creates a new child of given type -- if nil, uses
	// ChildType, else type of this struct -- and add at end of children list
	// -- assigns name (can be empty) and enforces UniqueName.
	AddNewChild(typ reflect.Type, name string) Ki

	// InsertNewChild creates a new child of given type -- if nil, uses
	// ChildType, else type of this struct -- and add at given position in
	// children list -- assigns name (can be empty) and enforces UniqueName.
	InsertNewChild(typ reflect.Type, at int, name string) Ki

	// SetChild sets child at given index to be the given item -- if name is
	// non-empty then it sets the name of the child as well -- just calls Init
	// (or InitName) on the child, and SetParent -- does NOT uniquify the
	// names -- this is for high-volume child creation -- call UniquifyNames
	// afterward if needed, but better to ensure that names are unique up front.
	SetChild(kid Ki, idx int, name string) error

	// MoveChild moves child from one position to another in the list of
	// children (see also corresponding Slice method, which does not
	// signal, like this one does).  Returns error if either index is invalid.
	MoveChild(from, to int) error

	// SwapChildren swaps children between positions (see also corresponding
	// Slice method which does not signal like this one does).  Returns error if
	// either index is invalid.
	SwapChildren(i, j int) error

	// SetNChildren ensures that there are exactly n children, deleting any
	// extra, and creating any new ones, using AddNewChild with given type and
	// naming according to nameStubX where X is the index of the child.
	//
	// IMPORTANT: returns whether any modifications were made (mods) AND if
	// that is true, the result from the corresponding UpdateStart call --
	// UpdateEnd is NOT called, allowing for further subsequent updates before
	// you call UpdateEnd(updt)
	//
	// Note that this does not ensure existing children are of given type, or
	// change their names, or call UniquifyNames -- use ConfigChildren for
	// those cases -- this function is for simpler cases where a parent uses
	// this function consistently to manage children all of the same type.
	SetNChildren(n int, typ reflect.Type, nameStub string) (mods, updt bool)

	// ConfigChildren configures children according to given list of
	// type-and-name's -- attempts to have minimal impact relative to existing
	// items that fit the type and name constraints (they are moved into the
	// corresponding positions), and any extra children are removed, and new
	// ones added, to match the specified config.  If uniqNm, then names
	// represent UniqueNames (this results in Name == UniqueName for created
	// children).
	//
	// IMPORTANT: returns whether any modifications were made (mods) AND if
	// that is true, the result from the corresponding UpdateStart call --
	// UpdateEnd is NOT called, allowing for further subsequent updates before
	// you call UpdateEnd(updt).
	ConfigChildren(config kit.TypeAndNameList, uniqNm bool) (mods, updt bool)

	//////////////////////////////////////////////////////////////////////////
	//  Deleting Children

	// DeleteChildAtIndex deletes child at given index (returns error for
	// invalid index) -- if child's parent = this node, then will call
	// SetParent(nil), so to transfer to another list, set new parent first --
	// destroy will add removed child to deleted list, to be destroyed later
	// -- otherwise child remains intact but parent is nil -- could be
	// inserted elsewhere.
	DeleteChildAtIndex(idx int, destroy bool) error

	// DeleteChild deletes child node, returning error if not found in
	// Children.  If child's parent = this node, then will call
	// SetParent(nil), so to transfer to another list, set new parent
	// first. See DeleteChildAtIndex for destroy info.
	DeleteChild(child Ki, destroy bool) error

	// DeleteChildByName deletes child node by name -- returns child, error
	// if not found -- if child's parent = this node, then will call
	// SetParent(nil), so to transfer to another list, set new parent first.
	// See DeleteChildAtIndex for destroy info.
	DeleteChildByName(name string, destroy bool) (Ki, error)

	// DeleteChildren deletes all children nodes -- destroy will add removed
	// children to deleted list, to be destroyed later -- otherwise children
	// remain intact but parent is nil -- could be inserted elsewhere, but you
	// better have kept a slice of them before calling this.
	DeleteChildren(destroy bool)

	// Delete deletes this node from its parent children list -- destroy will
	// add removed child to deleted list, to be destroyed later -- otherwise
	// child remains intact but parent is nil -- could be inserted elsewhere.
	Delete(destroy bool)

	// Destroy calls DisconnectAll to cut all pointers and signal connections,
	// and remove all children and their childrens-children, etc.
	Destroy()

	//////////////////////////////////////////////////////////////////////////
	//  Flags

	// Flag returns an atomically safe copy of the bit flags for this node --
	// can use bitflag package to check lags.
	// See Flags type for standard values used in Ki Node --
	// can be extended from FlagsN up to 64 bit capacity.
	// Note that we must always use atomic access as *some* things need to be atomic,
	// and with bits, that means that *all* access needs to be atomic,
	// as you cannot atomically update just a single bit.
	Flags() int64

	// HasFlag checks if flag is set
	// using atomic, safe for concurrent access
	HasFlag(flag int) bool

	// HasAnyFlag checks if *any* of a set of flags is set (logical OR)
	// using atomic, safe for concurrent access
	HasAnyFlag(flag ...int) bool

	// HasAllFlags checks if *all* of a set of flags is set (logical AND)
	// using atomic, safe for concurrent access
	HasAllFlags(flag ...int) bool

	// SetFlag sets the given flag(s)
	// using atomic, safe for concurrent access
	SetFlag(flag ...int)

	// SetFlagState sets the given flag(s) to given state
	// using atomic, safe for concurrent access
	SetFlagState(on bool, flag ...int)

	// SetFlagMask sets the given flags as a mask
	// using atomic, safe for concurrent access
	SetFlagMask(mask int64)

	// ClearFlag clears the given flag(s)
	// using atomic, safe for concurrent access
	ClearFlag(flag ...int)

	// ClearFlagMask clears the given flags as a bitmask
	// using atomic, safe for concurrent access
	ClearFlagMask(mask int64)

	// IsField checks if this is a field on a parent struct (via IsField
	// Flag), as opposed to a child in Children -- Ki nodes can be added as
	// fields to structs and they are automatically parented and named with
	// field name during Init function -- essentially they function as fixed
	// children of the parent struct, and are automatically included in
	// FuncDown* traversals, etc -- see also FunFields.
	IsField() bool

	// IsUpdating checks if node is currently updating.
	IsUpdating() bool

	// OnlySelfUpdate checks if this node only applies UpdateStart / End logic
	// to itself, not its children (which is the default) (via Flag of same
	// name) -- useful for a parent node that has a different function than
	// its children.
	OnlySelfUpdate() bool

	// SetOnlySelfUpdate sets the OnlySelfUpdate flag -- see OnlySelfUpdate
	// method and flag.
	SetOnlySelfUpdate()

	// IsDeleted checks if this node has just been deleted (within last update
	// cycle), indicated by the NodeDeleted flag which is set when the node is
	// deleted, and is cleared at next UpdateStart call.
	IsDeleted() bool

	// IsDestroyed checks if this node has been destroyed -- the NodeDestroyed
	// flag is set at start of Destroy function -- the Signal Emit process
	// checks for destroyed receiver nodes and removes connections to them
	// automatically -- other places where pointers to potentially destroyed
	// nodes may linger should also check this flag and reset those pointers.
	IsDestroyed() bool

	//////////////////////////////////////////////////////////////////////////
	//  Property interface with inheritance -- nodes can inherit props from parents

	// Properties (Node.Props) tell the GoGi GUI or other frameworks operating
	// on Trees about special features of each node -- functions below support
	// inheritance up Tree -- see kit convert.go for robust convenience
	// methods for converting interface{} values to standard types.
	Properties() *Props

	// SetProp sets given property key to value val -- initializes property
	// map if nil.
	SetProp(key string, val interface{})

	// SetProps sets a whole set of properties, and optionally sets the
	// updated flag and triggers an UpdateSig.
	SetProps(props Props, update bool)

	// SetPropUpdate sets given property key to value val, with update
	// notification (sets PropUpdated and emits UpdateSig) so other nodes
	// receiving update signals from this node can update to reflect these
	// changes.
	SetPropUpdate(key string, val interface{})

	// SetPropChildren sets given property key to value val for all Children.
	SetPropChildren(key string, val interface{})

	// Prop gets property value from key.
	Prop(key string) (interface{}, bool)

	// KnownProp gets property value from key that is known to exist --
	// returns nil if it actually doesn't -- less cumbersome for conversions.
	KnownProp(key string) interface{}

	// PropInherit gets property value from key with options for inheriting
	// property from parents and / or type-level properties.  If inherit, then
	// checks all parents.  If typ then checks property on type as well
	// (registered via KiT type registry).  Returns false if not set anywhere.
	PropInherit(key string, inherit, typ bool) (interface{}, bool)

	// DeleteProp deletes property key on this node.
	DeleteProp(key string)

	// DeleteAllProps deletes all properties on this node -- just makes a new
	// Props map -- can specify the capacity of the new map (0 means set to
	// nil instead of making a new one -- most efficient if potentially no
	// properties will be set).
	DeleteAllProps(cap int)

	// CopyPropsFrom copies our properties from another node -- if deep then
	// does a deep copy -- otherwise copied map just points to same values in
	// the original map (and we don't reset our map first -- call
	// DeleteAllProps to do that -- deep copy uses gob encode / decode --
	// usually not needed).
	CopyPropsFrom(from Ki, deep bool) error

	// PropTag returns the name to look for in type properties, for types
	// that are valid options for values that can be set in Props.  For example
	// in GoGi, it is "style-props" which is then set for all types that can
	// be used in a style (colors, enum options, etc)
	PropTag() string

	//////////////////////////////////////////////////////////////////////////
	//  Tree walking and Paths
	//   note: always put functions last -- looks better for inline functions

	// FuncFields calls function on all Ki fields within this node.
	FuncFields(level int, data interface{}, fun Func)

	// GoFuncFields calls concurrent goroutine function on all Ki fields
	// within this node.
	GoFuncFields(level int, data interface{}, fun Func)

	// FuncUp calls function on given node and all the way up to its parents,
	// and so on -- sequentially all in current go routine (generally
	// necessary for going up, which is typicaly quite fast anyway) -- level
	// is incremented after each step (starts at 0, goes up), and passed to
	// function -- returns false if fun aborts with false, else true.
	FuncUp(level int, data interface{}, fun Func) bool

	// FuncUpParent calls function on parent of node and all the way up to its
	// parents, and so on -- sequentially all in current go routine (generally
	// necessary for going up, which is typicaly quite fast anyway) -- level
	// is incremented after each step (starts at 0, goes up), and passed to
	// function -- returns false if fun aborts with false, else true.
	FuncUpParent(level int, data interface{}, fun Func) bool

	// FuncDownMeFirst calls function on this node (MeFirst) and then call
	// FuncDownMeFirst on all the children -- sequentially all in current go
	// routine -- level var is incremented before calling children -- if fun
	// returns false then any further traversal of that branch of the tree is
	// aborted, but other branches continue -- i.e., if fun on current node
	// returns false, then returns false and children are not processed
	// further -- this is the fastest, most natural form of traversal.
	FuncDownMeFirst(level int, data interface{}, fun Func) bool

	// FuncDownDepthFirst calls FuncDownDepthFirst on all children, then calls
	// function on this node -- sequentially all in current go routine --
	// level var is incremented before calling children -- runs
	// doChildTestFunc on each child first to determine if it should process
	// that child, and if that returns true, then it calls FuncDownDepthFirst
	// on that child.
	FuncDownDepthFirst(level int, data interface{}, doChildTestFunc Func, fun Func)

	// FuncDownBreadthFirst calls function on all children, then calls
	// FuncDownBreadthFirst on all the children -- does NOT call on first node
	// where this method is first called, due to nature of recursive logic --
	// level var is incremented before calling children -- if fun returns
	// false then any further traversal of that branch of the tree is aborted,
	// but other branches can continue.
	FuncDownBreadthFirst(level int, data interface{}, fun Func)

	// GoFuncDown calls concurrent goroutine function on given node and all
	// the way down to its children, and so on -- does not wait for completion
	// of the go routines -- returns immediately.
	GoFuncDown(level int, data interface{}, fun Func)

	// todo: GoFuncDownWait calls concurrent goroutine function on given node and
	// all the way down to its children, and so on -- does wait for the
	// completion of the go routines before returning.
	// GoFuncDownWait(level int, data interface{}, fun Func)

	//////////////////////////////////////////////////////////////////////////
	//  State update signaling -- automatically consolidates all changes across
	//   levels so there is only one update at end (optionally per node or only
	//   at highest level)
	//   All modification starts with UpdateStart() and ends with UpdateEnd()

	// NodeSignal returns the main signal for this node that is used for
	// update, child signals.
	NodeSignal() *Signal

	// UpdateStart should be called when starting to modify the tree (state or
	// structure) -- returns whether this node was first to set the Updating
	// flag (if so, all children have their Updating flag set -- pass the
	// result to UpdateEnd -- automatically determines the highest level
	// updated, within the normal top-down updating sequence -- can be called
	// multiple times at multiple levels -- it is essential to ensure that all
	// such Start's have an End!  Usage:
	//
	//   updt := n.UpdateStart()
	//   ... code
	//   n.UpdateEnd(updt)
	// or
	//   updt := n.UpdateStart()
	//   defer n.UpdateEnd(updt)
	//   ... code
	UpdateStart() bool

	// UpdateEnd should be called when done updating after an UpdateStart, and
	// passed the result of the UpdateStart call -- if this is true, the
	// NodeSignalUpdated signal will be emitted and the Updating flag will be
	// cleared, and DestroyDeleted called -- otherwise it is a no-op.
	UpdateEnd(updt bool)

	// UpdateEndNoSig is just like UpdateEnd except it does not emit a
	// NodeSignalUpdated signal -- use this for situations where updating is
	// already known to be in progress and the signal would be redundant.
	UpdateEndNoSig(updt bool)

	// UpdateSig just emits a NodeSignalUpdated if the Updating flag is not
	// set -- use this to trigger an update of a given node when there aren't
	// any structural changes and you don't need to prevent any lower-level
	// updates -- much more efficient than a pair of UpdateStart /
	// UpdateEnd's.  Returns true if an update signal was sent.
	UpdateSig() bool

	// UpdateReset resets Updating flag for this node and all children -- in
	// case they are out-of-sync due to more complex tree maninpulations --
	// only call at a known point of non-updating.
	UpdateReset()

	// Disconnect disconnects node -- reset all ptrs to nil, and
	// DisconnectAll() signals -- e.g., for freeing up all connnections so
	// node can be destroyed and making GC easier.
	Disconnect()

	// DisconnectAll disconnects all the way from me down the tree.
	DisconnectAll()

	//////////////////////////////////////////////////////////////////////////
	//  Field Value setting with notification

	// SetField sets given field name to given value, using very robust
	// conversion routines to e.g., convert from strings to numbers, and
	// vice-versa, automatically.  Returns error if not successfully set.
	// wrapped in UpdateStart / End and sets the FieldUpdated flag.
	SetField(field string, val interface{}) error

	// SetFieldDown sets given field name to given value, all the way down the
	// tree from me -- wrapped in UpdateStart / End.
	SetFieldDown(field string, val interface{})

	// SetFieldUp sets given field name to given value, all the way up the
	// tree from me -- wrapped in UpdateStart / End.
	SetFieldUp(field string, val interface{})

	// FieldByName returns field value by name (can be any type of field --
	// see KiFieldByName for Ki fields) -- returns nil if not found.
	FieldByName(field string) interface{}

	// FieldByNameTry returns field value by name (can be any type of field --
	// see KiFieldByName for Ki fields) -- returns error if not found.
	FieldByNameTry(field string) (interface{}, error)

	// FieldTag returns given field tag for that field, or empty string if not set.
	FieldTag(field, tag string) string

	//////////////////////////////////////////////////////////////////////////
	//  Deep Copy of Trees

	// CopyFrom another Ki node.  The Ki copy function recreates the entire
	// tree in the copy, duplicating children etc.  It is very efficient by
	// using the ConfigChildren method which attempts to preserve any existing
	// nodes in the destination if they have the same name and type -- so a
	// copy from a source to a target that only differ minimally will be
	// minimally destructive.  Only copies to same types are supported.
	// Pointers (Ptr) are copied by saving the current UniquePath and then
	// SetPtrsFmPaths is called.  Signal connections are NOT copied.  No other
	// Ki pointers are copied, and the field tag copy:"-" can be added for any
	// other fields that should not be copied (unexported, lower-case fields
	// are not copyable).
	//
	// When nodes are copied from one place to another within the same overall
	// tree, paths are updated so that pointers to items within the copied
	// sub-tree are updated to the new location there (i.e., the path to the
	// old loation is replaced with that of the new destination location),
	// whereas paths outside of the copied location are not changed and point
	// as before.  See also MoveTo function for moving nodes to other parts of
	// the tree.  Sequence of functions is: GetPtrPaths on from, CopyFromRaw,
	// UpdtPtrPaths, then SetPtrsFmPaths.
	CopyFrom(from Ki) error

	// Clone creates and returns a deep copy of the tree from this node down.
	// Any pointers within the cloned tree will correctly point within the new
	// cloned tree (see Copy info).
	Clone() Ki

	// CopyFromRaw performs a raw copy that just does the deep copy of the
	// bits and doesn't do anything with pointers.
	CopyFromRaw(from Ki) error

	// GetPtrPaths gets all Ptr path strings -- walks the tree down from
	// current node and calls GetPath on all Ptr fields -- this is called
	// prior to copying / moving.
	GetPtrPaths()

	// SetPtrsFmPaths walks the tree down from current node and calls
	// PtrFromPath on all Ptr fields found -- called after Copy, Unmarshal* to
	// recover pointers after entire structure is in place -- see
	// UnmarshalPost.
	SetPtrsFmPaths()

	// UpdatePtrPaths updates Ptr paths, replacing any occurrence of oldPath with
	// newPath, optionally only at the start of the path (typically true) --
	// for all nodes down from this one.
	UpdatePtrPaths(oldPath, newPath string, startOnly bool)

	//////////////////////////////////////////////////////////////////////////
	//  IO: for JSON and XML formats -- see also Slice, Ptr
	//  see https://github.com/goki/ki/wiki/Naming for IO naming conventions

	// WriteJSON writes the tree to an io.Writer, using MarshalJSON -- also
	// saves a critical starting record that allows file to be loaded de-novo
	// and recreate the proper root type for the tree.
	WriteJSON(writer io.Writer, indent bool) error

	// SaveJSON saves the tree to a JSON-encoded file, using WriteJSON.
	SaveJSON(filename string) error

	// ReadJSON reads and unmarshals tree starting at this node, from a
	// JSON-encoded byte stream via io.Reader.  First element in the stream
	// must be of same type as this node -- see ReadNewJSON function to
	// construct a new tree.  Uses ConfigureChildren to minimize changes from
	// current tree relative to loading one -- wraps UnmarshalJSON and calls
	// UnmarshalPost to recover pointers from paths.
	ReadJSON(reader io.Reader) error

	// OpenJSON opens file over this tree from a JSON-encoded file -- see
	// ReadJSON for details, and OpenNewJSON for opening an entirely new tree.
	OpenJSON(filename string) error

	// WriteXML writes the tree to an XML-encoded byte string over io.Writer
	// using MarshalXML.
	WriteXML(writer io.Writer, indent bool) error

	// ReadXML reads the tree from an XML-encoded byte string over io.Reader, calls
	// UnmarshalPost to recover pointers from paths.
	ReadXML(reader io.Reader) error

	// ParentAllChildren walks the tree down from current node and call
	// SetParent on all children -- needed after an Unmarshal.
	ParentAllChildren()

	// UnmarshalPost must be called after an Unmarshal -- calls
	// SetPtrsFmPaths and ParentAllChildren.
	UnmarshalPost()
}

// see node.go for struct implementing this interface

// IMPORTANT: all types must initialize entry in package kit Types Registry
//
// var KiT_TypeName = kit.Types.AddType(&TypeName{})

// Func is a function to call on ki objects walking the tree -- return bool
// = false means don't continue processing this branch of the tree, but other
// branches can continue.
type Func func(k Ki, level int, data interface{}) bool

// KiType is a Ki reflect.Type, suitable for checking for Type.Implements.
var KiType = reflect.TypeOf((*Ki)(nil)).Elem()

// IsKi returns true if the given type implements the Ki interface at any
// level of embedded structure.
func IsKi(typ reflect.Type) bool {
	if typ == nil {
		return false
	}
	return kit.EmbedImplements(typ, KiType)
}

// NewOfType makes a new Ki struct of given type -- must be a Ki type -- will
// return nil if not.
func NewOfType(typ reflect.Type) Ki {
	nkid := reflect.New(typ).Interface()
	kid, ok := nkid.(Ki)
	if !ok {
		log.Printf("ki.NewOfType: type %v cannot be converted into a Ki interface type\n", typ.String())
		return nil
	}
	return kid
}

// Flags are bit flags for efficient core state of nodes -- see bitflag
// package for using these ordinal values to manipulate bit flag field.
type Flags int32

const (
	// IsField indicates a node is a field in its parent node, not a child in children.
	IsField Flags = iota

	// Updating flag is set at UpdateStart and cleared if we were the first
	// updater at UpdateEnd.
	Updating

	// OnlySelfUpdate means that the UpdateStart / End logic only applies to
	// this node in isolation, not to its children -- useful for a parent node
	// that has a different functional role than its children.
	OnlySelfUpdate

	// following flags record what happened to a given node since the last
	// Update signal -- they are cleared at first UpdateStart and valid after
	// UpdateEnd

	// NodeAdded means a node was added to new parent.
	NodeAdded

	// NodeCopied means node was copied from other node.
	NodeCopied

	// NodeMoved means node was moved in the tree, or to a new tree.
	NodeMoved

	// NodeDeleted means this node has been deleted.
	NodeDeleted

	// NodeDestroyed means this node has been destroyed -- do not trigger any
	// more update signals on it.
	NodeDestroyed

	// ChildAdded means one or more new children were added to the node.
	ChildAdded

	// ChildMoved means one or more children were moved within the node.
	ChildMoved

	// ChildDeleted means one or more children were deleted from the node.
	ChildDeleted

	// ChildrenDeleted means all children were deleted.
	ChildrenDeleted

	// FieldUpdated means a field was updated.
	FieldUpdated

	// PropUpdated means a property was set.
	PropUpdated

	// FlagsN is total number of flags used by base Ki Node -- can extend from
	// here up to 64 bits.
	FlagsN

	// NodeUpdateFlagsMask is a mask for all node updates.
	NodeUpdateFlagsMask = (1 << uint32(NodeAdded)) | (1 << uint32(NodeCopied)) | (1 << uint32(NodeMoved))

	// ChildUpdateFlagsMask is a mask for all child updates.
	ChildUpdateFlagsMask = (1 << uint32(ChildAdded)) | (1 << uint32(ChildMoved)) | (1 << uint32(ChildDeleted)) | (1 << uint32(ChildrenDeleted))

	// StruUpdateFlagsMask is a mask for all structural changes update flags.
	StruUpdateFlagsMask = NodeUpdateFlagsMask | ChildUpdateFlagsMask | (1 << uint32(NodeDeleted))

	// ValUpdateFlagsMask is a mask for all non-structural, value-only changes update flags.
	ValUpdateFlagsMask = (1 << uint32(FieldUpdated)) | (1 << uint32(PropUpdated))

	// UpdateFlagsMask is a Mask for all the update flags -- destroyed is
	// excluded b/c otherwise it would get cleared.
	UpdateFlagsMask = StruUpdateFlagsMask | ValUpdateFlagsMask
)

//go:generate stringer -type=Flags

var KiT_Flags = kit.Enums.AddEnum(FlagsN, true, nil) // true = bitflags
