package proc

import (
	"bytes"
	"debug/dwarf"
	"encoding/binary"
	"errors"
	"fmt"
	"go/constant"
	"go/parser"
	"go/token"
	"math"
	"reflect"
	"sort"
	"strings"
	"unsafe"

	"github.com/go-delve/delve/pkg/dwarf/godwarf"
	"github.com/go-delve/delve/pkg/dwarf/op"
	"github.com/go-delve/delve/pkg/dwarf/reader"
)

const (
	maxErrCount = 3 // Max number of read errors to accept while evaluating slices, arrays and structs

	maxArrayStridePrefetch = 1024 // Maximum size of array stride for which we will prefetch the array contents

	chanRecv = "chan receive"
	chanSend = "chan send"

	hashTophashEmpty = 0 // used by map reading code, indicates an empty bucket
	hashMinTopHash   = 4 // used by map reading code, indicates minimum value of tophash that isn't empty or evacuated

	maxFramePrefetchSize = 1 * 1024 * 1024 // Maximum prefetch size for a stack frame

	maxMapBucketsFactor = 100 // Maximum numbers of map buckets to read for every requested map entry when loading variables through (*EvalScope).LocalVariables and (*EvalScope).FunctionArguments.
)

type floatSpecial uint8

const (
	// FloatIsNormal means the value is a normal float.
	FloatIsNormal floatSpecial = iota
	// FloatIsNaN means the float is a special NaN value.
	FloatIsNaN
	// FloatIsPosInf means the float is a special positive inifitiy value.
	FloatIsPosInf
	// FloatIsNegInf means the float is a special negative infinity value.
	FloatIsNegInf
)

type variableFlags uint16

const (
	// VariableEscaped is set for local variables that escaped to the heap
	//
	// The compiler performs escape analysis on local variables, the variables
	// that may outlive the stack frame are allocated on the heap instead and
	// only the address is recorded on the stack. These variables will be
	// marked with this flag.
	VariableEscaped variableFlags = (1 << iota)
	// VariableShadowed is set for local variables that are shadowed by a
	// variable with the same name in another scope
	VariableShadowed
	// VariableConstant means this variable is a constant value
	VariableConstant
	// VariableArgument means this variable is a function argument
	VariableArgument
	// VariableReturnArgument means this variable is a function return value
	VariableReturnArgument
)

// Variable represents a variable. It contains the address, name,
// type and other information parsed from both the Dwarf information
// and the memory of the debugged process.
// If OnlyAddr is true, the variables value has not been loaded.
type Variable struct {
	Addr      uintptr
	OnlyAddr  bool
	Name      string
	DwarfType godwarf.Type
	RealType  godwarf.Type
	Kind      reflect.Kind
	mem       MemoryReadWriter
	bi        *BinaryInfo

	Value        constant.Value
	FloatSpecial floatSpecial

	Len int64
	Cap int64

	Flags variableFlags

	// Base address of arrays, Base address of the backing array for slices (0 for nil slices)
	// Base address of the backing byte array for strings
	// address of the struct backing chan and map variables
	// address of the function entry point for function variables (0 for nil function pointers)
	Base      uintptr
	stride    int64
	fieldType godwarf.Type

	// number of elements to skip when loading a map
	mapSkip int

	Children []Variable

	loaded     bool
	Unreadable error

	LocationExpr string // location expression
	DeclLine     int64  // line number of this variable's declaration
}

// LoadConfig controls how variables are loaded from the targets memory.
type LoadConfig struct {
	// FollowPointers requests pointers to be automatically dereferenced.
	FollowPointers bool
	// MaxVariableRecurse is how far to recurse when evaluating nested types.
	MaxVariableRecurse int
	// MaxStringLen is the maximum number of bytes read from a string
	MaxStringLen int
	// MaxArrayValues is the maximum number of elements read from an array, a slice or a map.
	MaxArrayValues int
	// MaxStructFields is the maximum number of fields read from a struct, -1 will read all fields.
	MaxStructFields int

	// MaxMapBuckets is the maximum number of map buckets to read before giving up.
	// A value of 0 will read as many buckets as necessary until the entire map
	// is read or MaxArrayValues is reached.
	//
	// Loading a map is an operation that issues O(num_buckets) operations.
	// Normally the number of buckets is proportional to the number of elements
	// in the map, since the runtime tries to keep the load factor of maps
	// between 40% and 80%.
	//
	// It is possible, however, to create very sparse maps either by:
	// a) adding lots of entries to a map and then deleting most of them, or
	// b) using the make(mapType, N) expression with a very large N
	//
	// When this happens delve will have to scan many empty buckets to find the
	// few entries in the map.
	// MaxMapBuckets can be set to avoid annoying slowdowns␣while reading
	// very sparse maps.
	//
	// Since there is no good way for a user of delve to specify the value of
	// MaxMapBuckets, this field is not actually exposed through the API.
	// Instead (*EvalScope).LocalVariables and  (*EvalScope).FunctionArguments
	// set this field automatically to MaxArrayValues * maxMapBucketsFactor.
	// Every other invocation uses the default value of 0, obtaining the old behavior.
	// In practice this means that debuggers using the ListLocalVars or
	// ListFunctionArgs API will not experience a massive slowdown when a very
	// sparse map is in scope, but evaluating a single variable will still work
	// correctly, even if the variable in question is a very sparse map.
	MaxMapBuckets int
}

var loadSingleValue = LoadConfig{false, 0, 64, 0, 0, 0}
var loadFullValue = LoadConfig{true, 1, 64, 64, -1, 0}

// G status, from: src/runtime/runtime2.go
const (
	Gidle           uint64 = iota // 0
	Grunnable                     // 1 runnable and on a run queue
	Grunning                      // 2
	Gsyscall                      // 3
	Gwaiting                      // 4
	GmoribundUnused               // 5 currently unused, but hardcoded in gdb scripts
	Gdead                         // 6
	Genqueue                      // 7 Only the Gscanenqueue is used.
	Gcopystack                    // 8 in this state when newstack is moving the stack
)

// G represents a runtime G (goroutine) structure (at least the
// fields that Delve is interested in).
type G struct {
	ID         int    // Goroutine ID
	PC         uint64 // PC of goroutine when it was parked.
	SP         uint64 // SP of goroutine when it was parked.
	BP         uint64 // BP of goroutine when it was parked (go >= 1.7).
	GoPC       uint64 // PC of 'go' statement that created this goroutine.
	StartPC    uint64 // PC of the first function run on this goroutine.
	WaitReason string // Reason for goroutine being parked.
	Status     uint64
	stkbarVar  *Variable // stkbar field of g struct
	stkbarPos  int       // stkbarPos field of g struct
	stackhi    uint64    // value of stack.hi
	stacklo    uint64    // value of stack.lo

	SystemStack bool // SystemStack is true if this goroutine is currently executing on a system stack.

	// Information on goroutine location
	CurrentLoc Location

	// Thread that this goroutine is currently allocated to
	Thread Thread

	variable *Variable

	Unreadable error // could not read the G struct
}

type Ancestor struct {
	ID         int64 // Goroutine ID
	Unreadable error
	pcsVar     *Variable
}

// EvalScope is the scope for variable evaluation. Contains the thread,
// current location (PC), and canonical frame address.
type EvalScope struct {
	Location
	Regs    op.DwarfRegisters
	Mem     MemoryReadWriter // Target's memory
	Gvar    *Variable
	BinInfo *BinaryInfo

	frameOffset int64

	aordr *dwarf.Reader // extra reader to load DW_AT_abstract_origin entries, do not initialize
}

// IsNilErr is returned when a variable is nil.
type IsNilErr struct {
	name string
}

func (err *IsNilErr) Error() string {
	return fmt.Sprintf("%s is nil", err.name)
}

func globalScope(bi *BinaryInfo, mem MemoryReadWriter) *EvalScope {
	return &EvalScope{Location: Location{}, Regs: op.DwarfRegisters{StaticBase: bi.staticBase}, Mem: mem, Gvar: nil, BinInfo: bi, frameOffset: 0}
}

func (scope *EvalScope) newVariable(name string, addr uintptr, dwarfType godwarf.Type, mem MemoryReadWriter) *Variable {
	return newVariable(name, addr, dwarfType, scope.BinInfo, mem)
}

func newVariableFromThread(t Thread, name string, addr uintptr, dwarfType godwarf.Type) *Variable {
	return newVariable(name, addr, dwarfType, t.BinInfo(), t)
}

func (v *Variable) newVariable(name string, addr uintptr, dwarfType godwarf.Type, mem MemoryReadWriter) *Variable {
	return newVariable(name, addr, dwarfType, v.bi, mem)
}

