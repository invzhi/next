package next

import (
	"fmt"

	"github.com/sony/sonyflake"
	"gorm.io/gorm"
	"gorm.io/gorm/utils/tests"
)

func ExampleSonyflake() {
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

	type User struct {
		ID   uint64 `gorm:"primaryKey;next:sonyflake;column:id"`
		Name string `gorm:"column:name"`
	}
	user := User{Name: "test"}

	db, _ := gorm.Open(tests.DummyDialector{}, nil)
	_ = db.Use(plugin)
	db.Create(&user)
	fmt.Println(sonyflake.MachineID(user.ID))
	// Output:
	// 1024
}
