package gen

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"slices"
	"strconv"

	"github.com/go-jet/jet/v2/generator/metadata"
	genpostgres "github.com/go-jet/jet/v2/generator/postgres"
	"github.com/go-jet/jet/v2/generator/template"
	"github.com/go-jet/jet/v2/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"

	"github.com/webitel/webitel-go-kit/cmd/jet-gen-webitel/config"
)

func Generate(cfg config.Config) error {
	db, err := openDatabase(cfg.Database)
	if err != nil {
		return err
	}

	defer db.Close()
	for k, v := range cfg.Schemas {
		fmt.Println("Generating for schema", k)
		if err := genpostgres.GenerateDB(db, k, cfg.Path, genTemplate(v.Tables)); err != nil {
			return fmt.Errorf("generate schema %s: %w", k, err)
		}
	}

	return nil
}

func openDatabase(cfg config.Database) (*sql.DB, error) {
	dsn := fmt.Sprintf("postgresql://%s:%s@%s:%s/%s?sslmode=%s&application_name=jet-go-webitel",
		url.PathEscape(cfg.User),
		url.PathEscape(cfg.Password),
		cfg.Host,
		strconv.Itoa(cfg.Port),
		url.PathEscape(cfg.Name),
		cfg.SslMode,
	)

	connConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	connConfig.MaxConns = 3
	pool, err := pgxpool.NewWithConfig(context.Background(), connConfig)
	if err != nil {
		return nil, err
	}

	db := stdlib.OpenDBFromPool(pool)
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}

	return db, nil
}

func genTemplate(tables config.Tables) template.Template {
	return template.Default(postgres.Dialect).
		UseSchema(func(schemaMetaData metadata.Schema) template.Schema {
			return template.DefaultSchema(schemaMetaData).
				UseModel(template.DefaultModel().
					UseTable(func(table metadata.Table) template.TableModel {
						if shouldSkipTable(tables, table.Name) {
							return template.TableModel{Skip: true}
						}
						return template.DefaultTableModel(table)
					}).
					UseView(func(view metadata.Table) template.ViewModel {
						return template.ViewModel{Skip: true}
					}).
					UseEnum(func(enum metadata.Enum) template.EnumModel {
						return template.EnumModel{Skip: true}
					}),
				).
				UseSQLBuilder(template.DefaultSQLBuilder().
					UseTable(func(table metadata.Table) template.TableSQLBuilder {
						if shouldSkipTable(tables, table.Name) {
							return template.TableSQLBuilder{Skip: true}
						}

						return template.DefaultTableSQLBuilder(table)
					}).
					UseView(func(table metadata.Table) template.ViewSQLBuilder {
						return template.ViewSQLBuilder{Skip: true}
					}).
					UseEnum(func(enum metadata.Enum) template.EnumSQLBuilder {
						return template.EnumSQLBuilder{Skip: true}
					}),
				)
		})
}

func shouldSkipTable(tables config.Tables, table string) bool {
	if slices.Contains(tables.Exclude, table) {
		return true
	}

	if len(tables.Include) == 0 || slices.Contains(tables.Include, table) {
		return false
	}

	return true
}