func newVariable(name string, addr uintptr, dwarfType godwarf.Type, bi *BinaryInfo, mem MemoryReadWriter) *Variable {
	if styp, isstruct := dwarfType.(*godwarf.StructType); isstruct && !strings.Contains(styp.Name, "<") && !strings.Contains(styp.Name, "{") {
		// For named structs the compiler will emit a DW_TAG_structure_type entry
		// and a DW_TAG_typedef entry.
		//
		// Normally variables refer to the typedef entry but sometimes global
		// variables will refer to the struct entry incorrectly.
		// Also the runtime type offset resolution (runtimeTypeToDIE) will return
		// the struct entry directly.
		//
		// In both cases we prefer to have a typedef type for consistency's sake.
		//
		// So we wrap all struct types into a fake typedef type except for:
		// a. types not defined by go
		// b. anonymous struct types (they contain the '{' character)
		// c. Go internal struct types used to describe maps (they contain the '<'
		// character).
		cu := bi.findCompileUnitForOffset(dwarfType.Common().Offset)
		if cu != nil && cu.isgo {
			dwarfType = &godwarf.TypedefType{
				CommonType: *(dwarfType.Common()),
				Type:       dwarfType,
			}
		}
	}

	v := &Variable{
		Name:      name,
		Addr:      addr,
		DwarfType: dwarfType,
		mem:       mem,
		bi:        bi,
	}

	v.RealType = resolveTypedef(v.DwarfType)

	switch t := v.RealType.(type) {
	case *godwarf.PtrType:
		v.Kind = reflect.Ptr
		if _, isvoid := t.Type.(*godwarf.VoidType); isvoid {
			v.Kind = reflect.UnsafePointer
		}
	case *godwarf.ChanType:
		v.Kind = reflect.Chan
		if v.Addr != 0 {
			v.loadChanInfo()
		}
	case *godwarf.MapType:
		v.Kind = reflect.Map
	case *godwarf.StringType:
		v.Kind = reflect.String
		v.stride = 1
		v.fieldType = &godwarf.UintType{BasicType: godwarf.BasicType{CommonType: godwarf.CommonType{ByteSize: 1, Name: "byte"}, BitSize: 8, BitOffset: 0}}
		if v.Addr != 0 {
			v.Base, v.Len, v.Unreadable = readStringInfo(v.mem, v.bi.Arch, v.Addr)
		}
	case *godwarf.SliceType:
		v.Kind = reflect.Slice
		if v.Addr != 0 {
			v.loadSliceInfo(t)
		}
	case *godwarf.InterfaceType:
		v.Kind = reflect.Interface
	case *godwarf.StructType:
		v.Kind = reflect.Struct
	case *godwarf.ArrayType:
		v.Kind = reflect.Array
		v.Base = v.Addr
		v.Len = t.Count
		v.Cap = -1
		v.fieldType = t.Type
		v.stride = 0

		if t.Count > 0 {
			v.stride = t.ByteSize / t.Count
		}
	case *godwarf.ComplexType:
		switch t.ByteSize {
		case 8:
			v.Kind = reflect.Complex64
		case 16:
			v.Kind = reflect.Complex128
		}
	case *godwarf.IntType:
		v.Kind = reflect.Int
	case *godwarf.UintType:
		v.Kind = reflect.Uint
	case *godwarf.FloatType:
		switch t.ByteSize {
		case 4:
			v.Kind = reflect.Float32
		case 8:
			v.Kind = reflect.Float64
		}
	case *godwarf.BoolType:
		v.Kind = reflect.Bool
	case *godwarf.FuncType:
		v.Kind = reflect.Func
	case *godwarf.VoidType:
		v.Kind = reflect.Invalid
	case *godwarf.UnspecifiedType:
		v.Kind = reflect.Invalid
	default:
		v.Unreadable = fmt.Errorf("Unknown type: %T", t)
	}

	return v
}

func resolveTypedef(typ godwarf.Type) godwarf.Type {
	for {
		switch tt := typ.(type) {
		case *godwarf.TypedefType:
			typ = tt.Type
		case *godwarf.QualType:
			typ = tt.Type
		default:
			return typ
		}
	}
}

func newConstant(val constant.Value, mem MemoryReadWriter) *Variable {
	v := &Variable{Value: val, mem: mem, loaded: true}
	switch val.Kind() {
	case constant.Int:
		v.Kind = reflect.Int
	case constant.Float:
		v.Kind = reflect.Float64
	case constant.Bool:
		v.Kind = reflect.Bool
	case constant.Complex:
		v.Kind = reflect.Complex128
	case constant.String:
		v.Kind = reflect.String
		v.Len = int64(len(constant.StringVal(val)))
	}
	v.Flags |= VariableConstant
	return v
}

var nilVariable = &Variable{
	Name:     "nil",
	Addr:     0,
	Base:     0,
	Kind:     reflect.Ptr,
	Children: []Variable{{Addr: 0, OnlyAddr: true}},
}

func (v *Variable) clone() *Variable {
	r := *v
	return &r
}

// TypeString returns the string representation
// of the type of this variable.
func (v *Variable) TypeString() string {
	if v == nilVariable {
		return "nil"
	}
	if v.DwarfType != nil {
		return v.DwarfType.Common().Name
	}
	return v.Kind.String()
}

func (v *Variable) toField(field *godwarf.StructField) (*Variable, error) {
	if v.Unreadable != nil {
		return v.clone(), nil
	}
	if v.Addr == 0 {
		return nil, &IsNilErr{v.Name}
	}

	name := ""
	if v.Name != "" {
		parts := strings.Split(field.Name, ".")
		if len(parts) > 1 {
			name = fmt.Sprintf("%s.%s", v.Name, parts[1])
		} else {
			name = fmt.Sprintf("%s.%s", v.Name, field.Name)
		}
	}
	return v.newVariable(name, uintptr(int64(v.Addr)+field.ByteOffset), field.Type, v.mem), nil
}

// DwarfReader returns the DwarfReader containing the
// Dwarf information for the target process.
func (scope *EvalScope) DwarfReader() *reader.Reader {
	return scope.BinInfo.DwarfReader()
}

// PtrSize returns the size of a pointer.
func (scope *EvalScope) PtrSize() int {
	return scope.BinInfo.Arch.PtrSize()
}

// ErrNoGoroutine returned when a G could not be found
// for a specific thread.
type ErrNoGoroutine struct {
	tid int
}

func (ng ErrNoGoroutine) Error() string {
	return fmt.Sprintf("no G executing on thread %d", ng.tid)
}

func (v *Variable) parseG() (*G, error) {
	mem := v.mem
	gaddr := uint64(v.Addr)
	_, deref := v.RealType.(*godwarf.PtrType)

	if deref {
		gaddrbytes := make([]byte, v.bi.Arch.PtrSize())
		_, err := mem.ReadMemory(gaddrbytes, uintptr(gaddr))
		if err != nil {
			return nil, fmt.Errorf("error derefing *G %s", err)
		}
		gaddr = binary.LittleEndian.Uint64(gaddrbytes)
	}
	if gaddr == 0 {
		id := 0
		if thread, ok := mem.(Thread); ok {
			id = thread.ThreadID()
		}
		return nil, ErrNoGoroutine{tid: id}
	}
	for {
		if _, isptr := v.RealType.(*godwarf.PtrType); !isptr {
			break
		}
		v = v.maybeDereference()
	}
	v.loadValue(LoadConfig{false, 2, 64, 0, -1, 0})
	if v.Unreadable != nil {
		return nil, v.Unreadable
	}
	schedVar := v.fieldVariable("sched")
	pc, _ := constant.Int64Val(schedVar.fieldVariable("pc").Value)
	sp, _ := constant.Int64Val(schedVar.fieldVariable("sp").Value)
	var bp int64
	if bpvar := schedVar.fieldVariable("bp"); bpvar != nil && bpvar.Value != nil {
		bp, _ = constant.Int64Val(bpvar.Value)
	}
	id, _ := constant.Int64Val(v.fieldVariable("goid").Value)
	gopc, _ := constant.Int64Val(v.fieldVariable("gopc").Value)
	startpc, _ := constant.Int64Val(v.fieldVariable("startpc").Value)
	waitReason := ""
	if wrvar := v.fieldVariable("waitreason"); wrvar.Value != nil {
		switch wrvar.Kind {
		case reflect.String:
			waitReason = constant.StringVal(wrvar.Value)
		case reflect.Uint:
			waitReason = wrvar.ConstDescr()
		}

	}
	var stackhi, stacklo uint64
	if stackVar := v.fieldVariable("stack"); stackVar != nil {
		if stackhiVar := stackVar.fieldVariable("hi"); stackhiVar != nil {
			stackhi, _ = constant.Uint64Val(stackhiVar.Value)
		}
		if stackloVar := stackVar.fieldVariable("lo"); stackloVar != nil {
			stacklo, _ = constant.Uint64Val(stackloVar.Value)
		}
	}

	stkbarVar, _ := v.structMember("stkbar")
	stkbarVarPosFld := v.fieldVariable("stkbarPos")
	var stkbarPos int64
	if stkbarVarPosFld != nil { // stack barriers were removed in Go 1.9
		stkbarPos, _ = constant.Int64Val(stkbarVarPosFld.Value)
	}

	status, _ := constant.Int64Val(v.fieldVariable("atomicstatus").Value)
	f, l, fn := v.bi.PCToLine(uint64(pc))
	g := &G{
		ID:         int(id),
		GoPC:       uint64(gopc),
		StartPC:    uint64(startpc),
		PC:         uint64(pc),
		SP:         uint64(sp),
		BP:         uint64(bp),
		WaitReason: waitReason,
		Status:     uint64(status),
		CurrentLoc: Location{PC: uint64(pc), File: f, Line: l, Fn: fn},
		variable:   v,
		stkbarVar:  stkbarVar,
		stkbarPos:  int(stkbarPos),
		stackhi:    stackhi,
		stacklo:    stacklo,
	}
	return g, nil
}

