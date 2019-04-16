package debugger

import (
	"fmt"
	"go/constant"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"

	"github.com/go-delve/delve/pkg/proc"
	"github.com/go-delve/delve/service/api"
)

const maxFindLocationCandidates = 5

type LocationSpec interface {
	Find(d *Debugger, scope *proc.EvalScope, locStr string) ([]api.Location, error)
}

type NormalLocationSpec struct {
	Base       string
	FuncBase   *FuncLocationSpec
	LineOffset int
}

type RegexLocationSpec struct {
	FuncRegex string
}

type AddrLocationSpec struct {
	AddrExpr string
}

type OffsetLocationSpec struct {
	Offset int
}

type LineLocationSpec struct {
	Line int
}

type FuncLocationSpec struct {
	PackageName           string
	AbsolutePackage       bool
	ReceiverName          string
	PackageOrReceiverName string
	BaseName              string
}

func parseLocationSpec(locStr string) (LocationSpec, error) {
	rest := locStr

	malformed := func(reason string) error {
		return fmt.Errorf("Malformed breakpoint location \"%s\" at %d: %s", locStr, len(locStr)-len(rest), reason)
	}

	if len(rest) <= 0 {
		return nil, malformed("empty string")
	}

	switch rest[0] {
	case '+', '-':
		offset, err := strconv.Atoi(rest)
		if err != nil {
			return nil, malformed(err.Error())
		}
		return &OffsetLocationSpec{offset}, nil

	case '/':
		if rest[len(rest)-1] == '/' {
			rx, rest := readRegex(rest[1:])
			if len(rest) < 0 {
				return nil, malformed("non-terminated regular expression")
			}
			if len(rest) > 1 {
				return nil, malformed("no line offset can be specified for regular expression locations")
			}
			return &RegexLocationSpec{rx}, nil
		} else {
			return parseLocationSpecDefault(locStr, rest)
		}

	case '*':
		return &AddrLocationSpec{rest[1:]}, nil

	default:
		return parseLocationSpecDefault(locStr, rest)
	}
}

func parseLocationSpecDefault(locStr, rest string) (LocationSpec, error) {
	malformed := func(reason string) error {
		return fmt.Errorf("Malformed breakpoint location \"%s\" at %d: %s", locStr, len(locStr)-len(rest), reason)
	}

	v := strings.Split(rest, ":")
	if len(v) > 2 {
		// On Windows, path may contain ":", so split only on last ":"
		v = []string{strings.Join(v[0:len(v)-1], ":"), v[len(v)-1]}
	}

	if len(v) == 1 {
		n, err := strconv.ParseInt(v[0], 0, 64)
		if err == nil {
			return &LineLocationSpec{int(n)}, nil
		}
	}

	spec := &NormalLocationSpec{}

	spec.Base = v[0]
	spec.FuncBase = parseFuncLocationSpec(spec.Base)

	if len(v) < 2 {
		spec.LineOffset = -1
		return spec, nil
	}

	rest = v[1]

	var err error
	spec.LineOffset, err = strconv.Atoi(rest)
	if err != nil || spec.LineOffset < 0 {
		return nil, malformed("line offset negative or not a number")
	}

	return spec, nil
}

func readRegex(in string) (rx string, rest string) {
	out := make([]rune, 0, len(in))
	escaped := false
	for i, ch := range in {
		if escaped {
			if ch == '/' {
				out = append(out, '/')
			} else {
				out = append(out, '\\')
				out = append(out, ch)
			}
			escaped = false
		} else {
			switch ch {
			case '\\':
				escaped = true
			case '/':
				return string(out), in[i:]
			default:
				out = append(out, ch)
			}
		}
	}
	return string(out), ""
}

