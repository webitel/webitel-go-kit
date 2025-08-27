package filters

import (
	"reflect"
	"testing"

	"github.com/google/cel-go/cel"
	stubs "github.com/webitel/webitel-go-kit/pkg/filters/test_stubs/gen"
	"google.golang.org/genproto/googleapis/api/expr/v1alpha1"
)

func Test_extractIdentifier(t *testing.T) {
	type args struct {
		expr *expr.Expr
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{

		// "id"
		{
			name: "Singe identifier",
			args: args{
				expr: &expr.Expr{
					ExprKind: &expr.Expr_IdentExpr{
						IdentExpr: &expr.Expr_Ident{Name: "id"},
					},
				},
			},
			want:    "id",
			wantErr: false,
		},
		// "case.id"
		{
			name: "Nested identifier",
			args: args{
				expr: &expr.Expr{
					ExprKind: &expr.Expr_SelectExpr{
						SelectExpr: &expr.Expr_Select{
							Operand: &expr.Expr{
								ExprKind: &expr.Expr_IdentExpr{
									IdentExpr: &expr.Expr_Ident{Name: "case"},
								},
							},
							Field: "id",
						},
					},
				},
			},
			want:    "id",
			wantErr: false,
		},
		// "case.status_condition.id"
		{
			name: "Triple nested identifier",
			args: args{
				expr: &expr.Expr{
					ExprKind: &expr.Expr_SelectExpr{
						SelectExpr: &expr.Expr_Select{
							Operand: &expr.Expr{
								ExprKind: &expr.Expr_SelectExpr{
									SelectExpr: &expr.Expr_Select{
										Operand: &expr.Expr{
											ExprKind: &expr.Expr_IdentExpr{
												IdentExpr: &expr.Expr_Ident{Name: "case"},
											},
										},
										Field: "status_condition",
									},
								},
							},
							Field: "id",
						},
					},
				},
			},
			want:    "status_condition.id",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractIdentifier(tt.args.expr)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractIdentifier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("extractIdentifier() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseFilters(t *testing.T) {
	type args struct {
		env   *cel.Env
		query string
	}
	env, _ := cel.NewEnv(cel.Types(&stubs.TestingObject{}), cel.Variable("obj", cel.ObjectType("test_stubs.proto.v1.TestingObject")))
	tests := []struct {
		name    string
		args    args
		want    Filterer
		wantErr bool
	}{
		{
			name: "Single filter",
			args: args{
				env:   env,
				query: "obj.description == 'test'",
			},
			want: &Filter{
				Column:         "description",
				Value:          "test",
				ComparisonType: Equal,
			},
			wantErr: false,
		},
		{
			name: "Single filter with unknown field",
			args: args{
				env:   env,
				query: "obj.desc == 'test'",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Single filter with unknown operator",
			args: args{
				env:   env,
				query: "obj.created_by.id === 1",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Single filter more than",
			args: args{
				env:   env,
				query: "obj.created_by.id > 1",
			},
			want: &Filter{
				Column:         "created_by.id",
				Value:          int64(1),
				ComparisonType: GreaterThan,
			},
			wantErr: false,
		},
		{
			name: "Single filter less than",
			args: args{
				env:   env,
				query: "obj.created_by.id < 1",
			},
			want: &Filter{
				Column:         "created_by.id",
				Value:          int64(1),
				ComparisonType: LessThan,
			},
			wantErr: false,
		},
		{
			name: "Single filter less than or equal",
			args: args{
				env:   env,
				query: "obj.created_by.id <= 1",
			},
			want: &Filter{
				Column:         "created_by.id",
				Value:          int64(1),
				ComparisonType: LessThanOrEqual,
			},
			wantErr: false,
		},
		{
			name: "Single filter greater than or equal",
			args: args{
				env:   env,
				query: "obj.created_by.id >= 1",
			},
			want: &Filter{
				Column:         "created_by.id",
				Value:          int64(1),
				ComparisonType: GreaterThanOrEqual,
			},
			wantErr: false,
		},
		{
			name: "Single filter with lookup field",
			args: args{
				env:   env,
				query: "obj.created_by.id == 1",
			},
			want: &Filter{
				Column:         "created_by.id",
				Value:          int64(1),
				ComparisonType: Equal,
			},
			wantErr: false,
		},
		{
			name: "Single with unknown root",
			args: args{
				env:   env,
				query: "description == 'test'",
			},
			wantErr: true,
		},
		{
			name: "And filter",
			args: args{
				env:   env,
				query: "obj.description == 'test' && obj.description == '123'",
			},
			want: &FilterNode{
				Connection: And,
				Nodes: []Filterer{
					&Filter{
						Column:         "description",
						Value:          "test",
						ComparisonType: Equal,
					},
					&Filter{
						Column:         "description",
						Value:          "123",
						ComparisonType: Equal,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "OR filter",
			args: args{
				env:   env,
				query: "obj.description == 'test' || obj.description == '123'",
			},
			want: &FilterNode{
				Connection: Or,
				Nodes: []Filterer{
					&Filter{
						Column:         "description",
						Value:          "test",
						ComparisonType: Equal,
					},
					&Filter{
						Column:         "description",
						Value:          "123",
						ComparisonType: Equal,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Complex filter",
			args: args{
				env:   env,
				query: "(obj.description == 'test' || obj.description == '123') && obj.created_by.id == 1",
			},
			want: &FilterNode{
				Connection: And,
				Nodes: []Filterer{
					&FilterNode{
						Connection: Or,
						Nodes: []Filterer{
							&Filter{
								Column:         "description",
								Value:          "test",
								ComparisonType: Equal,
							},
							&Filter{
								Column:         "description",
								Value:          "123",
								ComparisonType: Equal,
							},
						},
					},
					&Filter{
						Column:         "created_by.id",
						Value:          int64(1),
						ComparisonType: Equal,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFilters(tt.args.env, tt.args.query)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFilters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseFilters() got = %v, want %v", got, tt.want)
			}
		})
	}
}
