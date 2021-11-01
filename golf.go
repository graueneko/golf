package golf

import (
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type any interface{}

type golf struct {
	shorts map[string]*golfOpt
	longs  map[string]*golfOpt
	all    []*golfOpt
}

var (
	tagPartReg = regexp.MustCompile("(([^;:]+):\\s*'([^']*)'\\s*;?)|(([^:;'\"]+);)|(([^;:]+):([^;]+);?)|([^:;'\"]+)")
	tagFullReg = regexp.MustCompile("((([^;:]+):\\s*'([^']*)'\\s*;?)|(([^:;'\"]+);)|(([^;:]+):([^;]+);?)|([^:;'\"]+)\\s*)+")
)

var g golf

func init() {
	Reset()
}

type optType int

const (
	optInt optType = iota
	optBool
	optFloat
	optString
	optArray
	optBareArray
	optBareString
)

type resultSetter struct {
	resultPtr *reflect.Value
}

func (p resultSetter) SetValue(v any) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()
	p.resultPtr.Set(reflect.ValueOf(v).Convert(p.resultPtr.Type()))
	return true
}

func (p resultSetter) AddValue(v any) (ok bool) {
	defer func() {
		if r := recover(); r != nil {
			ok = false
		}
	}()

	p.resultPtr.Set(reflect.Append(*p.resultPtr, reflect.ValueOf(v)))
	return true
}

type golfOpt struct {
	Short        string
	Long         string
	Name         string
	Required     bool
	Default      any
	Type         optType
	ResultSetter resultSetter
	IsSet        bool
	Help         string
}

func (g *golf) addOpt(rs resultSetter, short, long, name, help string, defaultVal any, required bool, kind optType) {
	opt := golfOpt{
		Short:        short,
		Long:         long,
		Name:         name,
		Required:     required,
		Default:      defaultVal,
		Type:         kind,
		ResultSetter: rs,
		IsSet:        false,
		Help:         help,
	}
	if short != "" {
		g.shorts[short] = &opt
	}
	if long != "" {
		g.longs[long] = &opt
	}
	g.all = append(g.all, &opt)
}

func (g *golf) addOptTag(rs resultSetter, gtag string) error {
	if !tagFullReg.MatchString(gtag) {
		return fmt.Errorf("invalid golf tag format")
	}
	t, ok := map[reflect.Kind]optType{
		reflect.Bool:      optBool,
		reflect.Int:       optInt,
		reflect.Float32:   optBool,
		reflect.Array:     optArray,
		reflect.Interface: optArray,
		reflect.Slice:     optArray,
		reflect.String:    optString,
	}[rs.resultPtr.Kind()]
	if !ok {
		return fmt.Errorf("unsupported type %s", rs.resultPtr.Kind())
	}

	opt := golfOpt{
		Short:        "",
		Long:         "",
		Name:         "",
		Required:     false,
		Default:      nil,
		Type:         t,
		ResultSetter: rs,
		IsSet:        false,
		Help:         "",
	}
	matches := tagPartReg.FindAllStringSubmatch(gtag, -1)
	for _, m := range matches {
		if err := opt.fillByTag(m); err != nil {
			return fmt.Errorf("parse tag [%s] failed: %v", m[0], err)
		}
	}
	if opt.Short != "" {
		g.shorts[opt.Short] = &opt
	}
	if opt.Long != "" {
		g.longs[opt.Long] = &opt
	}
	g.all = append(g.all, &opt)

	return nil
}

func (o golfOpt) debugArg() string {
	result := ""
	if o.Short != "" {
		result = "-" + o.Short
	}

	if o.Long != "" {
		if result != "" {
			result += "/"
		}
		result += "--" + o.Long
	}
	return result
}

func (o golfOpt) debugValue() string {
	if o.Name != "" {
		return fmt.Sprintf("%s", o.Name)
	}
	switch o.Type {
	case optInt:
		return "int"
	case optString:
		return "string"
	case optBool:
		return "true/false"
	case optFloat:
		return "float"
	case optArray:
		return "array"
	default:
		return "<unknown>"
	}
}

