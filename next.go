// Package next provides a gorm plugin to set next value for fields.
package next

import (
	"context"
	"errors"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"

	"github.com/invzhi/next/internal"
)

const (
	defaultKey = "NEXT"
)

var (
	// SkipField is used as a return error in Func to indicate that field is to be skipped.
	SkipField = errors.New("skip this field")
)

// Func is the function type to generate the next value.
type Func func(hasDefaultValue, zero bool) (interface{}, error)

type Plugin struct {
	key    string
	funcs  map[string]Func
	fields func(*schema.Schema) []*schema.Field
}

var _ gorm.Plugin = &Plugin{}

// NewPlugin constructs a gorm.Plugin for next.
func NewPlugin() *Plugin {
	return &Plugin{
		key:    defaultKey,
		funcs:  make(map[string]Func),
		fields: func(schema *schema.Schema) []*schema.Field { return schema.Fields },
	}
}

// SetKey sets the key to search in struct tag with gorm key.
// Please avoid gorm built-in tag in https://gorm.io/docs/models.html#Fields-Tags.
// Default value is "next".
func (p *Plugin) SetKey(key string) {
	p.key = strings.ToUpper(key)
}

// SetFields could customize the scope of fields need to generate a next value.
// Default scope is all fields in schema.
//
// For example, only generate next value for prioritized primary field:
//
//  plugin.SetFields(func(sch *schema.Schema) []*schema.Field {
//      return []*schema.Field{sch.PrioritizedPrimaryField}
//  })
func (p *Plugin) SetFields(fn func(*schema.Schema) []*schema.Field) {
	p.fields = fn
}

// Register registers the function to generate a next value for field with tag.
func (p *Plugin) Register(tag string, fn Func) {
	p.funcs[tag] = fn
}

// Name implements the gorm.Plugin interface.
func (p *Plugin) Name() string {
	return "next"
}

// Initialize implements the gorm.Plugin interface.
func (p *Plugin) Initialize(db *gorm.DB) error {
	return db.Callback().Create().Before("gorm:create").Register("next:before_create", func(db *gorm.DB) {
		if db.Statement.Schema == nil {
			return
		}

		switch db.Statement.ReflectValue.Kind() {
		case reflect.Slice, reflect.Array:
			for i := 0; i < db.Statement.ReflectValue.Len(); i++ {
				rv := db.Statement.ReflectValue.Index(i)
				if reflect.Indirect(rv).Kind() != reflect.Struct {
					return
				}
				p.trySetNextValue(db.Statement.Context, db.Statement.Schema, rv)
			}
		case reflect.Struct:
			p.trySetNextValue(db.Statement.Context, db.Statement.Schema, db.Statement.ReflectValue)
		}
	})
}

func (p *Plugin) trySetNextValue(ctx context.Context, schema *schema.Schema, rv reflect.Value) {
	for _, field := range p.fields(schema) {
		key, ok := field.TagSettings[p.key]
		if !ok {
			continue
		}

		next, ok := p.funcs[key]
		if !ok {
			continue
		}

		valueOfField := internal.ValueOf(field.ValueOf)
		_, zero := valueOfField(ctx, rv)
		value, err := next(field.HasDefaultValue, zero)
		if err != nil {
			continue
		}

		setField := internal.Set(field.Set)
		_ = setField(ctx, rv, value)
	}
}
