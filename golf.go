package golf

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type any interface{}

type golf struct {
	shorts map[string]*golfOpt
	longs  map[string]*golfOpt
	all    []*golfOpt
}

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

type golfOpt struct {
	Short    string
	Long     string
	Name     string
	Required bool
	Default  any
	Type     optType
	Result   any
	IsSet    bool
	Help     string
}

func (g *golf) addOpt(resultPtr any, short, long, name, help string, defaultVal any, required bool, kind optType) {
	opt := golfOpt{
		Short:    short,
		Long:     long,
		Name:     name,
		Required: required,
		Default:  defaultVal,
		Type:     kind,
		Result:   resultPtr,
		IsSet:    false,
		Help:     help,
	}
	if short != "" {
		g.shorts[short] = &opt
	}
	if long != "" {
		g.longs[long] = &opt
	}
	g.all = append(g.all, &opt)
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

func (o *golfOpt) Parse(value string) error {
	switch o.Type {
	case optString:
		fallthrough
	case optBareString:
		if ptr, ok := o.Result.(*string); ok {
			*ptr = value
		} else {
			return fmt.Errorf("arg<%s> result ptr is not string", o.debugArg())
		}
		break
	case optInt:
		if ptr, ok := o.Result.(*int); ok {
			conv, err := strconv.Atoi(value)
			if err != nil {
				return fmt.Errorf("arg<%s> require int, got <%s>", o.debugArg(), value)
			}
			*ptr = conv
		} else {
			return fmt.Errorf("arg<%s> result ptr is not int", o.debugArg())
		}
		break
	case optBool:
		if ptr, ok := o.Result.(*bool); ok {
			if strings.HasPrefix(value, "-") {
				*ptr = true
			} else {
				str := strings.ToLower(value)
				if existInArray([]string{
					"0", "false", "f", "no", "n",
				}, str) {
					*ptr = false
				} else if existInArray([]string{
					"1", "true", "t", "yes", "y", "",
				}, str) {
					*ptr = true
				} else {
					return fmt.Errorf("arg<%s> expect bool, got %s", o.debugArg(), value)
				}
			}
		} else {
			return fmt.Errorf("arg<%s> result ptr is not bool", o.debugArg())
		}
		break
	case optFloat:
		if ptr, ok := o.Result.(*float64); ok {
			f, err := strconv.ParseFloat(value, 64)
			if err != nil {
				return fmt.Errorf("arg<%s> require float64, got <%s>", o.debugArg(), value)
			}
			*ptr = f
		} else {
			return fmt.Errorf("arg<%s> result ptr is not float", o.debugArg())
		}
		break
	case optArray:
		if ptr, ok := o.Result.(*[]string); ok {
			*ptr = append(*ptr, value)
		} else {
			return fmt.Errorf("arg<%s> result ptr is not []string", o.debugArg())
		}
	default:
		return fmt.Errorf("arg<%s> unimplemented type %d", o.debugArg(), o.Type)
	}
	o.IsSet = true
	return nil
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
	g.addOpt(&result, short, long, name, help, defaultVal, false, optString)
	return &result
}

func MustString(short, long, name, help string) *string {
	result := ""
	g.addOpt(&result, short, long, name, help, "", true, optString)
	return &result
}

func Int(short, long, name, help string, defaultVal int) *int {
	result := defaultVal
	g.addOpt(&result, short, long, name, help, result, false, optInt)
	return &result
}

func MustInt(short, long, name, help string) *int {
	result := 0
	g.addOpt(&result, short, long, name, help, result, true, optInt)
	return &result
}

func Bool(short, long, name, help string, defaultVal bool) *bool {
	result := defaultVal
	g.addOpt(&result, short, long, name, help, result, false, optBool)
	return &result
}

func MustBool(short, long, name, help string) *bool {
	result := false
	g.addOpt(&result, short, long, name, help, false, true, optBool)
	return &result
}

func Array(short, long, name, help string) *[]string {
	result := make([]string, 0)
	g.addOpt(&result, short, long, name, help, result, false, optArray)
	return &result
}

func BareArray(name, help string) *[]string {
	result := make([]string, 0)
	g.addOpt(&result, "", "", name, help, result, false, optBareArray)
	return &result
}
func BareString(name, help string) *string {
	result := ""
	g.addOpt(&result, "", "", name, help, result, true, optBareString)
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

func Parse(args []string) error {
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
			result, ok := opt.Result.(*[]string)
			if !ok {
				return fmt.Errorf("arg<%s> result ptr is not []string", opt.debugArg())
			}
			*result = bares
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

func ParseOSArgs() (bool, error) {
	help := Bool("h", "help", "", "Show this message", false)
	if err := Parse(os.Args[1:]); err != nil {
		return *help, err
	}
	return *help, nil
}
