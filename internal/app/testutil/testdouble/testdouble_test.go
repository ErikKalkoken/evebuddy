package testdouble

import (
	"reflect"
	"testing"
)

// Guards against new fields in app.Signals being left nil in tests.
func TestNewSignalsSync_AllFieldsSet(t *testing.T) {
	v := reflect.ValueOf(newSignalsSync()).Elem()
	for i := range v.NumField() {
		f := v.Type().Field(i)
		if !f.IsExported() || f.Type.Kind() != reflect.Interface {
			continue
		}
		if v.Field(i).IsNil() {
			t.Errorf("signal %s is not set", f.Name)
		}
	}
}