func (o golfOpt) debugArgValue() string {
	arg := o.debugArg()
	val := o.debugValue()
	if arg == "" {
		return val
	}
	if val == "" {
		return arg
	}
	return arg + " " + val
}

func (o golfOpt) debugHelp() string {
	if o.Required {
		return "(required)"
	} else {
		return fmt.Sprintf("(default: \"%v\")", o.Default)
	}
}

func (o golfOpt) Usage() string {
	return fmt.Sprintf("  %-20s: %s %s", o.debugArgValue(), o.Help, o.debugHelp())
}

func existInArray(arr []string, val string) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}
	return false
}

func str2bool(value string) (bool, error) {
	str := strings.ToLower(value)
	if existInArray([]string{
		"0", "false", "f", "no", "n",
	}, str) {
		return false, nil
	} else if existInArray([]string{
		"1", "true", "t", "yes", "y", "",
	}, str) {
		return true, nil
	} else {
		return false, fmt.Errorf("expect bool, got %s", value)
	}
}

func (o *golfOpt) Parse(value string) (err error) {
	switch o.Type {
	case optString:
		fallthrough
	case optBareString:
		if ok := o.ResultSetter.SetValue(value); !ok {
			return fmt.Errorf("arg<%s> result ptr is not string", o.debugArg())
		}
		break
	case optInt:
		conv, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("arg<%s> require int, got <%s>", o.debugArg(), value)
		}
		if ok := o.ResultSetter.SetValue(conv); !ok {
			return fmt.Errorf("arg<%s> result ptr is not int", o.debugArg())
		}
		break
	case optBool:
		result := false
		if strings.HasPrefix(value, "-") {
			result = true
		} else if result, err = str2bool(value); err != nil {
			return fmt.Errorf("arg<%s> %v", o.debugArg(), err.Error())
		}
		if ok := o.ResultSetter.SetValue(result); !ok {
			return fmt.Errorf("arg<%s> result ptr is not bool", o.debugArg())
		}
		break
	case optFloat:
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("arg<%s> require float64, got <%s>", o.debugArg(), value)
		}

		if ok := o.ResultSetter.SetValue(f); !ok {
			return fmt.Errorf("arg<%s> result ptr is not float", o.debugArg())
		}
		break
	case optArray:
		if ok := o.ResultSetter.AddValue(value); !ok {
			return fmt.Errorf("arg<%s> result ptr is not []string", o.debugArg())
		}
	default:
		return fmt.Errorf("arg<%s> unimplemented type %d", o.debugArg(), o.Type)
	}
	o.IsSet = true
	return nil
}

func (o *golfOpt) fillByTag(m []string) error {
	if len(m) != 10 {
		return fmt.Errorf("invalid length %d", len(m))
	}
	key := ""
	val := ""
	if m[2] != "" {
		// <key>: '<value>'[;]
		// value in quote
		key = strings.TrimSpace(m[2])
		val = m[3]
	} else if m[5] != "" {
		// <key>;
		// key only, value empty
		key = strings.TrimSpace(m[5])
	} else if m[7] != "" {
		// <key>: <value>[;]
		// key and value, without quote
		key = strings.TrimSpace(m[7])
		val = strings.TrimSpace(m[8])
	} else if m[9] != "" {
		// <key>
		// key without value and colon
		key = strings.TrimSpace(m[9])
	}

	switch strings.ToLower(key) {
	case "s":
		fallthrough
	case "short":
		if val == "" {
			return fmt.Errorf("<short> cannot be empty")
		}
		o.Short = val
		break
	case "l":
		fallthrough
	case "long":
		if val == "" {
			return fmt.Errorf("<long> cannot be empty")
		}
		o.Long = val
		break
	case "n":
		fallthrough
	case "name":
		if val == "" {
			return fmt.Errorf("<name> cannot be empty")
		}
		o.Name = val
		break
	case "d":
		fallthrough
	case "default":
		defaultVal, err := str2optType(o.Type, val)
		if err != nil {
			return fmt.Errorf("invalid <default> val: %s", val)
		}
		o.Default = defaultVal
		break
	case "h":
		fallthrough
	case "help":
		o.Help = val
		break
	case "r":
		fallthrough
	case "required":
		req, err := str2bool(val)
		if err != nil {
			return fmt.Errorf("invalid <required> val: %s", val)
		}
		o.Required = req
	default:
		return fmt.Errorf("invalid tag option <%s>", key)
	}
	return nil
}

