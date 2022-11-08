package next

import (
	"errors"
	"reflect"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	defaultKey = "NEXT"
)

var (
	SkipField = errors.New("skip this field")
)

type Func func(hasDefaultValue, zero bool) (interface{}, error)

type Plugin struct {
	key    string
	funcs  map[string]Func
	fields func(*schema.Schema) []*schema.Field
}

var _ gorm.Plugin = &Plugin{}

// NewPlugin constructs a gorm.Plugin for next. It supports to generate a next value automatically.
func NewPlugin() *Plugin {
	return &Plugin{
		key:    defaultKey,
		funcs:  make(map[string]Func),
		fields: func(schema *schema.Schema) []*schema.Field { return schema.Fields },
	}
}

// SetKey sets the key to search in struct tag with gorm key.
// Please avoid built-in gorm tag in https://gorm.io/docs/models.html#Fields-Tags.
// Default value is "next".
func (p *Plugin) SetKey(key string) {
	p.key = strings.ToUpper(key)
}

// SetFields sets the function to get []*schema.Field from *schema.Schema.
// Default function will return all fields in schema.
func (p *Plugin) SetFields(fn func(*schema.Schema) []*schema.Field) {
	p.fields = fn
}

// Register registers the function to generate a next value.
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
				p.trySetNextValue(db.Statement.Schema, rv)
			}
		case reflect.Struct:
			p.trySetNextValue(db.Statement.Schema, db.Statement.ReflectValue)
		}
	})
}

func (p *Plugin) trySetNextValue(schema *schema.Schema, rv reflect.Value) {
	for _, field := range p.fields(schema) {
		key, ok := field.TagSettings[p.key]
		if !ok {
			continue
		}

		next, ok := p.funcs[key]
		if !ok {
			continue
		}

		_, zero := field.ValueOf(rv)
		value, err := next(field.HasDefaultValue, zero)
		if err != nil {
			continue
		}

		_ = field.Set(rv, value)
	}
}
