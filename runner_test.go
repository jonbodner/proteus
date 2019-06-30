package proteus

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/jonbodner/proteus/logger"
	"github.com/jonbodner/proteus/mapper"
)

func Test_getQArgs(t *testing.T) {
	type args struct {
		args []reflect.Value
		qps  []paramInfo
	}
	tests := []struct {
		name    string
		args    args
		want    []interface{}
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	for _, tt := range tests {
		got, err := buildQueryArgs(c, tt.args.args, tt.args.qps)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. buildQueryArgs() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. buildQueryArgs() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_buildExec(t *testing.T) {
	type args struct {
		funcType        reflect.Type
		qps             []paramInfo
		positionalQuery queryHolder
	}
	tests := []struct {
		name    string
		args    args
		want    func(args []reflect.Value) []reflect.Value
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	for _, tt := range tests {
		got := makeExecutorImplementation(c, tt.args.funcType, tt.args.positionalQuery, tt.args.qps)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. makeExecutorImplementation() = %T, want %T", tt.name, got, tt.want)
		}
	}
}

func Test_buildQuery(t *testing.T) {
	type args struct {
		funcType        reflect.Type
		qps             []paramInfo
		positionalQuery queryHolder
	}
	tests := []struct {
		name    string
		args    args
		want    func(args []reflect.Value) []reflect.Value
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	for _, tt := range tests {
		got, err := makeQuerierImplementation(c, tt.args.funcType, tt.args.positionalQuery, tt.args.qps)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. makeQuerierImplementation() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. makeQuerierImplementation() = %T, want %T", tt.name, got, tt.want)
		}
	}
}

func Test_handleMapping(t *testing.T) {
	type args struct {
		sType   reflect.Type
		rows    *sql.Rows
		builder mapper.Builder
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
	// TODO: Add test cases.
	}
	c := logger.WithLevel(context.Background(), logger.DEBUG)
	for _, tt := range tests {
		got, err := handleMapping(c, tt.args.sType, tt.args.rows, tt.args.builder)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. handleMapping() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. handleMapping() = %v, want %v", tt.name, got, tt.want)
		}
	}
}