func parseFuncLocationSpec(in string) *FuncLocationSpec {
	var v []string
	pathend := strings.LastIndex(in, "/")
	if pathend < 0 {
		v = strings.Split(in, ".")
	} else {
		v = strings.Split(in[pathend:], ".")
		if len(v) > 0 {
			v[0] = in[:pathend] + v[0]
		}
	}

	var spec FuncLocationSpec
	switch len(v) {
	case 1:
		spec.BaseName = v[0]

	case 2:
		spec.BaseName = v[1]
		r := stripReceiverDecoration(v[0])
		if r != v[0] {
			spec.ReceiverName = r
		} else if strings.Contains(r, "/") {
			spec.PackageName = r
		} else {
			spec.PackageOrReceiverName = r
		}

	case 3:
		spec.BaseName = v[2]
		spec.ReceiverName = stripReceiverDecoration(v[1])
		spec.PackageName = v[0]

	default:
		return nil
	}

	if strings.HasPrefix(spec.PackageName, "/") {
		spec.PackageName = spec.PackageName[1:]
		spec.AbsolutePackage = true
	}

	if strings.Contains(spec.BaseName, "/") || strings.Contains(spec.ReceiverName, "/") {
		return nil
	}

	return &spec
}

func stripReceiverDecoration(in string) string {
	if len(in) < 3 {
		return in
	}
	if (in[0] != '(') || (in[1] != '*') || (in[len(in)-1] != ')') {
		return in
	}

	return in[2 : len(in)-1]
}

func (spec *FuncLocationSpec) Match(sym proc.Function) bool {
	if spec.BaseName != sym.BaseName() {
		return false
	}

	recv := stripReceiverDecoration(sym.ReceiverName())
	if spec.ReceiverName != "" && spec.ReceiverName != recv {
		return false
	}
	if spec.PackageName != "" {
		if spec.AbsolutePackage {
			if spec.PackageName != sym.PackageName() {
				return false
			}
		} else {
			if !partialPathMatch(spec.PackageName, sym.PackageName()) {
				return false
			}
		}
	}
	if spec.PackageOrReceiverName != "" && !partialPathMatch(spec.PackageOrReceiverName, sym.PackageName()) && spec.PackageOrReceiverName != recv {
		return false
	}
	return true
}

func (loc *RegexLocationSpec) Find(d *Debugger, scope *proc.EvalScope, locStr string) ([]api.Location, error) {
	funcs := d.target.BinInfo().Functions
	matches, err := regexFilterFuncs(loc.FuncRegex, funcs)
	if err != nil {
		return nil, err
	}
	r := make([]api.Location, 0, len(matches))
	for i := range matches {
		addr, err := proc.FindFunctionLocation(d.target, matches[i], true, 0)
		if err == nil {
			r = append(r, api.Location{PC: addr})
		}
	}
	return r, nil
}

func (loc *AddrLocationSpec) Find(d *Debugger, scope *proc.EvalScope, locStr string) ([]api.Location, error) {
	if scope == nil {
		addr, err := strconv.ParseInt(loc.AddrExpr, 0, 64)
		if err != nil {
			return nil, fmt.Errorf("could not determine current location (scope is nil)")
		}
		return []api.Location{{PC: uint64(addr)}}, nil
	} else {
		v, err := scope.EvalExpression(loc.AddrExpr, proc.LoadConfig{true, 0, 0, 0, 0, 0})
		if err != nil {
			return nil, err
		}
		if v.Unreadable != nil {
			return nil, v.Unreadable
		}
		switch v.Kind {
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			addr, _ := constant.Uint64Val(v.Value)
			return []api.Location{{PC: addr}}, nil
		case reflect.Func:
			_, _, fn := d.target.BinInfo().PCToLine(uint64(v.Base))
			pc, err := proc.FirstPCAfterPrologue(d.target, fn, false)
			if err != nil {
				return nil, err
			}
			return []api.Location{{PC: uint64(pc)}}, nil
		default:
			return nil, fmt.Errorf("wrong expression kind: %v", v.Kind)
		}
	}
}

func (loc *NormalLocationSpec) FileMatch(path string) bool {
	return partialPathMatch(loc.Base, path)
}

func tryMatchRelativePathByProc(expr, debugname, file string) bool {
	return len(expr) > 0 && expr[0] == '.' && file == path.Join(path.Dir(debugname), expr)
}

func partialPathMatch(expr, path string) bool {
	if runtime.GOOS == "windows" {
		// Accept `expr` which is case-insensitive and slash-insensitive match to `path`
		expr = strings.ToLower(filepath.ToSlash(expr))
		path = strings.ToLower(filepath.ToSlash(path))
	}
	if len(expr) < len(path)-1 {
		return strings.HasSuffix(path, expr) && (path[len(path)-len(expr)-1] == '/')
	} else {
		return expr == path
	}
}

