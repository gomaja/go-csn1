package registry

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"math/rand"
	"path/filepath"
	"reflect"
	"regexp"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/gomaja/go-csn1/runtime"
)

// Typed error classes of the generated canonical entry points.
const (
	classRequired       = "extent-required"   // *ExtentError with Required set
	classLonger         = "extent-longer"     // encoding longer than the maximum
	classRange          = "extent-range"      // requested octets outside the extent
	classUnfilled       = "extent-unfilled"   // encoding shorter than requested
	classTarget         = "bound-target"      // BoundError CanonicalTarget
	classLength         = "bound-length"      // BoundError LengthBound
	classContext        = "context"           // wraps ErrContextRequired
	classContextLayout  = "context-acs"       // *ContextRequiredError (SI4 ACS)
	classDecode         = "decode"            // wraps a *DecodeError
	classFallback       = "fallback-known"    // *FallbackError
	entryCanonical      = "Canonical"         //
	entryAtLength       = "CanonicalAtLength" //
	entryWithContext    = "CanonicalWithContext"
	runtimeOctetMaximum = 1 << 17
)

// documentedClasses reads the classes each generated canonical entry point
// documents, keyed by package directory and function name: two packages may
// define the same function name. It also returns each package name's
// directory, and fails on a duplicate key.
func documentedClasses(t *testing.T) (map[string]map[string]bool, map[string]string) {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "ts*", "*", "generated.go"))
	if err != nil || len(files) != 8 {
		t.Fatalf("generated files %v, %v", files, err)
	}
	out := map[string]map[string]bool{}
	packages := map[string]string{}
	type delegation struct{ dir, text string }
	delegations := map[string]delegation{}
	for _, file := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, parser.ParseComments)
		if err != nil {
			t.Fatal(err)
		}
		dir := filepath.ToSlash(filepath.Dir(strings.TrimPrefix(file, ".."+string(filepath.Separator))))
		if prior, ok := packages[parsed.Name.Name]; ok && prior != dir {
			t.Fatalf("package name %s in %s and %s", parsed.Name.Name, prior, dir)
		}
		packages[parsed.Name.Name] = dir
		for _, decl := range parsed.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Doc == nil || !strings.HasPrefix(fn.Name.Name, "Encode") || !strings.Contains(fn.Name.Name, "Canonical") {
				continue
			}
			key := dir + "." + fn.Name.Name
			if _, ok := out[key]; ok {
				t.Fatalf("duplicate documented function %s", key)
			}
			doc := strings.Join(strings.Fields(fn.Doc.Text()), " ")
			classes := map[string]bool{}
			add := func(class string, present bool) {
				if present {
					classes[class] = true
				}
			}
			add(classRequired, strings.Contains(doc, "*runtime.ExtentError with Required set"))
			add(classLonger, strings.Contains(doc, "*runtime.ExtentError when the encoding is longer than"))
			add(classRange, strings.Contains(doc, "*runtime.ExtentError when octets is"))
			add(classUnfilled, strings.Contains(doc, "encoding does not fill exactly octets"))
			add(classTarget, strings.Contains(doc, "Kind runtime.CanonicalTarget"))
			add(classLength, strings.Contains(doc, "Kind runtime.LengthBound"))
			add(classContext, strings.Contains(doc, "wrapping runtime.ErrContextRequired"))
			add(classContextLayout, strings.Contains(doc, "*runtime.ContextRequiredError"))
			add(classFallback, strings.Contains(doc, "*runtime.FallbackError"))
			out[key] = classes
			if _, delegated, ok := strings.Cut(doc, "returns the errors of "); ok {
				delegations[key] = delegation{dir, delegated}
			}
		}
	}
	// A wrapper documents the errors of the canonical encoders it calls; an
	// unqualified callee is in the wrapper's own package.
	for key, d := range delegations {
		for _, callee := range delegatedCallee.FindAllStringSubmatch(d.text, -1) {
			dir := d.dir
			if callee[1] != "" {
				dir = packages[callee[1]]
			}
			classes, ok := out[dir+"."+callee[2]]
			if !ok {
				t.Fatalf("%s delegates to unknown %s", key, callee[0])
			}
			for class := range classes {
				out[key][class] = true
			}
		}
	}
	return out, packages
}

