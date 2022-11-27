package next

import (
	"errors"
	"fmt"
	"testing"

	"github.com/sony/sonyflake"
	"github.com/stretchr/testify/assert"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
	"gorm.io/gorm/utils/tests"
)

func Example() {
	sf := sonyflake.NewSonyflake(sonyflake.Settings{
		MachineID: func() (uint16, error) { return 1024, nil },
	})

	plugin := NewPlugin()
	plugin.Register("sonyflake", func(_, zero bool) (interface{}, error) {
		if !zero {
			return nil, SkipField
		}
		return sf.NextID()
	})

	db, _ := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true})
	_ = db.Use(plugin)

	type User struct {
		ID   uint64 `gorm:"primaryKey;next:sonyflake;column:id"`
		Name string `gorm:"column:name"`
	}
	user := User{Name: "test"}

	db.Create(&user)
	fmt.Println(sonyflake.MachineID(user.ID))
	// Output:
	// 1024
}

func TestSetKey(t *testing.T) {
	type User struct {
		ID        uint64 `gorm:"primaryKey;n:snowflake;column:id"`
		DisplayID string `gorm:"column:display_id;n:display_id"`
		Name      string `gorm:"column:name"`
	}

	snowflake := func(hasDefaultValue, zero bool) (interface{}, error) {
		if !zero {
			return nil, SkipField
		}
		return 750350266425, nil
	}
	displayID := func(hasDefaultValue, zero bool) (interface{}, error) {
		if !zero {
			return nil, SkipField
		}
		return "20220101A01", nil
	}

	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true})
	assert.NoError(t, err)

	plugin := NewPlugin()
	plugin.SetKey("n")
	plugin.Register("snowflake", snowflake)
	plugin.Register("display_id", displayID)
	assert.NoError(t, db.Use(plugin))

	user := User{Name: "test"}
	assert.NoError(t, db.Create(&user).Error)
	assert.Equal(t, User{ID: 750350266425, DisplayID: "20220101A01", Name: "test"}, user)
}

type User struct {
	ID        uint64 `gorm:"primaryKey;next:snowflake;column:id"`
	DisplayID string `gorm:"column:display_id;next:display_id"`
	Name      string `gorm:"column:name"`
}

func TestReturnError(t *testing.T) {
	target := errors.New("generate next value fail")
	errorNext := func(hasDefaultValue, zero bool) (interface{}, error) {
		return nil, target
	}
	snowflake := func(hasDefaultValue, zero bool) (interface{}, error) {
		if !zero {
			return nil, SkipField
		}
		return 750350266425, nil
	}
	displayID := func(hasDefaultValue, zero bool) (interface{}, error) {
		if !zero {
			return nil, SkipField
		}
		return "20220101A01", nil
	}

	t.Run("UnregisterFuncError", func(t *testing.T) {
		db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true})
		assert.NoError(t, err)

		plugin := NewPlugin()
		plugin.Register("snowflake", snowflake)
		assert.NoError(t, db.Use(plugin))

		user := User{Name: "test"}
		err = db.Create(&user).Error
		assert.Error(t, err)
		assert.Equal(t, "next: unregistered tag display_id", err.Error())

		var e *UnregisteredTagError
		assert.ErrorAs(t, err, &e)
		assert.Equal(t, "display_id", e.Tag)
	})

	t.Run("InvokeFuncError", func(t *testing.T) {
		db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true})
		assert.NoError(t, err)

		plugin := NewPlugin()
		plugin.Register("snowflake", errorNext)
		plugin.Register("display_id", displayID)
		assert.NoError(t, db.Use(plugin))

		user := User{Name: "test"}
		err = db.Create(&user).Error
		assert.Error(t, err)
		assert.Equal(t, "next: invoke func snowflake: generate next value fail", err.Error())

		var e *InvokeFuncError
		assert.ErrorAs(t, err, &e)
		assert.Equal(t, "snowflake", e.Tag)
		assert.Equal(t, target, e.Err)
	})
}

