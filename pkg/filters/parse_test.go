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
			want:    "case.id",
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
			want:    "case.status_condition.id",
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
	env, _ := cel.NewEnv(ProtoToCELVariables(&stubs.TestingObject{})...)
	tests := []struct {
		name    string
		args    args
		want    *FilterExpr
		wantErr bool
	}{
		{
			name: "Single filter",
			args: args{
				env:   env,
				query: "description == 'test'",
			},
			want: &FilterExpr{&Filter{
				Column:         "description",
				Value:          "test",
				ComparisonType: Equal,
			}},
			wantErr: false,
		},
		{
			name: "Empty query filter",
			args: args{
				env:   env,
				query: "",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Empty env",
			args: args{
				env:   nil,
				query: "description == 'test'",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Single filter with unknown field",
			args: args{
				env:   env,
				query: "desc == 'test'",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Single filter with unknown operator",
			args: args{
				env:   env,
				query: "created_by.id === 1",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Single filter more than",
			args: args{
				env:   env,
				query: "created_by.id > 1",
			},
			want: &FilterExpr{&Filter{
				Column:         "created_by.id",
				Value:          int64(1),
				ComparisonType: GreaterThan,
			}},
			wantErr: false,
		},
		{
			name: "Single filter less than",
			args: args{
				env:   env,
				query: "created_by.id < 1",
			},
			want: &FilterExpr{&Filter{
				Column:         "created_by.id",
				Value:          int64(1),
				ComparisonType: LessThan,
			}},
			wantErr: false,
		},
		{
			name: "Single filter less than or equal",
			args: args{
				env:   env,
				query: "created_by.id <= 1",
			},
			want: &FilterExpr{&Filter{
				Column:         "created_by.id",
				Value:          int64(1),
				ComparisonType: LessThanOrEqual,
			}},
			wantErr: false,
		},
		{
			name: "Single filter greater than or equal",
			args: args{
				env:   env,
				query: "created_by.id >= 1",
			},
			want: &FilterExpr{&Filter{
				Column:         "created_by.id",
				Value:          int64(1),
				ComparisonType: GreaterThanOrEqual,
			}},
			wantErr: false,
		},
		{
			name: "Single filter with unknown nested field",
			args: args{
				env:   env,
				query: "created_by.ok >= 1",
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "Single filter with known double nested field",
			args: args{
				env:   env,
				query: "related_entity.contact.id >= 1",
			},
			want: &FilterExpr{&Filter{
				Column:         "related_entity.contact.id",
				Value:          int64(1),
				ComparisonType: GreaterThanOrEqual,
			}},
			wantErr: false,
		},
		{
			name: "Single filter with lookup field",
			args: args{
				env:   env,
				query: "created_by.id == 1",
			},
			want: &FilterExpr{&Filter{
				Column:         "created_by.id",
				Value:          int64(1),
				ComparisonType: Equal,
			}},
			wantErr: false,
		},
		{
			name: "Single with unknown root",
			args: args{
				env:   env,
				query: "root.description == 'test'",
			},
			wantErr: true,
		},
		{
			name: "And filter",
			args: args{
				env:   env,
				query: "description == 'test' && description == '123'",
			},
			want: &FilterExpr{&FilterNode{
				Connection: And,
				Nodes: []*FilterExpr{
					{&Filter{
						Column:         "description",
						Value:          "test",
						ComparisonType: Equal,
					}},
					{&Filter{
						Column:         "description",
						Value:          "123",
						ComparisonType: Equal,
					}},
				},
			}},
			wantErr: false,
		},
		{
			name: "OR filter",
			args: args{
				env:   env,
				query: "description == 'test' || description == '123'",
			},
			want: &FilterExpr{&FilterNode{
				Connection: Or,
				Nodes: []*FilterExpr{
					{&Filter{
						Column:         "description",
						Value:          "test",
						ComparisonType: Equal,
					}},
					{&Filter{
						Column:         "description",
						Value:          "123",
						ComparisonType: Equal,
					}},
				},
			}},
			wantErr: false,
		},
		{
			name: "Complex filter",
			args: args{
				env:   env,
				query: "(description == 'test' || description == '123') && created_by.id == 1",
			},
			want: &FilterExpr{&FilterNode{
				Connection: And,
				Nodes: []*FilterExpr{
					{&FilterNode{
						Connection: Or,
						Nodes: []*FilterExpr{
							{&Filter{
								Column:         "description",
								Value:          "test",
								ComparisonType: Equal,
							}},
							{&Filter{
								Column:         "description",
								Value:          "123",
								ComparisonType: Equal,
							}},
						},
					}},
					{&Filter{
						Column:         "created_by.id",
						Value:          int64(1),
						ComparisonType: Equal,
					}},
				},
			}},
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