func str2optType(t optType, s string) (any, error) {
	switch t {
	case optInt:
		if i, err := strconv.Atoi(s); err != nil {
			return nil, fmt.Errorf("<%s> not valid int", s)
		} else {
			return i, nil
		}
	case optBool:
		if b, err := str2bool(s); err != nil {
			return false, err
		} else {
			return b, nil
		}
	case optString:
		return s, nil
	case optFloat:
		if f, err := strconv.ParseFloat(s, 64); err != nil {
			return nil, fmt.Errorf("<%s> not valid float", s)
		} else {
			return f, nil
		}
	default:
		return nil, fmt.Errorf("cannot parse type %v from string", t)
	}
}

func parseKV(k, v string) error {
	if strings.HasPrefix(k, "--") {
		key := strings.TrimPrefix(k, "--")
		if opt, ok := g.longs[key]; ok {
			if err := opt.Parse(v); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("unrecognized arg %v", k)
		}
	} else {
		key := strings.TrimPrefix(k, "-")
		if opt, ok := g.shorts[key]; ok {
			if err := opt.Parse(v); err != nil {
				return err
			}
		}
	}
	return nil
}

func String(short, long, name, help string, defaultVal string) *string {
	result := defaultVal
	resultValue := reflect.ValueOf(&result).Elem()
	setter := resultSetter{resultPtr: &resultValue}
	g.addOpt(setter, short, long, name, help, defaultVal, false, optString)
	return &result
}

func MustString(short, long, name, help string) *string {
	result := ""
	resultValue := reflect.ValueOf(&result).Elem()
	setter := resultSetter{resultPtr: &resultValue}
	g.addOpt(setter, short, long, name, help, "", true, optString)
	return &result
}

func Int(short, long, name, help string, defaultVal int) *int {
	result := defaultVal
	resultValue := reflect.ValueOf(&result).Elem()
	setter := resultSetter{resultPtr: &resultValue}
	g.addOpt(setter, short, long, name, help, result, false, optInt)
	return &result
}

func MustInt(short, long, name, help string) *int {
	result := 0
	resultValue := reflect.ValueOf(&result).Elem()
	setter := resultSetter{resultPtr: &resultValue}
	g.addOpt(setter, short, long, name, help, result, true, optInt)
	return &result
}

func Bool(short, long, name, help string, defaultVal bool) *bool {
	result := defaultVal
	resultValue := reflect.ValueOf(&result).Elem()
	setter := resultSetter{resultPtr: &resultValue}
	g.addOpt(setter, short, long, name, help, result, false, optBool)
	return &result
}

func MustBool(short, long, name, help string) *bool {
	result := false
	resultValue := reflect.ValueOf(&result).Elem()
	setter := resultSetter{resultPtr: &resultValue}
	g.addOpt(setter, short, long, name, help, false, true, optBool)
	return &result
}

func Array(short, long, name, help string) *[]string {
	result := make([]string, 0)
	resultValue := reflect.ValueOf(&result).Elem()
	setter := resultSetter{resultPtr: &resultValue}
	g.addOpt(setter, short, long, name, help, result, false, optArray)
	return &result
}

