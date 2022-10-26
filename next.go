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
	key   string
	funcs map[string]Func
}

var _ gorm.Plugin = &Plugin{}

// NewPlugin constructs a gorm.Plugin for next. It supports to generate a next value automatically.
func NewPlugin() *Plugin {
	funcs := make(map[string]Func)
	return &Plugin{key: defaultKey, funcs: funcs}
}

func (p *Plugin) SetKey(key string) {
	p.key = strings.ToUpper(key)
}

func (p *Plugin) Register(tag string, fn Func) {
	p.funcs[tag] = fn
}

func (p *Plugin) Name() string {
	return "next"
}

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
				p.trySetNextValue(db.Statement.Schema.Fields, rv)
			}
		case reflect.Struct:
			p.trySetNextValue(db.Statement.Schema.Fields, db.Statement.ReflectValue)
		}
	})
}

func (p *Plugin) trySetNextValue(fields []*schema.Field, rv reflect.Value) {
	for _, field := range fields {
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