var delegatedCallee = regexp.MustCompile(`(?:([a-z]+)\.)?(Encode[A-Za-z0-9]*Canonical)\b`)

// canonicalFunctions maps each descriptor (clause and name) to the package
// directory and function its Canonical closure calls, and fails when one
// descriptor key maps to two functions.
func canonicalFunctions(t *testing.T, packages map[string]string) map[string]string {
	t.Helper()
	files, _ := filepath.Glob(filepath.Join("..", "ts*", "*", "generated.go"))
	files = append(files, "registry.go")
	out := map[string]string{}
	for _, file := range files {
		parsed, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		own := packages[parsed.Name.Name]
		ast.Inspect(parsed, func(n ast.Node) bool {
			lit, ok := n.(*ast.CompositeLit)
			if !ok {
				return true
			}
			var clause, name, call string
			for _, element := range lit.Elts {
				kv, ok := element.(*ast.KeyValueExpr)
				if !ok {
					continue
				}
				key, _ := kv.Key.(*ast.Ident)
				if key == nil {
					continue
				}
				switch key.Name {
				case "Clause", "Name":
					if basic, ok := kv.Value.(*ast.BasicLit); ok {
						value, _ := strconv.Unquote(basic.Value)
						if key.Name == "Clause" {
							clause = value
						} else {
							name = value
						}
					}
				case "Canonical":
					ast.Inspect(kv.Value, func(m ast.Node) bool {
						if c, ok := m.(*ast.CallExpr); ok {
							switch fn := c.Fun.(type) {
							case *ast.Ident:
								if strings.HasSuffix(fn.Name, "Canonical") && own != "" {
									call = own + "." + fn.Name
								}
							case *ast.SelectorExpr:
								if pkg, ok := fn.X.(*ast.Ident); ok && strings.HasSuffix(fn.Sel.Name, "Canonical") && packages[pkg.Name] != "" {
									call = packages[pkg.Name] + "." + fn.Sel.Name
								}
							}
						}
						return call == ""
					})
				}
			}
			if name != "" && call != "" {
				key := clause + "\x00" + name
				if prior, ok := out[key]; ok && prior != call {
					t.Fatalf("descriptor %s §%s maps to %s and %s", name, clause, prior, call)
				}
				out[key] = call
			}
			return true
		})
	}
	return out
}

func classify(err error, octets int, entry string) string {
	var bound *runtime.BoundError
	var extent *runtime.ExtentError
	var layout *runtime.ContextRequiredError
	var decode *runtime.DecodeError
	var fallback *runtime.FallbackError
	switch {
	case err == nil:
		return ""
	case errors.As(err, &fallback):
		return classFallback
	case errors.As(err, &bound):
		if bound.Kind == runtime.CanonicalTarget {
			return classTarget
		}
		if bound.Kind == runtime.LengthBound {
			return classLength
		}
		return "bound-" + string(bound.Kind)
	case errors.As(err, &extent):
		switch {
		case extent.Required:
			return classRequired
		case entry == entryAtLength && extent.Actual == octets:
			return classRange
		case entry == entryAtLength && extent.Minimum == octets && extent.Maximum == octets && extent.Actual < octets:
			return classUnfilled
		case extent.Maximum > 0 && extent.Actual > extent.Maximum:
			return classLonger
		default:
			return fmt.Sprintf("extent-other(%+v)", *extent)
		}
	case errors.As(err, &layout):
		return classContextLayout
	case errors.Is(err, runtime.ErrContextRequired):
		return classContext
	case errors.As(err, &decode):
		// No generated comment lists a decode-back failure; provoking one
		// is a mismatch, for example an ambiguous choice.
		return classDecode
	}
	return ""
}