func BareArray(name, help string) *[]string {
	result := make([]string, 0)
	resultValue := reflect.ValueOf(&result).Elem()
	setter := resultSetter{resultPtr: &resultValue}
	g.addOpt(setter, "", "", name, help, result, false, optBareArray)
	return &result
}
func BareString(name, help string) *string {
	result := ""
	resultValue := reflect.ValueOf(&result).Elem()
	setter := resultSetter{resultPtr: &resultValue}
	g.addOpt(setter, "", "", name, help, result, true, optBareString)
	return &result
}

func Reset() {
	g = golf{
		shorts: map[string]*golfOpt{},
		longs:  map[string]*golfOpt{},
		all:    make([]*golfOpt, 0),
	}
}

func Usage(executable string) string {
	builder := strings.Builder{}
	builder.WriteString("Usage:\n")
	builder.WriteString(fmt.Sprintf("  %s", executable))
	for _, opt := range g.all {
		if opt.Required {
			builder.WriteString(fmt.Sprintf(" %s", opt.debugArgValue()))
		} else {
			builder.WriteString(fmt.Sprintf(" [%s]", opt.debugArgValue()))
		}
	}
	builder.WriteString("\n\n")
	msg := make([]string, len(g.all))
	for i, opt := range g.all {
		msg[i] = opt.Usage()
	}
	builder.WriteString(fmt.Sprintf("Arguments:\n%s\n", strings.Join(msg, "\n")))
	return builder.String()
}

func Parse(args []string) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("golf parse panic: %v", reflect.TypeOf(r).Elem().Name())
		}
	}()
	bares := make([]string, 0)
	key := ""
	for _, entry := range args {
		if key == "" {
			if strings.HasPrefix(entry, "-") {
				if idx := strings.Index(entry, "="); idx != -1 {
					k, v := entry[:idx], entry[idx+1:]
					if err := parseKV(k, v); err != nil {
						return err
					}
					key = ""
				} else {
					key = entry
				}
			} else {
				bares = append(bares, entry)
			}
		} else if err := parseKV(key, entry); err != nil {
			return err
		} else {
			key = ""
		}
	}
	if key != "" {
		if strings.HasPrefix(key, "-") {
			if err := parseKV(key, ""); err != nil {
				return err
			}
		} else {
			bares = append(bares, key)
		}
	}

	for _, opt := range g.all {
		if opt.Type == optBareString {
			if len(bares) != 0 {
				if err := opt.Parse(bares[0]); err != nil {
					return err
				}
				bares = bares[1:]
			}
		}
	}
	for _, opt := range g.all {
		if opt.Type == optBareArray {
			if ok := opt.ResultSetter.SetValue(bares); !ok {
				return fmt.Errorf("arg<%s> result ptr is not []string", opt.debugArg())
			}
			opt.IsSet = true
			bares = make([]string, 0)
		}
	}

	for _, opt := range g.all {
		if opt.Required && !opt.IsSet {
			return fmt.Errorf("missing argument: %s %s", opt.debugArg(), opt.debugValue())
		}
	}

	return nil
}

func ParseStruct(args []string, v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("parse struct panic: %s", reflect.TypeOf(r).Elem().Name())
		}
	}()
	vpt := reflect.ValueOf(v)
	if vpt.Kind() != reflect.Ptr {
		return fmt.Errorf("parse struct expects pointer to struct")
	}
	vt := vpt.Elem().Type()
	vv := vpt.Elem()
	for i := 0; i < vt.NumField(); i++ {
		st := vt.Field(i)
		gtag := st.Tag.Get("golf")
		if gtag == "" {
			continue
		}
		sv := vv.Field(i)
		setter := resultSetter{resultPtr: &sv}

		if err := g.addOptTag(setter, gtag); err != nil {
			return fmt.Errorf("golf parse tag of [%s] failed: %v", st.Name, err)
		}
	}

	return Parse(args)
}

func ParseOSArgs() (bool, error) {
	help := Bool("h", "help", "", "Show this message", false)
	if err := Parse(os.Args[1:]); err != nil {
		return *help, err
	}
	return *help, nil
}