func (v *Variable) loadFieldNamed(name string) *Variable {
	v, err := v.structMember(name)
	if err != nil {
		return nil
	}
	v.loadValue(loadFullValue)
	if v.Unreadable != nil {
		return nil
	}
	return v
}

func (v *Variable) fieldVariable(name string) *Variable {
	for i := range v.Children {
		if child := &v.Children[i]; child.Name == name {
			return child
		}
	}
	return nil
}

// Defer returns the top-most defer of the goroutine.
func (g *G) Defer() *Defer {
	if g.variable.Unreadable != nil {
		return nil
	}
	dvar := g.variable.fieldVariable("_defer").maybeDereference()
	if dvar.Addr == 0 {
		return nil
	}
	d := &Defer{variable: dvar}
	d.load()
	return d
}

// From $GOROOT/src/runtime/traceback.go:597
// isExportedRuntime reports whether name is an exported runtime function.
// It is only for runtime functions, so ASCII A-Z is fine.
func isExportedRuntime(name string) bool {
	const n = len("runtime.")
	return len(name) > n && name[:n] == "runtime." && 'A' <= name[n] && name[n] <= 'Z'
}

// UserCurrent returns the location the users code is at,
// or was at before entering a runtime function.
func (g *G) UserCurrent() Location {
	it, err := g.stackIterator()
	if err != nil {
		return g.CurrentLoc
	}
	for it.Next() {
		frame := it.Frame()
		if frame.Call.Fn != nil {
			name := frame.Call.Fn.Name
			if strings.Contains(name, ".") && (!strings.HasPrefix(name, "runtime.") || isExportedRuntime(name)) {
				return frame.Call
			}
		}
	}
	return g.CurrentLoc
}

// Go returns the location of the 'go' statement
// that spawned this goroutine.
func (g *G) Go() Location {
	pc := g.GoPC
	if fn := g.variable.bi.PCToFunc(pc); fn != nil {
		// Backup to CALL instruction.
		// Mimics runtime/traceback.go:677.
		if g.GoPC > fn.Entry {
			pc--
		}
	}
	f, l, fn := g.variable.bi.PCToLine(pc)
	return Location{PC: g.GoPC, File: f, Line: l, Fn: fn}
}

// StartLoc returns the starting location of the goroutine.
func (g *G) StartLoc() Location {
	f, l, fn := g.variable.bi.PCToLine(g.StartPC)
	return Location{PC: g.StartPC, File: f, Line: l, Fn: fn}
}

var errTracebackAncestorsDisabled = errors.New("tracebackancestors is disabled")

// Ancestors returns the list of ancestors for g.
func (g *G) Ancestors(n int) ([]Ancestor, error) {
	scope := globalScope(g.Thread.BinInfo(), g.Thread)
	tbav, err := scope.EvalExpression("runtime.debug.tracebackancestors", loadSingleValue)
	if err == nil && tbav.Unreadable == nil && tbav.Kind == reflect.Int {
		tba, _ := constant.Int64Val(tbav.Value)
		if tba == 0 {
			return nil, errTracebackAncestorsDisabled
		}
	}

	av, err := g.variable.structMember("ancestors")
	if err != nil {
		return nil, err
	}
	av = av.maybeDereference()
	av.loadValue(LoadConfig{MaxArrayValues: n, MaxVariableRecurse: 1, MaxStructFields: -1})
	if av.Unreadable != nil {
		return nil, err
	}
	if av.Addr == 0 {
		// no ancestors
		return nil, nil
	}

	r := make([]Ancestor, len(av.Children))

	for i := range av.Children {
		if av.Children[i].Unreadable != nil {
			r[i].Unreadable = av.Children[i].Unreadable
			continue
		}
		goidv := av.Children[i].fieldVariable("goid")
		if goidv.Unreadable != nil {
			r[i].Unreadable = goidv.Unreadable
			continue
		}
		r[i].ID, _ = constant.Int64Val(goidv.Value)
		pcsVar := av.Children[i].fieldVariable("pcs")
		if pcsVar.Unreadable != nil {
			r[i].Unreadable = pcsVar.Unreadable
		}
		pcsVar.loaded = false
		pcsVar.Children = pcsVar.Children[:0]
		r[i].pcsVar = pcsVar
	}

	return r, nil
}

// Stack returns the stack trace of ancestor 'a' as saved by the runtime.
func (a *Ancestor) Stack(n int) ([]Stackframe, error) {
	if a.Unreadable != nil {
		return nil, a.Unreadable
	}
	pcsVar := a.pcsVar.clone()
	pcsVar.loadValue(LoadConfig{MaxArrayValues: n})
	if pcsVar.Unreadable != nil {
		return nil, pcsVar.Unreadable
	}
	r := make([]Stackframe, len(pcsVar.Children))
	for i := range pcsVar.Children {
		if pcsVar.Children[i].Unreadable != nil {
			r[i] = Stackframe{Err: pcsVar.Children[i].Unreadable}
			continue
		}
		if pcsVar.Children[i].Kind != reflect.Uint {
			return nil, fmt.Errorf("wrong type for pcs item %d: %v", i, pcsVar.Children[i].Kind)
		}
		pc, _ := constant.Int64Val(pcsVar.Children[i].Value)
		fn := a.pcsVar.bi.PCToFunc(uint64(pc))
		if fn == nil {
			loc := Location{PC: uint64(pc)}
			r[i] = Stackframe{Current: loc, Call: loc}
			continue
		}
		pc2 := uint64(pc)
		if pc2-1 >= fn.Entry {
			pc2--
		}
		f, ln := fn.cu.lineInfo.PCToLine(fn.Entry, pc2)
		loc := Location{PC: uint64(pc), File: f, Line: ln, Fn: fn}
		r[i] = Stackframe{Current: loc, Call: loc}
	}
	r[len(r)-1].Bottom = pcsVar.Len == int64(len(pcsVar.Children))
	return r, nil
}

// Returns the list of saved return addresses used by stack barriers
func (g *G) stkbar() ([]savedLR, error) {
	if g.stkbarVar == nil { // stack barriers were removed in Go 1.9
		return nil, nil
	}
	g.stkbarVar.loadValue(LoadConfig{false, 1, 0, int(g.stkbarVar.Len), 3, 0})
	if g.stkbarVar.Unreadable != nil {
		return nil, fmt.Errorf("unreadable stkbar: %v", g.stkbarVar.Unreadable)
	}
	r := make([]savedLR, len(g.stkbarVar.Children))
	for i, child := range g.stkbarVar.Children {
		for _, field := range child.Children {
			switch field.Name {
			case "savedLRPtr":
				ptr, _ := constant.Int64Val(field.Value)
				r[i].ptr = uint64(ptr)
			case "savedLRVal":
				val, _ := constant.Int64Val(field.Value)
				r[i].val = uint64(val)
			}
		}
	}
	return r, nil
}

// EvalVariable returns the value of the given expression (backwards compatibility).
func (scope *EvalScope) EvalVariable(name string, cfg LoadConfig) (*Variable, error) {
	return scope.EvalExpression(name, cfg)
}

// SetVariable sets the value of the named variable
func (scope *EvalScope) SetVariable(name, value string) error {
	t, err := parser.ParseExpr(name)
	if err != nil {
		return err
	}

	xv, err := scope.evalAST(t)
	if err != nil {
		return err
	}

	if xv.Addr == 0 {
		return fmt.Errorf("Can not assign to \"%s\"", name)
	}

	if xv.Unreadable != nil {
		return fmt.Errorf("Expression \"%s\" is unreadable: %v", name, xv.Unreadable)
	}

	t, err = parser.ParseExpr(value)
	if err != nil {
		return err
	}

	yv, err := scope.evalAST(t)
	if err != nil {
		return err
	}

	return xv.setValue(yv, value)
}

// LocalVariables returns all local variables from the current function scope.
func (scope *EvalScope) LocalVariables(cfg LoadConfig) ([]*Variable, error) {
	vars, err := scope.Locals()
	if err != nil {
		return nil, err
	}
	vars = filterVariables(vars, func(v *Variable) bool {
		return (v.Flags & (VariableArgument | VariableReturnArgument)) == 0
	})
	cfg.MaxMapBuckets = maxMapBucketsFactor * cfg.MaxArrayValues
	loadValues(vars, cfg)
	return vars, nil
}

// FunctionArguments returns the name, value, and type of all current function arguments.
func (scope *EvalScope) FunctionArguments(cfg LoadConfig) ([]*Variable, error) {
	vars, err := scope.Locals()
	if err != nil {
		return nil, err
	}
	vars = filterVariables(vars, func(v *Variable) bool {
		return (v.Flags & (VariableArgument | VariableReturnArgument)) != 0
	})
	cfg.MaxMapBuckets = maxMapBucketsFactor * cfg.MaxArrayValues
	loadValues(vars, cfg)
	return vars, nil
}