func TestCreateNextStruct(t *testing.T) {
	snowflake := func(hasDefaultValue, zero bool) (interface{}, error) {
		if !zero {
			return nil, SkipField
		}
		return 750350266425, nil
	}
	displayID := func(hasDefaultValue, zero bool) (interface{}, error) {
		if !zero {
			return nil, SkipField
		}
		return "20220101A01", nil
	}

	t.Run("AllFields", func(t *testing.T) {
		cases := []struct {
			funcs   map[string]Func
			user    User
			created User
		}{
			{
				funcs: map[string]Func{
					"snowflake":  snowflake,
					"display_id": displayID,
				},
				user:    User{Name: "test"},
				created: User{ID: 750350266425, DisplayID: "20220101A01", Name: "test"},
			},
			{
				funcs: map[string]Func{
					"snowflake":  snowflake,
					"display_id": displayID,
				},
				user:    User{ID: 1, Name: "test"},
				created: User{ID: 1, DisplayID: "20220101A01", Name: "test"},
			},
			{
				funcs: map[string]Func{
					"snowflake":  snowflake,
					"display_id": displayID,
				},
				user:    User{DisplayID: "20220101B01", Name: "test"},
				created: User{ID: 750350266425, DisplayID: "20220101B01", Name: "test"},
			},
			{
				funcs: map[string]Func{
					"snowflake":  snowflake,
					"display_id": displayID,
				},
				user:    User{ID: 1, DisplayID: "20220101B01", Name: "test"},
				created: User{ID: 1, DisplayID: "20220101B01", Name: "test"},
			},
		}

		for _, tt := range cases {
			db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true})
			assert.NoError(t, err)

			plugin := NewPlugin()
			for tag, fn := range tt.funcs {
				plugin.Register(tag, fn)
			}
			assert.NoError(t, db.Use(plugin))

			assert.NoError(t, db.Create(&tt.user).Error)
			assert.Equal(t, tt.created, tt.user)
		}
	})

	t.Run("PrioritizedPrimaryField", func(t *testing.T) {
		cases := []struct {
			funcs   map[string]Func
			user    User
			created User
		}{
			{
				funcs: map[string]Func{
					"snowflake":  snowflake,
					"display_id": displayID,
				},
				user:    User{Name: "test"},
				created: User{ID: 750350266425, DisplayID: "", Name: "test"},
			},
			{
				funcs: map[string]Func{
					"snowflake":  snowflake,
					"display_id": displayID,
				},
				user:    User{ID: 1, DisplayID: "", Name: "test"},
				created: User{ID: 1, DisplayID: "", Name: "test"},
			},
		}

		for _, tt := range cases {
			db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true})
			assert.NoError(t, err)

			plugin := NewPlugin()
			plugin.SetFields(func(sch *schema.Schema) []*schema.Field {
				return []*schema.Field{sch.PrioritizedPrimaryField}
			})
			for tag, fn := range tt.funcs {
				plugin.Register(tag, fn)
			}
			assert.NoError(t, db.Use(plugin))

			assert.NoError(t, db.Create(&tt.user).Error)
			assert.Equal(t, tt.created, tt.user)
		}
	})
}

type Snowflake struct{ seq uint64 }

func (sf *Snowflake) Next(hasDefaultValue, zero bool) (interface{}, error) {
	if !zero {
		return nil, SkipField
	}

	sf.seq++
	return sf.seq, nil
}

type DisplayID struct{ seq uint64 }

func (d *DisplayID) Next(hasDefaultValue, zero bool) (interface{}, error) {
	if hasDefaultValue || !zero {
		return nil, SkipField
	}

	d.seq++
	return fmt.Sprintf("20220101A%02d", d.seq), nil
}

func TestCreateNextSlice(t *testing.T) {
	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true})
	assert.NoError(t, err)

	plugin := NewPlugin()
	plugin.Register("snowflake", (&Snowflake{}).Next)
	plugin.Register("display_id", (&DisplayID{}).Next)
	assert.NoError(t, db.Use(plugin))

	users := []User{
		{Name: "user1"},
		{Name: "user2"},
		{Name: "user3"},
		{Name: "user4"},
		{Name: "user5"},
	}
	assert.NoError(t, db.Create(users).Error)
	for i, user := range users {
		assert.Equal(t, uint64(i+1), user.ID)
		assert.Equal(t, fmt.Sprintf("20220101A%02d", i+1), user.DisplayID)
	}
}

func TestCreateNextArray(t *testing.T) {
	db, err := gorm.Open(tests.DummyDialector{}, &gorm.Config{DryRun: true})
	assert.NoError(t, err)

	plugin := NewPlugin()
	plugin.Register("snowflake", (&Snowflake{}).Next)
	plugin.Register("display_id", (&DisplayID{}).Next)
	assert.NoError(t, db.Use(plugin))

	users := [...]User{
		{Name: "user1"},
		{Name: "user2"},
		{Name: "user3"},
		{Name: "user4"},
		{Name: "user5"},
	}
	assert.NoError(t, db.Create(&users).Error)
	for i, user := range users {
		assert.Equal(t, uint64(i+1), user.ID)
		assert.Equal(t, fmt.Sprintf("20220101A%02d", i+1), user.DisplayID)
	}
}