// guidedDecode decodes biased random bits, resampling from the offset where
// the decoder rejects them, so most grammars yield valid values; a high
// one-bit bias lengthens {1 ...}** lists and selects optional arms.
func guidedDecode(rng *rand.Rand, decode func([]byte) (any, error)) []any {
	var out []any
	for _, bias := range []float64{0.1, 0.5, 0.8, 0.95} {
		// Empty and one-octet inputs select null alternatives.
		for _, size := range []int{0, 1, 8, 20, 64, 200} {
			bits := make([]bool, size*8)
			fill := func(from int) {
				for i := from; i < len(bits); i++ {
					bits[i] = rng.Float64() < bias
				}
			}
			fill(0)
			for range 60 {
				wire := make([]byte, size)
				for i, b := range bits {
					if b {
						wire[i/8] |= 1 << (7 - uint(i%8))
					}
				}
				value, err := decode(wire)
				if err == nil {
					out = append(out, value)
					if len(bits) == 0 {
						break
					}
					fill(rng.Intn(len(bits)))
					continue
				}
				at := 0
				var de *runtime.DecodeError
				if errors.As(err, &de) && de.Offset > 0 && de.Offset <= len(bits) {
					// Resample near the failure, sometimes further back:
					// some checks report after the bits they reject.
					at = de.Offset - 1 - rng.Intn(min(de.Offset, []int{8, 16, 64}[rng.Intn(3)]))
				}
				fill(at)
			}
		}
	}
	return out
}

// harvest records every struct value nested in a decoded value, so that
// component definitions without a standalone decode still get values.
func harvest(v reflect.Value, into map[reflect.Type][]reflect.Value, depth int) {
	if depth > 40 || !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			harvest(v.Elem(), into, depth+1)
		}
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[runtime.WireInfo]() || v.Type() == reflect.TypeFor[runtime.BitString]() {
			return
		}
		if len(into[v.Type()]) < 12 {
			into[v.Type()] = append(into[v.Type()], v)
		}
		for i := range v.NumField() {
			harvest(v.Field(i), into, depth+1)
		}
	case reflect.Slice:
		for i := range min(v.Len(), 4) {
			harvest(v.Index(i), into, depth+1)
		}
	}
}

