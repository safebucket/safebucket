package database

import (
	"reflect"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var uuidType = reflect.TypeFor[uuid.UUID]()

// RegisterCallbacks registers GORM callbacks for cross-database compatibility.
func RegisterCallbacks(db *gorm.DB) {
	if err := db.Callback().Create().Before("gorm:create").Register("generate_uuid", generateUUID); err != nil {
		zap.L().Fatal("Failed to register UUID callback", zap.Error(err))
	}
}

// generateUUID sets a UUID primary key if it is zero-valued before creation.
func generateUUID(db *gorm.DB) {
	if db.Statement.Schema == nil {
		return
	}

	for _, field := range db.Statement.Schema.PrimaryFields {
		if field.DBName != "id" {
			continue
		}
		fieldType := field.FieldType
		for fieldType.Kind() == reflect.Pointer {
			fieldType = fieldType.Elem()
		}
		if fieldType != uuidType {
			continue
		}

		//exhaustive:ignore
		switch db.Statement.ReflectValue.Kind() {
		case reflect.Struct:
			_, isZero := field.ValueOf(db.Statement.Context, db.Statement.ReflectValue)
			if isZero {
				if err := field.Set(db.Statement.Context, db.Statement.ReflectValue, uuid.New()); err != nil {
					zap.L().Error("Failed to set UUID on primary key field",
						zap.String("field", field.Name), zap.Error(err))
					_ = db.AddError(err)
					return
				}
			}
		case reflect.Slice, reflect.Array:
			for i := range db.Statement.ReflectValue.Len() {
				elem := db.Statement.ReflectValue.Index(i)
				_, isZero := field.ValueOf(db.Statement.Context, elem)
				if isZero {
					if err := field.Set(db.Statement.Context, elem, uuid.New()); err != nil {
						zap.L().Error("Failed to set UUID on primary key field",
							zap.String("field", field.Name), zap.Int("index", i), zap.Error(err))
						_ = db.AddError(err)
						return
					}
				}
			}
		}
	}
}
