package proteus

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestFixNameForTemplate(t *testing.T) {
	vals := []struct {
		input  string
		output string
	}{
		{"foo.bar", "fooDOTbar"},
		{"foo.bar.baz", "fooDOTbarDOTbaz"},
		{"fooDOTbar", "fooDOTDOTbar"},
	}
	for _, v := range vals {
		if fixNameForTemplate(v.input) != v.output {
			t.Error("Expected", v.output, "for", v.input, "got", fixNameForTemplate(v.input))
		}
	}
}

func Test_validateFunction(t *testing.T) {
	type args struct {
		funcType reflect.Type
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
		hasCtx  bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasCtx, err := validateFunction(tt.args.funcType)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFunction() error = %v, wantErr %v", err, tt.wantErr)
			}
			if hasCtx != tt.hasCtx {
				t.Errorf("validateFunction() hasCtx = %v, expected = %v", hasCtx, tt.hasCtx)
			}
		})
	}
}

func Test_buildParamMap(t *testing.T) {
	type args struct {
		prop       string
		paramCount int
		startPos   int
	}
	tests := []struct {
		name string
		args args
		want map[string]int
	}{
		{
			name: "simple",
			args: args{
				prop:       "a,b,c",
				paramCount: 3,
				startPos:   1,
			},
			want: map[string]int{
				"a": 1,
				"b": 2,
				"c": 3,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildNameOrderMap(tt.args.prop, tt.args.startPos)
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Error(diff)
			}
		})
	}
}

func Test_buildDummyParameters(t *testing.T) {
	type args struct {
		paramCount int
		startPos   int
	}
	tests := []struct {
		name string
		args args
		want map[string]int
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := buildDummyParameters(tt.args.paramCount, tt.args.startPos); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("buildDummyParameters() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_convertToPositionalParameters(t *testing.T) {
	type args struct {
		query    string
		paramMap map[string]int
		funcType reflect.Type
		pa       ParamAdapter
	}
	tests := []struct {
		name    string
		args    args
		want    queryHolder
		want1   []paramInfo
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	ctx := context.Background()
	for _, tt := range tests {
		got, got1, err := buildFixedQueryAndParamOrder(ctx, tt.args.query, tt.args.paramMap, tt.args.funcType, tt.args.pa)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. buildFixedQueryAndParamOrder() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. buildFixedQueryAndParamOrder() got = %v, want %v", tt.name, got, tt.want)
		}
		if !reflect.DeepEqual(got1, tt.want1) {
			t.Errorf("%q. buildFixedQueryAndParamOrder() got1 = %v, want %v", tt.name, got1, tt.want1)
		}
	}
}

func Test_joinFactory(t *testing.T) {
	type args struct {
		startPos     int
		paramAdapter ParamAdapter
	}
	tests := []struct {
		name string
		args args
		want func(int) string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		if got := joinFactory(tt.args.startPos, tt.args.paramAdapter); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%q. joinFactory() = %T, want %T", tt.name, got, tt.want)
		}
	}
}

func Test_fixNameForTemplate(t *testing.T) {
	type args struct {
		name string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		if got := fixNameForTemplate(tt.args.name); got != tt.want {
			t.Errorf("%q. fixNameForTemplate() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_addSlice(t *testing.T) {
	type args struct {
		sliceName string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		if got := addSlice(tt.args.sliceName); got != tt.want {
			t.Errorf("%q. addSlice() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func Test_validIdentifier(t *testing.T) {
	type args struct {
		curVar string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	ctx := context.Background()
	for _, tt := range tests {
		got, err := validIdentifier(ctx, tt.args.curVar)
		if (err != nil) != tt.wantErr {
			t.Errorf("%q. validIdentifier() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			continue
		}
		if got != tt.want {
			t.Errorf("%q. validIdentifier() = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestBuilderBadFunction(t *testing.T) {
	type bad struct {
		Insert func(ctx context.Context, e ContextExecutor, email string) error `proq:"insert into foo (email) values (:$1:)"`
	}

	var b bad
	err := Build(&b, Postgres)
	if err == nil {
		t.Error("should return error")
	}
}