// variants yields edited copies of a value: random edits, grown lists and
// one unsigned field at a time set to an extreme.
func variants(rng *rand.Rand, base any, leaves int, yield func(any)) {
	yield(base)
	for range 4 {
		m := deepCopy(base)
		mutateTyped(rng, m, 0)
		yield(m.Interface())
	}
	for _, times := range []int{1, 8, 30} {
		g := deepCopy(base)
		grow(g, 0, times, true)
		yield(g.Interface())
		g = deepCopy(base)
		grow(g, 0, times, false)
		yield(g.Interface())
	}
	// Grow one list at a time, so a counted list elsewhere keeps its count.
	var single []reflect.Value
	sliceFields(deepCopy(base), 0, &single)
	for li := range min(len(single), 8) {
		for _, size := range []int{8, 30} {
			c := deepCopy(base)
			var ls []reflect.Value
			sliceFields(c, 0, &ls)
			if li >= len(ls) || ls[li].Len() == 0 {
				break
			}
			list := ls[li]
			for k := 0; list.Len() < size; k++ {
				list.Set(reflect.Append(list, list.Index(k)))
			}
			yield(c.Interface())
		}
	}
	// Switch each fallback to its ignored arm with random bits, and set the
	// enclosing length to them: TS 44.060 V19.0.0 §12.24 and TS 44.018
	// V19.0.0 §10.5.2.33b print < bit (val(<length>) + 1) & { <known> !
	// <ignored> } >. The known arm then often decodes the bits.
	var fallbacks []fallbackSite
	fallbackFields(deepCopy(base), 0, &fallbacks)
	for fi := range min(len(fallbacks), 4) {
		for range 3 {
			c := deepCopy(base)
			var fs []fallbackSite
			fallbackFields(c, 0, &fs)
			if fi >= len(fs) {
				break
			}
			site := fs[fi]
			// 1..64 bits: Extension Length is 6 bits in §12.24, 8 in §10.5.2.33b.
			n := 1 + rng.Intn(64)
			raw := make([]byte, (n+7)/8)
			rng.Read(raw)
			bits := runtime.BitString{Bytes: raw, BitLength: n}
			site.fallback.FieldByName("Alternative").SetUint(1)
			known := site.fallback.FieldByName("Known")
			known.Set(reflect.Zero(known.Type()))
			site.fallback.FieldByName("Ignored").Set(reflect.ValueOf(&bits))
			site.length.SetUint(uint64(n - 1))
			yield(c.Interface())
		}
	}
	// Append a copy of the last element of each list, then set the copy's
	// fields to extremes: this reaches counts that must fill a remainder.
	var lists []reflect.Value
	sliceFields(deepCopy(base), 0, &lists)
	for li := range min(len(lists), 8) {
		for _, extreme := range []uint64{0, 1, 31, 63, 255} {
			for leaf := 0; leaf < 6; leaf++ {
				c := deepCopy(base)
				var ls []reflect.Value
				sliceFields(c, 0, &ls)
				if li >= len(ls) || ls[li].Len() == 0 {
					break
				}
				list := ls[li]
				list.Set(reflect.Append(list, list.Index(list.Len()-1)))
				var leaves []reflect.Value
				uintLeaves(list.Index(list.Len()-1), 0, &leaves)
				if leaf >= len(leaves) || leaves[leaf].OverflowUint(extreme) {
					break
				}
				leaves[leaf].SetUint(extreme)
				yield(c.Interface())
			}
		}
	}
	var all []reflect.Value
	uintLeaves(deepCopy(base), 0, &all)
	picks := rng.Perm(len(all))
	if len(picks) > leaves {
		picks = picks[:leaves]
	}
	for _, i := range picks {
		for _, extreme := range []uint64{0, 1, 31, 63, 255} {
			c := deepCopy(base)
			var ls []reflect.Value
			uintLeaves(c, 0, &ls)
			if i >= len(ls) || ls[i].OverflowUint(extreme) {
				continue
			}
			ls[i].SetUint(extreme)
			yield(c.Interface())
		}
	}
}

func deepCopy(value any) reflect.Value {
	c := runtime.Canonical(value)
	out := reflect.New(reflect.TypeOf(c)).Elem()
	out.Set(reflect.ValueOf(c))
	return out
}

func uintLeaves(v reflect.Value, depth int, out *[]reflect.Value) {
	if depth > 30 || !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v.CanSet() {
			*out = append(*out, v)
		}
	case reflect.Pointer:
		if !v.IsNil() {
			uintLeaves(v.Elem(), depth+1, out)
		}
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[runtime.WireInfo]() || v.Type() == reflect.TypeFor[runtime.BitString]() {
			return
		}
		for i := range v.NumField() {
			uintLeaves(v.Field(i), depth+1, out)
		}
	case reflect.Slice:
		for i := range v.Len() {
			uintLeaves(v.Index(i), depth+1, out)
		}
	}
}

// grow appends copies of list elements up to 64. Nested growth also grows
// lists inside elements; outer growth keeps each copied element intact.
func grow(v reflect.Value, depth, times int, nested bool) {
	if depth > 30 || !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			grow(v.Elem(), depth+1, times, nested)
		}
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[runtime.WireInfo]() || v.Type() == reflect.TypeFor[runtime.BitString]() {
			return
		}
		for i := range v.NumField() {
			if v.Field(i).CanSet() {
				grow(v.Field(i), depth+1, times, nested)
			}
		}
	case reflect.Slice:
		if nested {
			for i := range v.Len() {
				grow(v.Index(i), depth+1, times, nested)
			}
		}
		if v.Len() > 0 && v.CanSet() {
			n := v.Len()
			for k := 0; k < n*times && v.Len() < 64; k++ {
				v.Set(reflect.Append(v, v.Index(k%n)))
			}
		}
	}
}