func filterVariables(vars []*Variable, pred func(v *Variable) bool) []*Variable {
	r := make([]*Variable, 0, len(vars))
	for i := range vars {
		if pred(vars[i]) {
			r = append(r, vars[i])
		}
	}
	return r
}

// PackageVariables returns the name, value, and type of all package variables in the application.
func (scope *EvalScope) PackageVariables(cfg LoadConfig) ([]*Variable, error) {
	var vars []*Variable
	reader := scope.DwarfReader()

	var utypoff dwarf.Offset
	utypentry, err := reader.SeekToTypeNamed("<unspecified>")
	if err == nil {
		utypoff = utypentry.Offset
	}

	for entry, err := reader.NextPackageVariable(); entry != nil; entry, err = reader.NextPackageVariable() {
		if err != nil {
			return nil, err
		}

		if typoff, ok := entry.Val(dwarf.AttrType).(dwarf.Offset); !ok || typoff == utypoff {
			continue
		}

		// Ignore errors trying to extract values
		val, err := scope.extractVarInfoFromEntry(entry)
		if err != nil {
			continue
		}
		val.loadValue(cfg)
		vars = append(vars, val)
	}

	return vars, nil
}

func (scope *EvalScope) findGlobal(name string) (*Variable, error) {
	for _, pkgvar := range scope.BinInfo.packageVars {
		if pkgvar.name == name || strings.HasSuffix(pkgvar.name, "/"+name) {
			reader := scope.DwarfReader()
			reader.Seek(pkgvar.offset)
			entry, err := reader.Next()
			if err != nil {
				return nil, err
			}
			return scope.extractVarInfoFromEntry(entry)
		}
	}
	for _, fn := range scope.BinInfo.Functions {
		if fn.Name == name || strings.HasSuffix(fn.Name, "/"+name) {
			//TODO(aarzilli): convert function entry into a function type?
			r := scope.newVariable(fn.Name, uintptr(fn.Entry), &godwarf.FuncType{}, scope.Mem)
			r.Value = constant.MakeString(fn.Name)
			r.Base = uintptr(fn.Entry)
			r.loaded = true
			return r, nil
		}
	}
	for offset, ctyp := range scope.BinInfo.consts {
		for _, cval := range ctyp.values {
			if cval.fullName == name || strings.HasSuffix(cval.fullName, "/"+name) {
				t, err := scope.BinInfo.Type(offset)
				if err != nil {
					return nil, err
				}
				v := scope.newVariable(name, 0x0, t, scope.Mem)
				switch v.Kind {
				case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
					v.Value = constant.MakeInt64(cval.value)
				case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
					v.Value = constant.MakeUint64(uint64(cval.value))
				default:
					return nil, fmt.Errorf("unsupported constant kind %v", v.Kind)
				}
				v.Flags |= VariableConstant
				v.loaded = true
				return v, nil
			}
		}
	}
	return nil, fmt.Errorf("could not find symbol value for %s", name)
}

func (v *Variable) structMember(memberName string) (*Variable, error) {
	if v.Unreadable != nil {
		return v.clone(), nil
	}
	vname := v.Name
	switch v.Kind {
	case reflect.Chan:
		v = v.clone()
		v.RealType = resolveTypedef(&(v.RealType.(*godwarf.ChanType).TypedefType))
	case reflect.Interface:
		v.loadInterface(0, false, LoadConfig{})
		if len(v.Children) > 0 {
			v = &v.Children[0]
		}
	}
	structVar := v.maybeDereference()
	structVar.Name = v.Name
	if structVar.Unreadable != nil {
		return structVar, nil
	}

	switch t := structVar.RealType.(type) {
	case *godwarf.StructType:
		for _, field := range t.Field {
			if field.Name != memberName {
				continue
			}
			return structVar.toField(field)
		}
		// Check for embedded field only if field was
		// not a regular struct member
		for _, field := range t.Field {
			isEmbeddedStructMember :=
				field.Embedded ||
					(field.Type.Common().Name == field.Name) ||
					(len(field.Name) > 1 &&
						field.Name[0] == '*' &&
						field.Type.Common().Name[1:] == field.Name[1:])
			if !isEmbeddedStructMember {
				continue
			}
			// Check for embedded field referenced by type name
			parts := strings.Split(field.Name, ".")
			if len(parts) > 1 && parts[1] == memberName {
				embeddedVar, err := structVar.toField(field)
				if err != nil {
					return nil, err
				}
				return embeddedVar, nil
			}
			// Recursively check for promoted fields on the embedded field
			embeddedVar, err := structVar.toField(field)
			if err != nil {
				return nil, err
			}
			embeddedVar.Name = structVar.Name
			embeddedField, _ := embeddedVar.structMember(memberName)
			if embeddedField != nil {
				return embeddedField, nil
			}
		}
		return nil, fmt.Errorf("%s has no member %s", vname, memberName)
	default:
		if v.Name == "" {
			return nil, fmt.Errorf("type %s is not a struct", structVar.TypeString())
		}
		return nil, fmt.Errorf("%s (type %s) is not a struct", vname, structVar.TypeString())
	}
}

func readVarEntry(varEntry *dwarf.Entry, bi *BinaryInfo) (entry reader.Entry, name string, typ godwarf.Type, err error) {
	entry, _ = reader.LoadAbstractOrigin(varEntry, bi.dwarfReader)

	name, ok := entry.Val(dwarf.AttrName).(string)
	if !ok {
		return nil, "", nil, fmt.Errorf("malformed variable DIE (name)")
	}

	offset, ok := entry.Val(dwarf.AttrType).(dwarf.Offset)
	if !ok {
		return nil, "", nil, fmt.Errorf("malformed variable DIE (offset)")
	}

	typ, err = bi.Type(offset)
	if err != nil {
		return nil, "", nil, err
	}

	return entry, name, typ, nil
}

// Extracts the name and type of a variable from a dwarf entry
// then executes the instructions given in the  DW_AT_location attribute to grab the variable's address
func (scope *EvalScope) extractVarInfoFromEntry(varEntry *dwarf.Entry) (*Variable, error) {
	if varEntry == nil {
		return nil, fmt.Errorf("invalid entry")
	}

	if varEntry.Tag != dwarf.TagFormalParameter && varEntry.Tag != dwarf.TagVariable {
		return nil, fmt.Errorf("invalid entry tag, only supports FormalParameter and Variable, got %s", varEntry.Tag.String())
	}

	entry, n, t, err := readVarEntry(varEntry, scope.BinInfo)
	if err != nil {
		return nil, err
	}

	addr, pieces, descr, err := scope.BinInfo.Location(entry, dwarf.AttrLocation, scope.PC, scope.Regs)
	mem := scope.Mem
	if pieces != nil {
		addr = fakeAddress
		var cmem *compositeMemory
		cmem, err = newCompositeMemory(scope.Mem, scope.Regs, pieces)
		if cmem != nil {
			mem = cmem
		}
	}

	v := scope.newVariable(n, uintptr(addr), t, mem)
	v.LocationExpr = descr
	v.DeclLine, _ = entry.Val(dwarf.AttrDeclLine).(int64)
	if err != nil {
		v.Unreadable = err
	}
	return v, nil
}

// If v is a pointer a new variable is returned containing the value pointed by v.
func (v *Variable) maybeDereference() *Variable {
	if v.Unreadable != nil {
		return v
	}

	switch t := v.RealType.(type) {
	case *godwarf.PtrType:
		if v.Addr == 0 && len(v.Children) == 1 && v.loaded {
			// fake pointer variable constructed by casting an integer to a pointer type
			return &v.Children[0]
		}
		ptrval, err := readUintRaw(v.mem, uintptr(v.Addr), t.ByteSize)
		r := v.newVariable("", uintptr(ptrval), t.Type, DereferenceMemory(v.mem))
		if err != nil {
			r.Unreadable = err
		}

		return r
	default:
		return v
	}
}

func loadValues(vars []*Variable, cfg LoadConfig) {
	for i := range vars {
		vars[i].loadValueInternal(0, cfg)
	}
}

// Extracts the value of the variable at the given address.
func (v *Variable) loadValue(cfg LoadConfig) {
	v.loadValueInternal(0, cfg)
}

