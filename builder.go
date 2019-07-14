package proteus

import (
	"bytes"
	"context"
	"fmt"
	"go/scanner"
	"go/token"
	"reflect"
	"strings"
	"text/template"

	"database/sql/driver"

	"github.com/jonbodner/proteus/logger"
	"github.com/jonbodner/proteus/mapper"
)

func buildNameOrderMap(paramOrder string, startPos int) map[string]int {
	out := map[string]int{}
	params := strings.Split(paramOrder, ",")
	for k, v := range params {
		out[strings.TrimSpace(v)] = k + startPos
	}
	return out
}

func buildDummyParameters(paramCount int, startPos int) map[string]int {
	m := map[string]int{}
	for i := startPos; i < paramCount; i++ {
		m[fmt.Sprintf("$%d", i-startPos+1)] = i
	}
	return m
}

// template slice support
type queryHolder interface {
	finalize(c context.Context, args []reflect.Value) (string, error)
}

type simpleQueryHolder string

func (sq simpleQueryHolder) finalize(c context.Context, args []reflect.Value) (string, error) {
	return string(sq), nil
}

type templateQueryHolder struct {
	queryString string
	pa          ParamAdapter
	paramOrder  []paramInfo
}

func (tq templateQueryHolder) finalize(c context.Context, args []reflect.Value) (string, error) {
	return doFinalize(c, tq.queryString, tq.paramOrder, tq.pa, args)
}

var (
	valueType = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
)

type posType interface {
	In(i int) reflect.Type
}

func buildFixedQueryAndParamOrder(c context.Context, query string, nameOrderMap map[string]int, funcType posType, pa ParamAdapter) (queryHolder, []paramInfo, error) {
	var out bytes.Buffer

	var paramOrder []paramInfo

	// escapes:
	// \ (any character), that character literally (meant for escaping : and \)
	// ending on a single \ means the \ is ignored
	inEscape := false
	inVar := false
	curVar := []rune{}
	hasSlice := false
	for k, v := range query {
		if inEscape {
			out.WriteRune(v)
			inEscape = false
			continue
		}
		switch v {
		case '\\':
			inEscape = true
		case ':':
			if inVar {
				if len(curVar) == 0 {
					//error! must have a something
					return nil, nil, fmt.Errorf("empty variable declaration at position %d", k)
				}
				curVarS := string(curVar)
				id, err := validIdentifier(c, curVarS)
				if err != nil {
					//error, identifier must be valid go identifier with . for path
					return nil, nil, err
				}
				//it's a valid identifier, but now we need to know if it's a slice or a scalar.
				//all we have is the name, not the mapping of the name to the position in the in parameters for the function.
				//so we need to do that search now, using the information in the struct tag prop.
				//mapper.ExtractType can tell us the kind of what we're expecting
				//if it's a scalar, then we use pa to write out the correct symbol for this db type and increment pos.
				//if it's a slice, then we put in the slice template syntax instead.

				//get just the first part of the name, before any .
				path := strings.SplitN(id, ".", 2)
				paramName := path[0]
				if paramPos, ok := nameOrderMap[paramName]; ok {
					//if the path has more than one part, make sure that the type of the function parameter is map or struct
					paramType := funcType.In(paramPos)
					if len(path) > 1 {
						switch paramType.Kind() {
						case reflect.Map, reflect.Struct:
							//do nothing
						default:
							return nil, nil, fmt.Errorf("query Parameter %s has a path, but the incoming parameter is not a map or a struct", paramName)
						}
					}
					pathType, err := mapper.ExtractType(c, paramType, path)
					if err != nil {
						return nil, nil, err
					}
					out.WriteString(addSlice(id))
					isSlice := false
					//special case -- slice of bytes is never expanded out into a comma-separated list
					if pathType != nil && pathType.Kind() == reflect.Slice && !pathType.Implements(valueType) && pathType.Elem().Kind() != reflect.Uint8 {
						hasSlice = true
						isSlice = true
					}
					paramOrder = append(paramOrder, paramInfo{id, paramPos, isSlice})
				} else {
					return nil, nil, fmt.Errorf("query Parameter %s cannot be found in the incoming parameters", paramName)
				}

				inVar = false
				curVar = []rune{}
			} else {
				inVar = true
			}
		default:
			if inVar {
				curVar = append(curVar, v)
			} else {
				out.WriteRune(v)
			}
		}
	}
	if inVar {
		return nil, nil, fmt.Errorf("missing a closing : somewhere: %s", query)
	}

	queryString := out.String()

	if !hasSlice {
		//no slices, so last param is never going to be referenced in doFinalize
		queryString, err := doFinalize(c, queryString, paramOrder, pa, nil)
		if err != nil {
			return nil, nil, err
		}
		return simpleQueryHolder(queryString), paramOrder, nil
	}
	return templateQueryHolder{queryString: queryString, pa: pa, paramOrder: paramOrder}, paramOrder, nil
}