// TestCanonicalDocsListExactlyReachableErrors checks the per-definition
// error lists on the generated canonical entry points. For every definition
// and entry point, each documented typed error class must be provoked by a
// synthetic value, and each provoked class must be documented. The generator
// derives the lists statically (csn1/gen/errorprofile.go); this test is the
// independent evidence that the static analysis is neither loose nor
// incomplete on the generated codecs.
func TestCanonicalDocsListExactlyReachableErrors(t *testing.T) {
	if testing.Short() {
		t.Skip("explores every definition")
	}
	documented, packages := documentedClasses(t)
	functions := canonicalFunctions(t, packages)
	definitions := descriptors()

	// Decode-guided values, then nested values harvested by type.
	values := make([][]any, len(definitions))
	locals := make([]map[reflect.Type][]reflect.Value, len(definitions))
	var wg sync.WaitGroup
	for index, d := range definitions {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rng := rand.New(rand.NewSource(int64(index)))
			var found []any
			switch {
			case d.DecodeWithContext != nil:
				for _, acs := range []runtime.SI4ACS{runtime.SI4ACSZero, runtime.SI4ACSOne} {
					found = append(found, guidedDecode(rng, func(b []byte) (any, error) {
						decoded, err := d.DecodeWithContext(b, acs)
						if err != nil {
							return nil, err
						}
						return reflect.ValueOf(decoded).FieldByName("Value").Interface(), nil
					})...)
				}
			case d.DecodeFrom != nil:
				found = guidedDecode(rng, func(b []byte) (any, error) { return d.DecodeFrom(runtime.NewReader(b)) })
			default:
				found = guidedDecode(rng, func(b []byte) (any, error) {
					decoded, err := d.Decode(b)
					if err != nil {
						return nil, err
					}
					return reflect.ValueOf(decoded).FieldByName("Value").Interface(), nil
				})
			}
			for _, seed := range canonicalSeeds[d.Clause+"\x00"+d.Name] {
				wire, err := hex.DecodeString(seed)
				if err != nil {
					t.Error(err)
					continue
				}
				if decoded, err := d.Decode(wire); err == nil {
					found = append(found, reflect.ValueOf(decoded).FieldByName("Value").Interface())
				}
			}
			local := map[reflect.Type][]reflect.Value{}
			for _, v := range found {
				harvest(reflect.ValueOf(v), local, 0)
			}
			values[index], locals[index] = found, local
		}()
	}
	wg.Wait()
	// Merge in descriptor order, so the capped corpus does not depend on
	// which worker finished first.
	harvested := map[reflect.Type][]reflect.Value{}
	for _, local := range locals {
		for typ, list := range local {
			for _, v := range list {
				if len(harvested[typ]) < 12 {
					harvested[typ] = append(harvested[typ], v)
				}
			}
		}
	}
	t.Logf("harvest digest %x", corpusDigest(values, harvested))

	types := map[reflect.Type]bool{}
	for _, list := range values {
		for _, v := range list {
			structTypes(reflect.TypeOf(v), types)
		}
	}

	type result struct {
		key      string
		observed map[string]map[string]bool
	}
	results := make([]result, len(definitions))
	sem := make(chan struct{}, max(goruntime.GOMAXPROCS(0), 4))
	for index, d := range definitions {
		own := values[index]
		if len(own) == 0 {
			// A component bound by its containing message: use harvested
			// values of the definition's type, or else its zero value.
			for typ := range types {
				zero := reflect.Zero(typ).Interface()
				if _, err := d.Canonical(zero); err != nil && strings.Contains(err.Error(), "wrong value type") {
					continue
				}
				own = append(own, zero)
				for _, v := range harvested[typ] {
					own = append(own, v.Interface())
				}
				break
			}
		}
		own = selectValues(d, own, canonicalSeedCount(d))
		if len(own) > 0 {
			// The zero value selects first and empty alternatives.
			own = append(own, reflect.Zero(reflect.TypeOf(own[0])).Interface())
		}
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer func() { <-sem; wg.Done() }()
			rng := rand.New(rand.NewSource(int64(index) + 1<<20))
			observed := map[string]map[string]bool{entryCanonical: {}, entryAtLength: {}, entryWithContext: {}}
			record := func(entry string, err error, octets int) {
				if c := classify(err, octets, entry); c != "" {
					observed[entry][c] = true
				}
			}
			octets := []int{-1, 0, 1, 8, 20, 64, runtimeOctetMaximum + 1}
			if d.MaxOctets > 0 {
				octets = append(octets, d.MaxOctets, d.MaxOctets+1)
			}
			for _, base := range own {
				variants(rng, base, 40, func(v any) {
					if d.Canonical != nil {
						_, err := d.Canonical(v)
						record(entryCanonical, err, 0)
					}
					if d.CanonicalAtLength != nil {
						for _, o := range octets {
							_, err := d.CanonicalAtLength(v, o)
							record(entryAtLength, err, o)
						}
					}
					if d.CanonicalWithContext != nil {
						for _, acs := range []runtime.SI4ACS{runtime.SI4ACSZero, runtime.SI4ACSOne} {
							_, err := d.CanonicalWithContext(v, acs)
							record(entryWithContext, err, 0)
						}
					}
				})
			}
			results[index] = result{d.Standard + " §" + d.Clause + " <" + d.Name + ">", observed}
		}()
	}
	wg.Wait()

	var failures []string
	for index, d := range definitions {
		function := functions[d.Clause+"\x00"+d.Name]
		if function == "" {
			t.Fatalf("%s: no generated canonical function", results[index].key)
		}
		for _, entry := range []string{entryCanonical, entryAtLength, entryWithContext} {
			name := function
			switch entry {
			case entryAtLength:
				if d.CanonicalAtLength == nil {
					continue
				}
				name += "AtLength"
			case entryWithContext:
				if d.CanonicalWithContext == nil {
					continue
				}
				name += "WithContext"
			}
			docs, ok := documented[name]
			if !ok {
				t.Fatalf("%s: %s has no documentation", results[index].key, name)
			}
			observed := results[index].observed[entry]
			var missing, undocumented []string
			for c := range docs {
				if !observed[c] {
					missing = append(missing, c)
				}
			}
			for c := range observed {
				if !docs[c] {
					undocumented = append(undocumented, c)
				}
			}
			sort.Strings(missing)
			sort.Strings(undocumented)
			if pending := pendingAmbiguity[d.Clause+"\x00"+d.Name+"\x00"+entry]; pending != "" {
				// The mismatch must be exactly the pending one; a fix or a
				// new class fails until this table is updated.
				if len(missing) != 0 || strings.Join(undocumented, ",") != pending {
					failures = append(failures, fmt.Sprintf("%s %s: pending ambiguity changed: documented but not provoked %v; provoked but undocumented %v", results[index].key, name, missing, undocumented))
				}
				continue
			}
			if len(missing)+len(undocumented) > 0 {
				failures = append(failures, fmt.Sprintf("%s %s: documented but not provoked %v; provoked but undocumented %v", results[index].key, name, missing, undocumented))
			}
		}
	}
	sort.Strings(failures)
	for _, f := range failures {
		t.Error(f)
	}
	t.Logf("%d definitions, %d mismatches", len(definitions), len(failures))
}