func (v *Variable) loadValueInternal(recurseLevel int, cfg LoadConfig) {
	if v.Unreadable != nil || v.loaded || (v.Addr == 0 && v.Base == 0) {
		return
	}

	v.loaded = true
	switch v.Kind {
	case reflect.Ptr, reflect.UnsafePointer:
		v.Len = 1
		v.Children = []Variable{*v.maybeDereference()}
		if cfg.FollowPointers {
			// Don't increase the recursion level when dereferencing pointers
			// unless this is a pointer to interface (which could cause an infinite loop)
			nextLvl := recurseLevel
			if v.Children[0].Kind == reflect.Interface {
				nextLvl++
			}
			v.Children[0].loadValueInternal(nextLvl, cfg)
		} else {
			v.Children[0].OnlyAddr = true
		}

	case reflect.Chan:
		sv := v.clone()
		sv.RealType = resolveTypedef(&(sv.RealType.(*godwarf.ChanType).TypedefType))
		sv = sv.maybeDereference()
		sv.loadValueInternal(0, loadFullValue)
		v.Children = sv.Children
		v.Len = sv.Len
		v.Base = sv.Addr

	case reflect.Map:
		if recurseLevel <= cfg.MaxVariableRecurse {
			v.loadMap(recurseLevel, cfg)
		} else {
			// loads length so that the client knows that the map isn't empty
			v.mapIterator()
		}

	case reflect.String:
		var val string
		val, v.Unreadable = readStringValue(DereferenceMemory(v.mem), v.Base, v.Len, cfg)
		v.Value = constant.MakeString(val)

	case reflect.Slice, reflect.Array:
		v.loadArrayValues(recurseLevel, cfg)

	case reflect.Struct:
		v.mem = cacheMemory(v.mem, v.Addr, int(v.RealType.Size()))
		t := v.RealType.(*godwarf.StructType)
		v.Len = int64(len(t.Field))
		// Recursively call extractValue to grab
		// the value of all the members of the struct.
		if recurseLevel <= cfg.MaxVariableRecurse {
			v.Children = make([]Variable, 0, len(t.Field))
			for i, field := range t.Field {
				if cfg.MaxStructFields >= 0 && len(v.Children) >= cfg.MaxStructFields {
					break
				}
				f, _ := v.toField(field)
				v.Children = append(v.Children, *f)
				v.Children[i].Name = field.Name
				v.Children[i].loadValueInternal(recurseLevel+1, cfg)
			}
		}

	case reflect.Interface:
		v.loadInterface(recurseLevel, true, cfg)

	case reflect.Complex64, reflect.Complex128:
		v.readComplex(v.RealType.(*godwarf.ComplexType).ByteSize)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var val int64
		val, v.Unreadable = readIntRaw(v.mem, v.Addr, v.RealType.(*godwarf.IntType).ByteSize)
		v.Value = constant.MakeInt64(val)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		var val uint64
		val, v.Unreadable = readUintRaw(v.mem, v.Addr, v.RealType.(*godwarf.UintType).ByteSize)
		v.Value = constant.MakeUint64(val)

	case reflect.Bool:
		val := make([]byte, 1)
		_, err := v.mem.ReadMemory(val, v.Addr)
		v.Unreadable = err
		if err == nil {
			v.Value = constant.MakeBool(val[0] != 0)
		}
	case reflect.Float32, reflect.Float64:
		var val float64
		val, v.Unreadable = v.readFloatRaw(v.RealType.(*godwarf.FloatType).ByteSize)
		v.Value = constant.MakeFloat64(val)
		switch {
		case math.IsInf(val, +1):
			v.FloatSpecial = FloatIsPosInf
		case math.IsInf(val, -1):
			v.FloatSpecial = FloatIsNegInf
		case math.IsNaN(val):
			v.FloatSpecial = FloatIsNaN
		}
	case reflect.Func:
		v.readFunctionPtr()
	default:
		v.Unreadable = fmt.Errorf("unknown or unsupported kind: \"%s\"", v.Kind.String())
	}
}

// setValue writes the value of srcv to dstv.
// * If srcv is a numerical literal constant and srcv is of a compatible type
//   the necessary type conversion is performed.
// * If srcv is nil and dstv is of a nil'able type then dstv is nilled.
// * If srcv is the empty string and dstv is a string then dstv is set to the
//   empty string.
// * If dstv is an "interface {}" and srcv is either an interface (possibly
//   non-empty) or a pointer shaped type (map, channel, pointer or struct
//   containing a single pointer field) the type conversion to "interface {}"
//   is performed.
// * If srcv and dstv have the same type and are both addressable then the
//   contents of srcv are copied byte-by-byte into dstv
func (v *Variable) setValue(srcv *Variable, srcExpr string) error {
	srcv.loadValue(loadSingleValue)

	typerr := srcv.isType(v.RealType, v.Kind)
	if _, isTypeConvErr := typerr.(*typeConvErr); isTypeConvErr {
		// attempt iface -> eface and ptr-shaped -> eface conversions.
		return convertToEface(srcv, v)
	}
	if typerr != nil {
		return typerr
	}

	if srcv.Unreadable != nil {
		return fmt.Errorf("Expression \"%s\" is unreadable: %v", srcExpr, srcv.Unreadable)
	}

	// Numerical types
	switch v.Kind {
	case reflect.Float32, reflect.Float64:
		f, _ := constant.Float64Val(srcv.Value)
		return v.writeFloatRaw(f, v.RealType.Size())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, _ := constant.Int64Val(srcv.Value)
		return v.writeUint(uint64(n), v.RealType.Size())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, _ := constant.Uint64Val(srcv.Value)
		return v.writeUint(n, v.RealType.Size())
	case reflect.Bool:
		return v.writeBool(constant.BoolVal(srcv.Value))
	case reflect.Complex64, reflect.Complex128:
		real, _ := constant.Float64Val(constant.Real(srcv.Value))
		imag, _ := constant.Float64Val(constant.Imag(srcv.Value))
		return v.writeComplex(real, imag, v.RealType.Size())
	}

	// nilling nillable variables
	if srcv == nilVariable {
		return v.writeZero()
	}

	// set a string to ""
	if srcv.Kind == reflect.String && srcv.Len == 0 {
		return v.writeZero()
	}

	// slice assignment (this is not handled by the writeCopy below so that
	// results of a reslice operation can be used here).
	if srcv.Kind == reflect.Slice {
		return v.writeSlice(srcv.Len, srcv.Cap, srcv.Base)
	}

	// allow any integer to be converted to any pointer
	if t, isptr := v.RealType.(*godwarf.PtrType); isptr {
		return v.writeUint(uint64(srcv.Children[0].Addr), int64(t.ByteSize))
	}

	// byte-by-byte copying for everything else, but the source must be addressable
	if srcv.Addr != 0 {
		return v.writeCopy(srcv)
	}

	return fmt.Errorf("can not set variables of type %s (not implemented)", v.Kind.String())
}

// convertToEface converts srcv into an "interface {}" and writes it to
// dstv.
// Dstv must be a variable of type "inteface {}" and srcv must either be an
// interface or a pointer shaped variable (map, channel, pointer or struct
// containing a single pointer)
func convertToEface(srcv, dstv *Variable) error {
	if dstv.RealType.String() != "interface {}" {
		return &typeConvErr{srcv.DwarfType, dstv.RealType}
	}
	if _, isiface := srcv.RealType.(*godwarf.InterfaceType); isiface {
		// iface -> eface conversion
		_type, data, _ := srcv.readInterface()
		if srcv.Unreadable != nil {
			return srcv.Unreadable
		}
		_type = _type.maybeDereference()
		dstv.writeEmptyInterface(uint64(_type.Addr), data)
		return nil
	}
	typeAddr, typeKind, runtimeTypeFound, err := dwarfToRuntimeType(srcv.bi, srcv.mem, srcv.RealType)
	if err != nil {
		return err
	}
	if !runtimeTypeFound || typeKind&kindDirectIface == 0 {
		return &typeConvErr{srcv.DwarfType, dstv.RealType}
	}
	return dstv.writeEmptyInterface(typeAddr, srcv)
}

func readStringInfo(mem MemoryReadWriter, arch Arch, addr uintptr) (uintptr, int64, error) {
	// string data structure is always two ptrs in size. Addr, followed by len
	// http://research.swtch.com/godata

	mem = cacheMemory(mem, addr, arch.PtrSize()*2)

	// read len
	val := make([]byte, arch.PtrSize())
	_, err := mem.ReadMemory(val, addr+uintptr(arch.PtrSize()))
	if err != nil {
		return 0, 0, fmt.Errorf("could not read string len %s", err)
	}
	strlen := int64(binary.LittleEndian.Uint64(val))
	if strlen < 0 {
		return 0, 0, fmt.Errorf("invalid length: %d", strlen)
	}

	// read addr
	_, err = mem.ReadMemory(val, addr)
	if err != nil {
		return 0, 0, fmt.Errorf("could not read string pointer %s", err)
	}
	addr = uintptr(binary.LittleEndian.Uint64(val))
	if addr == 0 {
		return 0, 0, nil
	}

	return addr, strlen, nil
}

func readStringValue(mem MemoryReadWriter, addr uintptr, strlen int64, cfg LoadConfig) (string, error) {
	if strlen == 0 {
		return "", nil
	}

	count := strlen
	if count > int64(cfg.MaxStringLen) {
		count = int64(cfg.MaxStringLen)
	}

	val := make([]byte, int(count))
	_, err := mem.ReadMemory(val, addr)
	if err != nil {
		return "", fmt.Errorf("could not read string at %#v due to %s", addr, err)
	}

	retstr := *(*string)(unsafe.Pointer(&val))

	return retstr, nil
}

const (
	sliceArrayFieldName = "array"
	sliceLenFieldName   = "len"
	sliceCapFieldName   = "cap"
)

