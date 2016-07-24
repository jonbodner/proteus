package proteus

import (
	"bytes"
	"errors"
	"fmt"
	"go/scanner"
	"go/token"
	"reflect"
	"strings"
	"text/template"

	log "github.com/Sirupsen/logrus"
	"github.com/jonbodner/proteus/api"
	"github.com/jonbodner/proteus/mapper"
)

func validateFunction(funcType reflect.Type, isExec bool) error {
	//first parameter is Executor
	if funcType.NumIn() == 0 {
		return errors.New("need to supply an Executor parameter")
	}
	exType := reflect.TypeOf((*api.Executor)(nil)).Elem()
	if !funcType.In(0).Implements(exType) {
		return errors.New("first parameter must be of type api.Executor")
	}
	//no in parameter can be a channel
	for i := 1; i < funcType.NumIn(); i++ {
		if funcType.In(i).Kind() == reflect.Chan {
			return errors.New("no input parameter can be a channel")
		}
	}

	//has 0, 1, or 2 return values
	if funcType.NumOut() > 2 {
		return errors.New("must return 0, 1, or 2 values")
	}

	//if 2 return values, second is error
	if funcType.NumOut() == 2 {
		errType := reflect.TypeOf((*error)(nil)).Elem()
		if !funcType.Out(1).Implements(errType) {
			return errors.New("2nd output parameter must be of type error")
		}
	}

	//if 1 or 2, 1st param is not a channel (handle map, I guess)
	if funcType.NumOut() > 0 {
		if funcType.Out(0).Kind() == reflect.Chan {
			return errors.New("1st output parameter cannot be a channel")
		}
		if isExec && funcType.Out(0).Kind() != reflect.Int64 {
			return errors.New("the 1st output parameter of an Exec must be int64")
		}
	}
	return nil
}

func buildParamMap(prop string) map[string]int {
	queryParams := strings.Split(prop, ",")
	m := map[string]int{}
	for k, v := range queryParams {
		m[v] = k + 1
	}
	return m
}

type kind int

const (
	simple kind = iota
	templ
)

type processedQuery struct {
	kind   kind
	simple string
	temp   *template.Template
}

func convertToPositionalParameters(query string, paramMap map[string]int, funcType reflect.Type, pa api.ParamAdapter) (processedQuery, queryParams, error) {
	var out bytes.Buffer

	var qps queryParams

	// escapes:
	// \ (any character), that character literally (meant for escaping : and \)
	// ending on a single \ means the \ is ignored
	inEscape := false
	inVar := false
	curVar := []rune{}
	queryKind := simple
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
					return processedQuery{}, nil, fmt.Errorf("empty variable declaration at position %d", k)
				}
				curVarS := string(curVar)
				id, err := validIdentifier(curVarS)
				if err != nil {
					//error, identifier must be valid go identifier with . for path
					return processedQuery{}, nil, err
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
				if paramPos, ok := paramMap[paramName]; ok {
					//if the path has more than one part, make sure that the type of the function parameter is map or struct
					paramType := funcType.In(paramPos)
					if len(path) > 1 {
						switch paramType.Kind() {
						case reflect.Map, reflect.Struct:
							//do nothing
						default:
							return processedQuery{}, nil, fmt.Errorf("query Parameter %s has a path, but the incoming parameter is not a map or a struct", paramName)
						}
					}
					pathType, err := mapper.ExtractType(paramType, path)
					if err != nil {
						return processedQuery{}, nil, err
					}
					out.WriteString(addSlice(id))
					isSlice := false
					if pathType != nil && pathType.Kind() == reflect.Slice {
						queryKind = templ
						isSlice = true
					}
					qps = append(qps, paramInfo{id, paramPos, isSlice})
				} else {
					return processedQuery{}, nil, fmt.Errorf("query Parameter %s cannot be found in the incoming parameters", paramName)
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
		return processedQuery{}, nil, fmt.Errorf("missing a closing : somewhere: %s", query)
	}

	queryString := out.String()
	temp, err := template.New("query").Funcs(template.FuncMap{"join": joinFactory(1, pa)}).Parse(queryString)
	if err != nil {
		return processedQuery{}, nil, err
	}
	if queryKind == simple {
		//can evaluate the template now, with 1 for the length for each item
		m := map[string]interface{}{}
		for _, v := range qps {
			m[fixNameForTemplate(v.name)] = 1
		}
		var b bytes.Buffer
		err = temp.Execute(&b, m)
		if err != nil {
			return processedQuery{}, nil, err
		}
		queryString = b.String()
		temp = nil
	}
	return processedQuery{kind: queryKind, simple: queryString, temp: temp}, qps, nil
}

type paramInfo struct {
	name        string
	posInParams int
	isSlice     bool
}

// key == position in query
// value == name to evaluate & position in function in parameters
type queryParams []paramInfo

const (
	sliceTemplate = `{{.%s | join}}`
)

func joinFactory(startPos int, paramAdapter api.ParamAdapter) func(int) string {
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
	return strings.Replace(name, ".", "DOT", -1)
}

func addSlice(sliceName string) string {
	return fmt.Sprintf(sliceTemplate, fixNameForTemplate(sliceName))
}

func validIdentifier(curVar string) (string, error) {
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
loop:
	for {
		pos, tok, lit := s.Scan()
		log.Debugf("%s\t%s\t%q\n", fset.Position(pos), tok, lit)
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
			if !first && !lastPeriod {
				return "", fmt.Errorf(". missing between parts of an identifier: %s", curVar)
			}
			first = false
			lastPeriod = false
			identifier += lit
		case token.PERIOD:
			if first || lastPeriod {
				return "", fmt.Errorf("identifier cannot start with . or have two . in a row: %s", curVar)
			}
			lastPeriod = true
			identifier += "."
		default:
			return "", fmt.Errorf("invalid character found in identifier: %s", curVar)
		}
	}
	return identifier, nil
}