// structTypes collects every struct type reachable from a value type,
// including types of nil optional fields.
func structTypes(t reflect.Type, into map[reflect.Type]bool) {
	switch t.Kind() {
	case reflect.Pointer, reflect.Slice:
		structTypes(t.Elem(), into)
	case reflect.Struct:
		if into[t] || t == reflect.TypeFor[runtime.WireInfo]() || t == reflect.TypeFor[runtime.BitString]() {
			return
		}
		into[t] = true
		for i := range t.NumField() {
			structTypes(t.Field(i).Type, into)
		}
	}
}

// canonicalSeeds are synthetic inputs for arms that random bits rarely
// select. pycrate 0.7.11 decodes 82000008 as IA Rest Octets HL, Length of
// frequency parameters 2 and the MAIO arm, and 8008 as HL, length 0 and the
// null arm.
var canonicalSeeds = map[string][]string{
	"10.5.2.16\x00IA Rest Octets": {"82000008", "8008"},
	// Short GPRS Cell Options extensions (TS 44.060 V19.0.0 §12.24), alone
	// and inside SI 13; pycrate and Wireshark agree on their decoding.
	"12.24\x00GPRS Cell Options IE":   {"b0e1d5122d103fc76dfdb8ebeb652a", "b48f27c6a62ca8ce639d2adb"},
	"10.5.2.37b\x00SI 13 Rest Octets": {"e1b88bbdf784a1a279df24ba4896ad235775bf40", "ea15976ab36992bb4a4df259d63dd40957e08528", "dd0011a6fea7fbd89c88ffa33d38bfbdb253df4b"},
	// TS 44.018 V19.0.0 §10.5.2.37o with UTRAN neighbour frequencies,
	// confirmed by pycrate 0.7.11 (ts44018/restoctets/utran_fdd_tdd_test.go).
	"10.5.2.37o\x00SI 23 Rest Octets":                {"206cf51e2d49fa50f4ae1fb0012b2b2b2b2b2b2b"},
	"10.5.2.37o\x00UTRAN FDD/TDD Description struct": {"9ea3c5a93f4a1e95c3f600"},
	// TS 44.018 V19.0.0 §10.5.2.37h: an 8-bit header, one Non-GSM message
	// with 17 container octets ending at bit 152, then the stop record.
	"10.5.2.37h\x00SI 18 Rest Octets": {"0031" + strings.Repeat("00", 17) + "20"},
	"10.5.2.37i\x00SI 20 Rest Octets": {"0031" + strings.Repeat("00", 17) + "20"},
}