type AmbiguousLocationError struct {
	Location           string
	CandidatesString   []string
	CandidatesLocation []api.Location
}

func (ale AmbiguousLocationError) Error() string {
	var candidates []string
	if ale.CandidatesLocation != nil {
		for i := range ale.CandidatesLocation {
			candidates = append(candidates, ale.CandidatesLocation[i].Function.Name())
		}

	} else {
		candidates = ale.CandidatesString
	}
	return fmt.Sprintf("Location \"%s\" ambiguous: %s…", ale.Location, strings.Join(candidates, ", "))
}

func (loc *NormalLocationSpec) Find(d *Debugger, scope *proc.EvalScope, locStr string) ([]api.Location, error) {
	limit := maxFindLocationCandidates
	var candidateFiles []string
	for _, file := range d.target.BinInfo().Sources {
		if loc.FileMatch(file) || (len(d.processArgs) >= 1 && tryMatchRelativePathByProc(loc.Base, d.processArgs[0], file)) {
			candidateFiles = append(candidateFiles, file)
			if len(candidateFiles) >= limit {
				break
			}
		}
	}

	limit -= len(candidateFiles)

	var candidateFuncs []string
	if loc.FuncBase != nil {
		for _, f := range d.target.BinInfo().Functions {
			if !loc.FuncBase.Match(f) {
				continue
			}
			if loc.Base == f.Name {
				// if an exact match for the function name is found use it
				candidateFuncs = []string{f.Name}
				break
			}
			candidateFuncs = append(candidateFuncs, f.Name)
			if len(candidateFuncs) >= limit {
				break
			}
		}
	}

	if matching := len(candidateFiles) + len(candidateFuncs); matching == 0 {
		// if no result was found treat this locations string could be an
		// expression that the user forgot to prefix with '*', try treating it as
		// such.
		addrSpec := &AddrLocationSpec{locStr}
		locs, err := addrSpec.Find(d, scope, locStr)
		if err != nil {
			return nil, fmt.Errorf("Location \"%s\" not found", locStr)
		}
		return locs, nil
	} else if matching > 1 {
		return nil, AmbiguousLocationError{Location: locStr, CandidatesString: append(candidateFiles, candidateFuncs...)}
	}

	// len(candidateFiles) + len(candidateFuncs) == 1
	var addr uint64
	var err error
	if len(candidateFiles) == 1 {
		if loc.LineOffset < 0 {
			return nil, fmt.Errorf("Malformed breakpoint location, no line offset specified")
		}
		addr, err = proc.FindFileLocation(d.target, candidateFiles[0], loc.LineOffset)
	} else { // len(candidateFUncs) == 1
		if loc.LineOffset < 0 {
			addr, err = proc.FindFunctionLocation(d.target, candidateFuncs[0], true, 0)
		} else {
			addr, err = proc.FindFunctionLocation(d.target, candidateFuncs[0], false, loc.LineOffset)
		}
	}

	if err != nil {
		return nil, err
	}
	return []api.Location{{PC: addr}}, nil
}

func (loc *OffsetLocationSpec) Find(d *Debugger, scope *proc.EvalScope, locStr string) ([]api.Location, error) {
	if scope == nil {
		return nil, fmt.Errorf("could not determine current location (scope is nil)")
	}
	if loc.Offset == 0 {
		return []api.Location{{PC: scope.PC}}, nil
	}
	file, line, fn := d.target.BinInfo().PCToLine(scope.PC)
	if fn == nil {
		return nil, fmt.Errorf("could not determine current location")
	}
	addr, err := proc.FindFileLocation(d.target, file, line+loc.Offset)
	return []api.Location{{PC: addr}}, err
}

func (loc *LineLocationSpec) Find(d *Debugger, scope *proc.EvalScope, locStr string) ([]api.Location, error) {
	if scope == nil {
		return nil, fmt.Errorf("could not determine current location (scope is nil)")
	}
	file, _, fn := d.target.BinInfo().PCToLine(scope.PC)
	if fn == nil {
		return nil, fmt.Errorf("could not determine current location")
	}
	addr, err := proc.FindFileLocation(d.target, file, loc.Line)
	return []api.Location{{PC: addr}}, err
}
