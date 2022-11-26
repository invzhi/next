package internal

import (
	"context"
	"reflect"
)

type (
	// ValueOfFunc is the function signature of [gorm.io/gorm/schema.Field.ValueOf] after normalize.
	ValueOfFunc = func(context.Context, reflect.Value) (value interface{}, zero bool)

	// SetFunc is the function signature of [gorm.io/gorm/schema.Field.Set] after normalize.
	SetFunc = func(context.Context, reflect.Value, interface{}) error
)

// ValueOf normalizes the function signature of [gorm.io/gorm/schema.Field.ValueOf].
func ValueOf(fn interface{}) ValueOfFunc {
	return func(ctx context.Context, v reflect.Value) (interface{}, bool) {
		switch valueOf := fn.(type) {
		case func(reflect.Value) (interface{}, bool): // until gorm v1.22.5
			return valueOf(v)
		case func(context.Context, reflect.Value) (interface{}, bool): // since gorm v1.23.0
			return valueOf(ctx, v)
		default:
			panic("unsupported function signature for gorm/schema.Field.ValueOf")
		}
	}
}

// Set normalizes the function signature of [gorm.io/gorm/schema.Field.Set].
func Set(fn interface{}) SetFunc {
	return func(ctx context.Context, value reflect.Value, v interface{}) error {
		switch set := fn.(type) {
		case func(reflect.Value, interface{}) error: // until gorm v1.22.5
			return set(value, v)
		case func(context.Context, reflect.Value, interface{}) error: // since gorm v1.23.0
			return set(ctx, value, v)
		default:
			panic("unsupported function signature for gorm/schema.Field.Set")
		}
	}
}