// sliceFields lists settable lists of struct elements.
func sliceFields(v reflect.Value, depth int, out *[]reflect.Value) {
	if depth > 30 || !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			sliceFields(v.Elem(), depth+1, out)
		}
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[runtime.WireInfo]() || v.Type() == reflect.TypeFor[runtime.BitString]() {
			return
		}
		for i := range v.NumField() {
			sliceFields(v.Field(i), depth+1, out)
		}
	case reflect.Slice:
		if v.CanSet() && v.Type().Elem().Kind() == reflect.Struct {
			*out = append(*out, v)
		}
		for i := range v.Len() {
			sliceFields(v.Index(i), depth+1, out)
		}
	}
}

// fallbackSite is a generated fallback (fields Alternative, Known and
// Ignored) with the length field of the group that encloses it.
type fallbackSite struct{ fallback, length reflect.Value }

// fallbackFields collects settable fallbacks whose enclosing struct has an
// unsigned field named ...Length, the printed val(<length>) + 1 bound.
func fallbackFields(v reflect.Value, depth int, out *[]fallbackSite) {
	if depth > 30 || !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer:
		if !v.IsNil() {
			fallbackFields(v.Elem(), depth+1, out)
		}
	case reflect.Struct:
		if v.Type() == reflect.TypeFor[runtime.WireInfo]() || v.Type() == reflect.TypeFor[runtime.BitString]() {
			return
		}
		var length reflect.Value
		for i := range v.NumField() {
			if f := v.Field(i); strings.HasSuffix(v.Type().Field(i).Name, "Length") && f.CanSet() && f.Kind() >= reflect.Uint8 && f.Kind() <= reflect.Uint64 {
				length = f
			}
		}
		for i := range v.NumField() {
			f := v.Field(i)
			if length.IsValid() && f.Kind() == reflect.Struct && f.CanSet() {
				if g := f.FieldByName("Ignored"); g.IsValid() && g.Type() == reflect.TypeFor[*runtime.BitString]() && f.FieldByName("Known").IsValid() && f.FieldByName("Alternative").IsValid() {
					*out = append(*out, fallbackSite{f, length})
				}
			}
			fallbackFields(f, depth+1, out)
		}
	case reflect.Slice:
		for i := range v.Len() {
			fallbackFields(v.Index(i), depth+1, out)
		}
	}
}