func doFinalize(c context.Context, queryString string, paramOrder []paramInfo, pa ParamAdapter, args []reflect.Value) (string, error) {
	temp, err := template.New("query").Funcs(template.FuncMap{"join": joinFactory(1, pa)}).Parse(queryString)
	if err != nil {
		return "", err
	}

	//can evaluate the template now, with 1 for the length for each item
	sliceMap := map[string]interface{}{}
	for _, v := range paramOrder {
		if v.isSlice {
			var val interface{}
			val, err = mapper.Extract(c, args[v.posInParams].Interface(), strings.Split(v.name, "."))
			if err != nil {
				break
			}
			valV := reflect.ValueOf(val)
			sliceMap[fixNameForTemplate(v.name)] = valV.Len()
		} else {
			sliceMap[fixNameForTemplate(v.name)] = 1
		}
	}
	var b bytes.Buffer
	err = temp.Execute(&b, sliceMap)
	if err != nil {
		return "", err
	}
	return b.String(), err
}

type paramInfo struct {
	name        string
	posInParams int
	isSlice     bool
}

const (
	sliceTemplate = `{{.%s | join}}`
)

func joinFactory(startPos int, paramAdapter ParamAdapter) func(int) string {
	return func(total int) string {
		var b bytes.Buffer
		for i := 0; i < total; i++ {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(paramAdapter(startPos + i))
		}
		startPos += total
		return b.String()
	}
}

func fixNameForTemplate(name string) string {
	//need to make sure that foo.bar and fooDOTbar don't collide, however unlikely
	name = strings.Replace(name, "DOT", "DOTDOT", -1)
	name = strings.Replace(name, ".", "DOT", -1)
	name = strings.Replace(name, "DOLLAR", "DOLLARDOLLAR", -1)
	name = strings.Replace(name, "$", "DOLLAR", -1)
	return name
}

func addSlice(sliceName string) string {
	return fmt.Sprintf(sliceTemplate, fixNameForTemplate(sliceName))
}

func validIdentifier(c context.Context, curVar string) (string, error) {
	if strings.Contains(curVar, ";") {
		return "", fmt.Errorf("; is not allowed in an identifier: %s", curVar)
	}
	curVarB := []byte(curVar)

	var s scanner.Scanner
	fset := token.NewFileSet()                          // positions are relative to fset
	file := fset.AddFile("", fset.Base(), len(curVarB)) // register input "file"
	s.Init(file, curVarB, nil, scanner.Mode(0))

	lastPeriod := false
	first := true
	identifier := ""
	dollar := false
	lastFloat := false
loop:
	for {
		pos, tok, lit := s.Scan()
		logger.Log(c, logger.DEBUG, fmt.Sprintf("%s\t%s\t%q\n", fset.Position(pos), tok, lit))
		switch tok {
		case token.EOF:
			if first || lastPeriod {
				return "", fmt.Errorf("identifiers cannot be empty or end with a .: %s", curVar)
			}
			break loop
		case token.SEMICOLON:
			//happens with auto-insert from scanner
			//any explicit semicolons are illegal and handled earlier
			continue
		case token.IDENT:
			if !first && !lastPeriod && !lastFloat {
				return "", fmt.Errorf(". missing between parts of an identifier: %s", curVar)
			}
			first = false
			lastPeriod = false
			lastFloat = false
			identifier += lit
		case token.PERIOD:
			if first || lastPeriod {
				return "", fmt.Errorf("identifier cannot start with . or have two . in a row: %s", curVar)
			}
			lastPeriod = true
			identifier += "."
		case token.ILLEGAL:
			//special case to support $N notation, only valid for first part
			if lit == "$" && first {
				identifier = "$"
				dollar = true
				first = false
				continue
			}
			return "", fmt.Errorf("invalid character found in identifier: %s", curVar)
		case token.INT:
			if !dollar {
				return "", fmt.Errorf("invalid character found in identifier: %s", curVar)
			}
			identifier += lit
			dollar = false
		case token.FLOAT:
			//this is weird. If we have $1.NAME, it will think that there's a FLOAT token with value 1.
			//due to float support for exponents, if we have an E after the decimal point, the FLOAT token
			//will include the E and any subsequent digits. Obviously, only valid for $ notation
			if !dollar {
				return "", fmt.Errorf("invalid character found in identifier: %s", curVar)
			}
			identifier += lit
			dollar = false
			lastFloat = true
			if lit[len(lit)-1] == '.' {
				lastPeriod = true
			}
		default:
			return "", fmt.Errorf("invalid character found in identifier: %s", curVar)
		}
	}
	return identifier, nil
}