func (v *Variable) loadSliceInfo(t *godwarf.SliceType) {
	v.mem = cacheMemory(v.mem, v.Addr, int(t.Size()))

	var err error
	for _, f := range t.Field {
		switch f.Name {
		case sliceArrayFieldName:
			var base uint64
			base, err = readUintRaw(v.mem, uintptr(int64(v.Addr)+f.ByteOffset), f.Type.Size())
			if err == nil {
				v.Base = uintptr(base)
				// Dereference array type to get value type
				ptrType, ok := f.Type.(*godwarf.PtrType)
				if !ok {
					v.Unreadable = fmt.Errorf("Invalid type %s in slice array", f.Type)
					return
				}
				v.fieldType = ptrType.Type
			}
		case sliceLenFieldName:
			lstrAddr, _ := v.toField(f)
			lstrAddr.loadValue(loadSingleValue)
			err = lstrAddr.Unreadable
			if err == nil {
				v.Len, _ = constant.Int64Val(lstrAddr.Value)
			}
		case sliceCapFieldName:
			cstrAddr, _ := v.toField(f)
			cstrAddr.loadValue(loadSingleValue)
			err = cstrAddr.Unreadable
			if err == nil {
				v.Cap, _ = constant.Int64Val(cstrAddr.Value)
			}
		}
		if err != nil {
			v.Unreadable = err
			return
		}
	}

	v.stride = v.fieldType.Size()
	if t, ok := v.fieldType.(*godwarf.PtrType); ok {
		v.stride = t.ByteSize
	}
}

// loadChanInfo loads the buffer size of the channel and changes the type of
// the buf field from unsafe.Pointer to an array of the correct type.
func (v *Variable) loadChanInfo() {
	chanType, ok := v.RealType.(*godwarf.ChanType)
	if !ok {
		v.Unreadable = errors.New("bad channel type")
		return
	}
	sv := v.clone()
	sv.RealType = resolveTypedef(&(chanType.TypedefType))
	sv = sv.maybeDereference()
	if sv.Unreadable != nil || sv.Addr == 0 {
		return
	}
	v.Base = sv.Addr
	structType, ok := sv.DwarfType.(*godwarf.StructType)
	if !ok {
		v.Unreadable = errors.New("bad channel type")
		return
	}

	lenAddr, _ := sv.toField(structType.Field[1])
	lenAddr.loadValue(loadSingleValue)
	if lenAddr.Unreadable != nil {
		v.Unreadable = fmt.Errorf("unreadable length: %v", lenAddr.Unreadable)
		return
	}
	chanLen, _ := constant.Uint64Val(lenAddr.Value)

	newStructType := &godwarf.StructType{}
	*newStructType = *structType
	newStructType.Field = make([]*godwarf.StructField, len(structType.Field))

	for i := range structType.Field {
		field := &godwarf.StructField{}
		*field = *structType.Field[i]
		if field.Name == "buf" {
			stride := chanType.ElemType.Common().ByteSize
			atyp := &godwarf.ArrayType{
				CommonType: godwarf.CommonType{
					ReflectKind: reflect.Array,
					ByteSize:    int64(chanLen) * stride,
					Name:        fmt.Sprintf("[%d]%s", chanLen, chanType.ElemType.String())},
				Type:          chanType.ElemType,
				StrideBitSize: stride * 8,
				Count:         int64(chanLen)}

			field.Type = pointerTo(atyp, v.bi.Arch)
		}
		newStructType.Field[i] = field
	}

	v.RealType = &godwarf.ChanType{
		TypedefType: godwarf.TypedefType{
			CommonType: chanType.TypedefType.CommonType,
			Type:       pointerTo(newStructType, v.bi.Arch),
		},
		ElemType: chanType.ElemType,
	}
}

func (v *Variable) loadArrayValues(recurseLevel int, cfg LoadConfig) {
	if v.Unreadable != nil {
		return
	}
	if v.Len < 0 {
		v.Unreadable = errors.New("Negative array length")
		return
	}

	count := v.Len
	// Cap number of elements
	if count > int64(cfg.MaxArrayValues) {
		count = int64(cfg.MaxArrayValues)
	}

	if v.stride < maxArrayStridePrefetch {
		v.mem = cacheMemory(v.mem, v.Base, int(v.stride*count))
	}

	errcount := 0

	mem := v.mem
	if v.Kind != reflect.Array {
		mem = DereferenceMemory(mem)
	}

	for i := int64(0); i < count; i++ {
		fieldvar := v.newVariable("", uintptr(int64(v.Base)+(i*v.stride)), v.fieldType, mem)
		fieldvar.loadValueInternal(recurseLevel+1, cfg)

		if fieldvar.Unreadable != nil {
			errcount++
		}

		v.Children = append(v.Children, *fieldvar)
		if errcount > maxErrCount {
			break
		}
	}
}

func (v *Variable) readComplex(size int64) {
	var fs int64
	switch size {
	case 8:
		fs = 4
	case 16:
		fs = 8
	default:
		v.Unreadable = fmt.Errorf("invalid size (%d) for complex type", size)
		return
	}

	ftyp := &godwarf.FloatType{BasicType: godwarf.BasicType{CommonType: godwarf.CommonType{ByteSize: fs, Name: fmt.Sprintf("float%d", fs)}, BitSize: fs * 8, BitOffset: 0}}

	realvar := v.newVariable("real", v.Addr, ftyp, v.mem)
	imagvar := v.newVariable("imaginary", v.Addr+uintptr(fs), ftyp, v.mem)
	realvar.loadValue(loadSingleValue)
	imagvar.loadValue(loadSingleValue)
	v.Value = constant.BinaryOp(realvar.Value, token.ADD, constant.MakeImag(imagvar.Value))
}

func (v *Variable) writeComplex(real, imag float64, size int64) error {
	err := v.writeFloatRaw(real, int64(size/2))
	if err != nil {
		return err
	}
	imagaddr := *v
	imagaddr.Addr += uintptr(size / 2)
	return imagaddr.writeFloatRaw(imag, int64(size/2))
}

func readIntRaw(mem MemoryReadWriter, addr uintptr, size int64) (int64, error) {
	var n int64

	val := make([]byte, int(size))
	_, err := mem.ReadMemory(val, addr)
	if err != nil {
		return 0, err
	}

	switch size {
	case 1:
		n = int64(int8(val[0]))
	case 2:
		n = int64(int16(binary.LittleEndian.Uint16(val)))
	case 4:
		n = int64(int32(binary.LittleEndian.Uint32(val)))
	case 8:
		n = int64(binary.LittleEndian.Uint64(val))
	}

	return n, nil
}

func (v *Variable) writeUint(value uint64, size int64) error {
	val := make([]byte, size)

	switch size {
	case 1:
		val[0] = byte(value)
	case 2:
		binary.LittleEndian.PutUint16(val, uint16(value))
	case 4:
		binary.LittleEndian.PutUint32(val, uint32(value))
	case 8:
		binary.LittleEndian.PutUint64(val, uint64(value))
	}

	_, err := v.mem.WriteMemory(v.Addr, val)
	return err
}

func readUintRaw(mem MemoryReadWriter, addr uintptr, size int64) (uint64, error) {
	var n uint64

	val := make([]byte, int(size))
	_, err := mem.ReadMemory(val, addr)
	if err != nil {
		return 0, err
	}

	switch size {
	case 1:
		n = uint64(val[0])
	case 2:
		n = uint64(binary.LittleEndian.Uint16(val))
	case 4:
		n = uint64(binary.LittleEndian.Uint32(val))
	case 8:
		n = uint64(binary.LittleEndian.Uint64(val))
	}

	return n, nil
}

func (v *Variable) readFloatRaw(size int64) (float64, error) {
	val := make([]byte, int(size))
	_, err := v.mem.ReadMemory(val, v.Addr)
	if err != nil {
		return 0.0, err
	}
	buf := bytes.NewBuffer(val)

	switch size {
	case 4:
		n := float32(0)
		binary.Read(buf, binary.LittleEndian, &n)
		return float64(n), nil
	case 8:
		n := float64(0)
		binary.Read(buf, binary.LittleEndian, &n)
		return n, nil
	}

	return 0.0, fmt.Errorf("could not read float")
}

func (v *Variable) writeFloatRaw(f float64, size int64) error {
	buf := bytes.NewBuffer(make([]byte, 0, size))

	switch size {
	case 4:
		n := float32(f)
		binary.Write(buf, binary.LittleEndian, n)
	case 8:
		n := float64(f)
		binary.Write(buf, binary.LittleEndian, n)
	}

	_, err := v.mem.WriteMemory(v.Addr, buf.Bytes())
	return err
}

func (v *Variable) writeBool(value bool) error {
	val := []byte{0}
	val[0] = *(*byte)(unsafe.Pointer(&value))
	_, err := v.mem.WriteMemory(v.Addr, val)
	return err
}

func (v *Variable) writeZero() error {
	val := make([]byte, v.RealType.Size())
	_, err := v.mem.WriteMemory(v.Addr, val)
	return err
}

// writeInterface writes the empty interface of type typeAddr and data as the data field.
func (v *Variable) writeEmptyInterface(typeAddr uint64, data *Variable) error {
	dstType, dstData, _ := v.readInterface()
	if v.Unreadable != nil {
		return v.Unreadable
	}
	dstType.writeUint(typeAddr, dstType.RealType.Size())
	dstData.writeCopy(data)
	return nil
}