func canonicalSeedCount(d runtime.Descriptor) int {
	return len(canonicalSeeds[d.Clause+"\x00"+d.Name])
}

// selectValues keeps every seed (they come last), the values with the
// longest plain encodings, which reach maxima and fixed sizes, the shortest,
// which leave a requested extent unfilled, and an even spread of the rest.
func selectValues(d runtime.Descriptor, values []any, seeds int) []any {
	const keepLongest, keepShortest, keepLists, keepSpread = 16, 4, 8, 16
	seeds = min(seeds, len(values))
	rest := values[:len(values)-seeds]
	kept := append([]any(nil), values[len(values)-seeds:]...)
	if len(rest) <= keepLongest+keepShortest+keepLists+keepSpread {
		return append(kept, rest...)
	}
	size := func(v any) int {
		out, err := d.Encode(v)
		if err != nil {
			return -1
		}
		return len(out)
	}
	order := make([]int, len(rest))
	sizes := make([]int, len(rest))
	for i, v := range rest {
		order[i], sizes[i] = i, size(v)
	}
	sort.SliceStable(order, func(a, b int) bool { return sizes[order[a]] > sizes[order[b]] })
	taken := map[int]bool{}
	var shortest []int
	for k := len(order) - 1; k >= keepLongest && len(shortest) < keepShortest; k-- {
		if sizes[order[k]] >= 0 {
			shortest = append(shortest, order[k])
		}
	}
	// Values with the most non-empty lists grow past maxima and fixed sizes.
	lists := make([]int, len(rest))
	byLists := make([]int, len(rest))
	for i, v := range rest {
		var ls []reflect.Value
		sliceFields(deepCopy(v), 0, &ls)
		for _, l := range ls {
			if l.Len() > 0 {
				lists[i]++
			}
		}
		byLists[i] = i
	}
	sort.SliceStable(byLists, func(a, b int) bool { return lists[byLists[a]] > lists[byLists[b]] })
	picks := append(append([]int(nil), order[:keepLongest]...), shortest...)
	picks = append(picks, byLists[:keepLists]...)
	for _, i := range picks {
		if !taken[i] {
			taken[i] = true
			kept = append(kept, rest[i])
		}
	}
	for i := 0; i < len(rest) && len(kept) < seeds+keepLongest+keepShortest+keepLists+keepSpread; i += len(rest)/keepSpread + 1 {
		if !taken[i] {
			kept = append(kept, rest[i])
		}
	}
	return kept
}

// corpusDigest hashes the decoded values and the harvested nested values in
// descriptor and type order, as evidence that the searched corpus is fixed.
func corpusDigest(values [][]any, harvested map[reflect.Type][]reflect.Value) []byte {
	h := sha256.New()
	for index, list := range values {
		for _, v := range list {
			encoded, _ := json.Marshal(v)
			_, _ = fmt.Fprintf(h, "%d %T %s\n", index, v, encoded)
		}
	}
	types := make([]reflect.Type, 0, len(harvested))
	for typ := range harvested {
		types = append(types, typ)
	}
	sort.Slice(types, func(a, b int) bool { return types[a].String() < types[b].String() })
	for _, typ := range types {
		for _, v := range harvested[typ] {
			encoded, _ := json.Marshal(v.Interface())
			_, _ = fmt.Fprintf(h, "%s %s\n", typ, encoded)
		}
	}
	return h.Sum(nil)
}

// pendingAmbiguity pins a published grammar defect that awaits a decision,
// keyed by clause, name and entry point: the mismatch must stay exactly as
// pinned. It is empty. TS 44.018 V19.0.0 §10.5.2.37o, the last entry, was
// resolved by a source correction (gomaja/go-csn1#20).
var pendingAmbiguity = map[string]string{}
