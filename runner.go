package proteus

import (
	"bytes"
	"errors"
	"reflect"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jonbodner/proteus/api"
	"github.com/jonbodner/proteus/mapper"
)

func getExecAndQArgs(args []reflect.Value, qps queryParams) (api.Executor, []interface{}, error) {
	//pull out that first input parameter as an Executor
	exec := args[0].Interface().(api.Executor)

	//walk through the rest of the input parameters and build a slice for args

	qArgs := make([]interface{}, len(qps))
	for k, v := range qps {
		//todo later: add support for slices as parameters
		val, err := mapper.Extract(args[v.posInParams].Interface(), strings.Split(v.name, "."))
		if err != nil {
			return nil, nil, err
		}
		qArgs[k] = val
	}

	return exec, qArgs, nil
}

func finalizeQuery(positionalQuery processedQuery, sliceQPs queryParams, args []reflect.Value) (string, []interface{}, error) {
	queryToRun := positionalQuery.simple
	sliceArgs := []interface{}{}
	var err error
	if positionalQuery.kind == templ {
		log.Debugf("Processing template query with sliceQPs %#v\n", sliceQPs)
		var b bytes.Buffer
		sliceMap := map[string]interface{}{}
		for _, v := range sliceQPs {
			var val interface{}
			val, err = mapper.Extract(args[v.posInParams].Interface(), strings.Split(v.name, "."))
			if err != nil {
				break
			}
			valV := reflect.ValueOf(val)
			sliceMap[v.name] = valV.Len()
			for i := 0; i < valV.Len(); i++ {
				sliceArgs = append(sliceArgs, valV.Index(i).Interface())
			}
		}
		if err == nil {
			err = positionalQuery.temp.Execute(&b, sliceMap)
		}
		if err == nil {
			queryToRun = b.String()
		}
	}
	return queryToRun, sliceArgs, err
}

func buildExec(funcType reflect.Type, qps queryParams, sliceQPs queryParams, positionalQuery processedQuery) (func(args []reflect.Value) []reflect.Value, error) {
	numOut := funcType.NumOut()
	return func(args []reflect.Value) []reflect.Value {
		exec, qArgs, err := getExecAndQArgs(args, qps)
		var result api.Result
		if err == nil {
			//call executor.Exec with query and parameters
			queryToRun, sliceArgs, err2 := finalizeQuery(positionalQuery, sliceQPs, args)
			if err2 == nil {
				qArgs = append(qArgs, sliceArgs...)
				log.Debugln("calling", queryToRun, "with params", qArgs)
				result, err = exec.Exec(queryToRun, qArgs...)
			}
			err = err2
		}

		//handle the 0,1,2 out parameter cases
		if numOut == 0 {
			return []reflect.Value{}
		}

		zero := reflect.ValueOf(int64(0))
		sType := funcType.Out(0)
		if numOut == 1 {
			if err != nil {
				return []reflect.Value{zero}
			}
			val, err := result.LastInsertId()
			if err != nil {
				return []reflect.Value{zero}
			}
			if val != 0 {
				return []reflect.Value{reflect.ValueOf(val).Convert(sType)}
			}
			val, err = result.RowsAffected()
			if err != nil {
				return []reflect.Value{zero}
			}
			return []reflect.Value{reflect.ValueOf(val).Convert(sType)}
		}
		if numOut == 2 {
			eType := funcType.Out(1)
			if err != nil {
				return []reflect.Value{zero, reflect.ValueOf(err).Convert(eType)}
			}
			val, err := result.LastInsertId()
			if err != nil {
				return []reflect.Value{zero, reflect.ValueOf(err).Convert(eType)}
			}
			if val != 0 {
				return []reflect.Value{reflect.ValueOf(val).Convert(sType), errZero}
			}
			val, err = result.RowsAffected()
			if err != nil {
				return []reflect.Value{zero, reflect.ValueOf(err).Convert(eType)}
			}
			return []reflect.Value{reflect.ValueOf(val).Convert(sType), errZero}
		}
		return []reflect.Value{zero, reflect.ValueOf(errors.New("Should never get here!"))}
	}, nil
}

func buildQuery(funcType reflect.Type, qps queryParams, sliceQPs queryParams, positionalQuery processedQuery) (func(args []reflect.Value) []reflect.Value, error) {
	numOut := funcType.NumOut()
	var builder mapper.Builder
	var err error
	if numOut > 0 {
		builder, err = mapper.MakeBuilder(funcType.Out(0))
		if err != nil {
			return nil, err
		}
	}
	return func(args []reflect.Value) []reflect.Value {
		exec, qArgs, err := getExecAndQArgs(args, qps)

		var rows api.Rows
		//call executor.Query with query and parameters
		if err == nil {
			queryToRun, sliceArgs, err2 := finalizeQuery(positionalQuery, sliceQPs, args)
			if err2 == nil {
				qArgs = append(qArgs, sliceArgs...)
				log.Debugln("calling", queryToRun, "with params", qArgs)
				rows, err = exec.Query(queryToRun, qArgs...)
			}
			err = err2
		}

		//handle the 0,1,2 out parameter cases
		if numOut == 0 {
			return []reflect.Value{}
		}

		sType := funcType.Out(0)
		zero := reflect.Zero(sType)
		if numOut == 1 {
			if err != nil {
				return []reflect.Value{zero}
			}
			// handle mapping
			val, err := handleMapping(sType, rows, builder)
			if err != nil {
				return []reflect.Value{zero}
			}
			return []reflect.Value{reflect.ValueOf(val).Convert(sType)}
		}
		if numOut == 2 {
			eType := funcType.Out(1)
			if err != nil {
				return []reflect.Value{zero, reflect.ValueOf(err).Convert(eType)}
			}
			// handle mapping
			val, err := handleMapping(sType, rows, builder)
			var eVal reflect.Value
			if err == nil {
				eVal = errZero
			} else {
				eVal = reflect.ValueOf(err).Convert(eType)
			}
			return []reflect.Value{reflect.ValueOf(val).Convert(sType), eVal}
		}
		return []reflect.Value{zero, reflect.ValueOf(errors.New("Should never get here!"))}
	}, nil
}

var errZero = reflect.Zero(reflect.TypeOf((*error)(nil)).Elem())

func handleMapping(sType reflect.Type, rows api.Rows, builder mapper.Builder) (interface{}, error) {
	var val interface{}
	var err error
	if sType.Kind() == reflect.Slice {
		s := reflect.MakeSlice(sType, 0, 0)
		var result interface{}
		for {
			result, err = mapper.Map(rows, builder)
			if result == nil {
				break
			}
			s = reflect.Append(s, reflect.ValueOf(result))
		}
		val = s.Interface()
	} else {
		val, err = mapper.Map(rows, builder)
	}
	rows.Close()
	return val, err
}