func (v *Variable) writeSlice(len, cap int64, base uintptr) error {
	for _, f := range v.RealType.(*godwarf.SliceType).Field {
		switch f.Name {
		case sliceArrayFieldName:
			arrv, _ := v.toField(f)
			if err := arrv.writeUint(uint64(base), arrv.RealType.Size()); err != nil {
				return err
			}
		case sliceLenFieldName:
			lenv, _ := v.toField(f)
			if err := lenv.writeUint(uint64(len), lenv.RealType.Size()); err != nil {
				return err
			}
		case sliceCapFieldName:
			capv, _ := v.toField(f)
			if err := capv.writeUint(uint64(cap), capv.RealType.Size()); err != nil {
				return err
			}
		}
	}
	return nil
}

func (v *Variable) writeCopy(srcv *Variable) error {
	buf := make([]byte, srcv.RealType.Size())
	_, err := srcv.mem.ReadMemory(buf, srcv.Addr)
	if err != nil {
		return err
	}
	_, err = v.mem.WriteMemory(v.Addr, buf)
	return err
}

func (v *Variable) readFunctionPtr() {
	// dereference pointer to find function pc
	fnaddr := v.funcvalAddr()
	if v.Unreadable != nil {
		return
	}
	if fnaddr == 0 {
		v.Base = 0
		v.Value = constant.MakeString("")
		return
	}

	val := make([]byte, v.bi.Arch.PtrSize())
	_, err := v.mem.ReadMemory(val, uintptr(fnaddr))
	if err != nil {
		v.Unreadable = err
		return
	}

	v.Base = uintptr(binary.LittleEndian.Uint64(val))
	fn := v.bi.PCToFunc(uint64(v.Base))
	if fn == nil {
		v.Unreadable = fmt.Errorf("could not find function for %#v", v.Base)
		return
	}

	v.Value = constant.MakeString(fn.Name)
}

// funcvalAddr reads the address of the funcval contained in a function variable.
func (v *Variable) funcvalAddr() uint64 {
	val := make([]byte, v.bi.Arch.PtrSize())
	_, err := v.mem.ReadMemory(val, v.Addr)
	if err != nil {
		v.Unreadable = err
		return 0
	}
	return binary.LittleEndian.Uint64(val)
}

func (v *Variable) loadMap(recurseLevel int, cfg LoadConfig) {
	it := v.mapIterator()
	if it == nil {
		return
	}
	it.maxNumBuckets = uint64(cfg.MaxMapBuckets)

	if v.Len == 0 || int64(v.mapSkip) >= v.Len || cfg.MaxArrayValues == 0 {
		return
	}

	for skip := 0; skip < v.mapSkip; skip++ {
		if ok := it.next(); !ok {
			v.Unreadable = fmt.Errorf("map index out of bounds")
			return
		}
	}

	count := 0
	errcount := 0
	for it.next() {
		key := it.key()
		var val *Variable
		if it.values.fieldType.Size() > 0 {
			val = it.value()
		} else {
			val = v.newVariable("", it.values.Addr, it.values.fieldType, DereferenceMemory(v.mem))
		}
		key.loadValueInternal(recurseLevel+1, cfg)
		val.loadValueInternal(recurseLevel+1, cfg)
		if key.Unreadable != nil || val.Unreadable != nil {
			errcount++
		}
		v.Children = append(v.Children, *key)
		v.Children = append(v.Children, *val)
		count++
		if errcount > maxErrCount {
			break
		}
		if count >= cfg.MaxArrayValues || int64(count) >= v.Len {
			break
		}
	}
}

type mapIterator struct {
	v          *Variable
	numbuckets uint64
	oldmask    uint64
	buckets    *Variable
	oldbuckets *Variable
	b          *Variable
	bidx       uint64

	tophashes *Variable
	keys      *Variable
	values    *Variable
	overflow  *Variable

	maxNumBuckets uint64 // maximum number of buckets to scan

	idx int64
}

// Code derived from go/src/runtime/hashmap.go
func (v *Variable) mapIterator() *mapIterator {
	sv := v.clone()
	sv.RealType = resolveTypedef(&(sv.RealType.(*godwarf.MapType).TypedefType))
	sv = sv.maybeDereference()
	v.Base = sv.Addr

	maptype, ok := sv.RealType.(*godwarf.StructType)
	if !ok {
		v.Unreadable = fmt.Errorf("wrong real type for map")
		return nil
	}

	it := &mapIterator{v: v, bidx: 0, b: nil, idx: 0}

	if sv.Addr == 0 {
		it.numbuckets = 0
		return it
	}

	v.mem = cacheMemory(v.mem, v.Base, int(v.RealType.Size()))

	for _, f := range maptype.Field {
		var err error
		field, _ := sv.toField(f)
		switch f.Name {
		case "count":
			v.Len, err = field.asInt()
		case "B":
			var b uint64
			b, err = field.asUint()
			it.numbuckets = 1 << b
			it.oldmask = (1 << (b - 1)) - 1
		case "buckets":
			it.buckets = field.maybeDereference()
		case "oldbuckets":
			it.oldbuckets = field.maybeDereference()
		}
		if err != nil {
			v.Unreadable = err
			return nil
		}
	}

	if it.buckets.Kind != reflect.Struct || it.oldbuckets.Kind != reflect.Struct {
		v.Unreadable = errMapBucketsNotStruct
		return nil
	}

	return it
}

var errMapBucketContentsNotArray = errors.New("malformed map type: keys, values or tophash of a bucket is not an array")
var errMapBucketContentsInconsistentLen = errors.New("malformed map type: inconsistent array length in bucket")
var errMapBucketsNotStruct = errors.New("malformed map type: buckets, oldbuckets or overflow field not a struct")

func (it *mapIterator) nextBucket() bool {
	if it.overflow != nil && it.overflow.Addr > 0 {
		it.b = it.overflow
	} else {
		it.b = nil

		if it.maxNumBuckets > 0 && it.bidx >= it.maxNumBuckets {
			return false
		}

		for it.bidx < it.numbuckets {
			it.b = it.buckets.clone()
			it.b.Addr += uintptr(uint64(it.buckets.DwarfType.Size()) * it.bidx)

			if it.oldbuckets.Addr <= 0 {
				break
			}

			// if oldbuckets is not nil we are iterating through a map that is in
			// the middle of a grow.
			// if the bucket we are looking at hasn't been filled in we iterate
			// instead through its corresponding "oldbucket" (i.e. the bucket the
			// elements of this bucket are coming from) but only if this is the first
			// of the two buckets being created from the same oldbucket (otherwise we
			// would print some keys twice)

			oldbidx := it.bidx & it.oldmask
			oldb := it.oldbuckets.clone()
			oldb.Addr += uintptr(uint64(it.oldbuckets.DwarfType.Size()) * oldbidx)

			if mapEvacuated(oldb) {
				break
			}

			if oldbidx == it.bidx {
				it.b = oldb
				break
			}

			// oldbucket origin for current bucket has not been evacuated but we have already
			// iterated over it so we should just skip it
			it.b = nil
			it.bidx++
		}

		if it.b == nil {
			return false
		}
		it.bidx++
	}

	if it.b.Addr <= 0 {
		return false
	}

	it.b.mem = cacheMemory(it.b.mem, it.b.Addr, int(it.b.RealType.Size()))

	it.tophashes = nil
	it.keys = nil
	it.values = nil
	it.overflow = nil

	for _, f := range it.b.DwarfType.(*godwarf.StructType).Field {
		field, err := it.b.toField(f)
		if err != nil {
			it.v.Unreadable = err
			return false
		}
		if field.Unreadable != nil {
			it.v.Unreadable = field.Unreadable
			return false
		}

		switch f.Name {
		case "tophash":
			it.tophashes = field
		case "keys":
			it.keys = field
		case "values":
			it.values = field
		case "overflow":
			it.overflow = field.maybeDereference()
		}
	}

	// sanity checks
	if it.tophashes == nil || it.keys == nil || it.values == nil {
		it.v.Unreadable = fmt.Errorf("malformed map type")
		return false
	}

	if it.tophashes.Kind != reflect.Array || it.keys.Kind != reflect.Array || it.values.Kind != reflect.Array {
		it.v.Unreadable = errMapBucketContentsNotArray
		return false
	}

	if it.tophashes.Len != it.keys.Len {
		it.v.Unreadable = errMapBucketContentsInconsistentLen
		return false
	}

	if it.values.fieldType.Size() > 0 && it.tophashes.Len != it.values.Len {
		// if the type of the value is zero-sized (i.e. struct{}) then the values
		// array's length is zero.
		it.v.Unreadable = errMapBucketContentsInconsistentLen
		return false
	}

	if it.overflow.Kind != reflect.Struct {
		it.v.Unreadable = errMapBucketsNotStruct
		return false
	}

	return true
}

func (it *mapIterator) next() bool {
	for {
		if it.b == nil || it.idx >= it.tophashes.Len {
			r := it.nextBucket()
			if !r {
				return false
			}
			it.idx = 0
		}
		tophash, _ := it.tophashes.sliceAccess(int(it.idx))
		h, err := tophash.asUint()
		if err != nil {
			it.v.Unreadable = fmt.Errorf("unreadable tophash: %v", err)
			return false
		}
		it.idx++
		if h != hashTophashEmpty {
			return true
		}
	}
}

func (it *mapIterator) key() *Variable {
	k, _ := it.keys.sliceAccess(int(it.idx - 1))
	return k
}

func (it *mapIterator) value() *Variable {
	v, _ := it.values.sliceAccess(int(it.idx - 1))
	return v
}

func mapEvacuated(b *Variable) bool {
	if b.Addr == 0 {
		return true
	}
	for _, f := range b.DwarfType.(*godwarf.StructType).Field {
		if f.Name != "tophash" {
			continue
		}
		tophashes, _ := b.toField(f)
		tophash0var, _ := tophashes.sliceAccess(0)
		tophash0, err := tophash0var.asUint()
		if err != nil {
			return true
		}
		return tophash0 > hashTophashEmpty && tophash0 < hashMinTopHash
	}
	return true
}

func (v *Variable) readInterface() (_type, data *Variable, isnil bool) {
	// An interface variable is implemented either by a runtime.iface
	// struct or a runtime.eface struct. The difference being that empty
	// interfaces (i.e. "interface {}") are represented by runtime.eface
	// and non-empty interfaces by runtime.iface.
	//
	// For both runtime.ifaces and runtime.efaces the data is stored in v.data
	//
	// The concrete type however is stored in v.tab._type for non-empty
	// interfaces and in v._type for empty interfaces.
	//
	// For nil empty interface variables _type will be nil, for nil
	// non-empty interface variables tab will be nil
	//
	// In either case the _type field is a pointer to a runtime._type struct.
	//
	// The following code works for both runtime.iface and runtime.eface.

	v.mem = cacheMemory(v.mem, v.Addr, int(v.RealType.Size()))

	ityp := resolveTypedef(&v.RealType.(*godwarf.InterfaceType).TypedefType).(*godwarf.StructType)

	for _, f := range ityp.Field {
		switch f.Name {
		case "tab": // for runtime.iface
			tab, _ := v.toField(f)
			tab = tab.maybeDereference()
			isnil = tab.Addr == 0
			if !isnil {
				var err error
				_type, err = tab.structMember("_type")
				if err != nil {
					v.Unreadable = fmt.Errorf("invalid interface type: %v", err)
					return
				}
			}
		case "_type": // for runtime.eface
			_type, _ = v.toField(f)
			isnil = _type.maybeDereference().Addr == 0
		case "data":
			data, _ = v.toField(f)
		}
	}
	return
}

func (v *Variable) loadInterface(recurseLevel int, loadData bool, cfg LoadConfig) {
	_type, data, isnil := v.readInterface()

	if isnil {
		// interface to nil
		data = data.maybeDereference()
		v.Children = []Variable{*data}
		if loadData {
			v.Children[0].loadValueInternal(recurseLevel, cfg)
		}
		return
	}

	if data == nil {
		v.Unreadable = fmt.Errorf("invalid interface type")
		return
	}

	typ, kind, err := runtimeTypeToDIE(_type, data.Addr)
	if err != nil {
		v.Unreadable = err
		return
	}

	deref := false
	if kind&kindDirectIface == 0 {
		realtyp := resolveTypedef(typ)
		if _, isptr := realtyp.(*godwarf.PtrType); !isptr {
			typ = pointerTo(typ, v.bi.Arch)
			deref = true
		}
	}

	data = data.newVariable("data", data.Addr, typ, data.mem)
	if deref {
		data = data.maybeDereference()
		data.Name = "data"
	}

	v.Children = []Variable{*data}
	if loadData && recurseLevel <= cfg.MaxVariableRecurse {
		v.Children[0].loadValueInternal(recurseLevel, cfg)
	} else {
		v.Children[0].OnlyAddr = true
	}
}

// ConstDescr describes the value of v using constants.
func (v *Variable) ConstDescr() string {
	if v.bi == nil || (v.Flags&VariableConstant != 0) {
		return ""
	}
	ctyp := v.bi.consts.Get(v.DwarfType)
	if ctyp == nil {
		return ""
	}
	if typename := v.DwarfType.Common().Name; strings.Index(typename, ".") < 0 || strings.HasPrefix(typename, "C.") {
		// only attempt to use constants for user defined type, otherwise every
		// int variable with value 1 will be described with os.SEEK_CUR and other
		// similar problems.
		return ""
	}

	switch v.Kind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		fallthrough
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		n, _ := constant.Int64Val(v.Value)
		return ctyp.describe(n)
	}
	return ""
}

// popcnt is the number of bits set to 1 in x.
// It's the same as math/bits.OnesCount64, copied here so that we can build
// on versions of go that don't have math/bits.
func popcnt(x uint64) int {
	const m0 = 0x5555555555555555 // 01010101 ...
	const m1 = 0x3333333333333333 // 00110011 ...
	const m2 = 0x0f0f0f0f0f0f0f0f // 00001111 ...
	const m = 1<<64 - 1
	x = x>>1&(m0&m) + x&(m0&m)
	x = x>>2&(m1&m) + x&(m1&m)
	x = (x>>4 + x) & (m2 & m)
	x += x >> 8
	x += x >> 16
	x += x >> 32
	return int(x) & (1<<7 - 1)
}

func (cm constantsMap) Get(typ godwarf.Type) *constantType {
	ctyp := cm[typ.Common().Offset]
	if ctyp == nil {
		return nil
	}
	typepkg := packageName(typ.String()) + "."
	if !ctyp.initialized {
		ctyp.initialized = true
		sort.Sort(constantValuesByValue(ctyp.values))
		for i := range ctyp.values {
			if strings.HasPrefix(ctyp.values[i].name, typepkg) {
				ctyp.values[i].name = ctyp.values[i].name[len(typepkg):]
			}
			if popcnt(uint64(ctyp.values[i].value)) == 1 {
				ctyp.values[i].singleBit = true
			}
		}
	}
	return ctyp
}

func (ctyp *constantType) describe(n int64) string {
	for _, val := range ctyp.values {
		if val.value == n {
			return val.name
		}
	}

	if n == 0 {
		return ""
	}

	// If all the values for this constant only have one bit set we try to
	// represent the value as a bitwise or of constants.

	fields := []string{}
	for _, val := range ctyp.values {
		if !val.singleBit {
			continue
		}
		if n&val.value != 0 {
			fields = append(fields, val.name)
			n = n & ^val.value
		}
	}
	if n == 0 {
		return strings.Join(fields, "|")
	}
	return ""
}

type variablesByDepth struct {
	vars   []*Variable
	depths []int
}

func (v *variablesByDepth) Len() int { return len(v.vars) }

func (v *variablesByDepth) Less(i int, j int) bool { return v.depths[i] < v.depths[j] }

func (v *variablesByDepth) Swap(i int, j int) {
	v.depths[i], v.depths[j] = v.depths[j], v.depths[i]
	v.vars[i], v.vars[j] = v.vars[j], v.vars[i]
}

// Locals fetches all variables of a specific type in the current function scope.
func (scope *EvalScope) Locals() ([]*Variable, error) {
	if scope.Fn == nil {
		return nil, errors.New("unable to find function context")
	}

	var vars []*Variable
	var depths []int
	varReader := reader.Variables(scope.BinInfo.dwarf, scope.Fn.offset, reader.ToRelAddr(scope.PC, scope.BinInfo.staticBase), scope.Line, true)
	hasScopes := false
	for varReader.Next() {
		entry := varReader.Entry()
		val, err := scope.extractVarInfoFromEntry(entry)
		if err != nil {
			// skip variables that we can't parse yet
			continue
		}
		vars = append(vars, val)
		depth := varReader.Depth()
		if entry.Tag == dwarf.TagFormalParameter {
			if depth <= 1 {
				depth = 0
			}
			isret, _ := entry.Val(dwarf.AttrVarParam).(bool)
			if isret {
				val.Flags |= VariableReturnArgument
			} else {
				val.Flags |= VariableArgument
			}
		}
		depths = append(depths, depth)
		if depth > 1 {
			hasScopes = true
		}
	}

	if err := varReader.Err(); err != nil {
		return nil, err
	}

	if len(vars) <= 0 {
		return vars, nil
	}

	if hasScopes {
		sort.Stable(&variablesByDepth{vars, depths})
	}

	lvn := map[string]*Variable{} // lvn[n] is the last variable we saw named n

	for i, v := range vars {
		if name := v.Name; len(name) > 1 && name[0] == '&' {
			v = v.maybeDereference()
			if v.Addr == 0 {
				v.Unreadable = fmt.Errorf("no address for escaped variable")
			}
			v.Name = name[1:]
			v.Flags |= VariableEscaped
			vars[i] = v
		}
		if hasScopes {
			if otherv := lvn[v.Name]; otherv != nil {
				otherv.Flags |= VariableShadowed
			}
			lvn[v.Name] = v
		}
	}

	return vars, nil
}

type constantValuesByValue []constantValue

func (v constantValuesByValue) Len() int               { return len(v) }
func (v constantValuesByValue) Less(i int, j int) bool { return v[i].value < v[j].value }
func (v constantValuesByValue) Swap(i int, j int)      { v[i], v[j] = v[j], v[i] }
