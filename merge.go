package go-merge

import (
	"reflect"
	"fmt"
	"errors"
)

type MergeOptionType uint32

const (
	OnlyMerge MergeOptionType = 0
	RMerge MergeOptionType = 1
	Override MergeOptionType = 2
	ROverride MergeOptionType = 3
)

type mergeOption struct {
	Override MergeOptionType
	LookUpJson bool
}

func getDefaultmergeOption() mergeOption {
	return mergeOption {
		Override : OnlyMerge,
		LookUpJson : false,
	}
}

func SetMergeOption(override MergeOptionType) func(op *mergeOption) {
	return func(op *mergeOption) {
		op.Override = override
	}
}

func SetLookUpJson(j bool) func(op *mergeOption) {
	return func(op *mergeOption) {
		op.LookUpJson = j
	}
}

func mergeValue(defaultV, userV reflect.Value, op mergeOption) error {
	if defaultV.CanSet() == false {
		return errors.New(fmt.Sprintf("defualt value cannot be set! value name:%s", defaultV.Type().String()))
	}
	if op.Override == Override {
		defaultV.Set(userV)
		return nil
	}
	switch defaultV.Kind() {
		case reflect.Array:
			if err := mergeArray(defaultV, userV, op); err != nil {
				return err
			}
		case reflect.Map:
			if err := mergeMap(defaultV, userV, op); err != nil {
				return err
			}
		case reflect.Struct:
			if err := mergeStruct(defaultV, userV, op); err != nil {
				return err
			}
		case reflect.Slice:
			if err := mergeSlice(defaultV, userV, op); err != nil {
				return err
			}
		default:
			if op.Override == ROverride {
				defaultV.Set(userV)
			}
	}
	return nil
}

func mergeArray(defaultValue, userValue reflect.Value, op mergeOption) error {
	for i := 0; i < defaultValue.Len(); i++ {
		defaultV := defaultValue.Index(i)
		userV := userValue.Index(i)
		if err := mergeValue(defaultV, userV, op); err != nil {
			return err
		}
	}
	return nil
}

func mergeSlice(defaultValue, userValue reflect.Value, op mergeOption) error {
	if userValue.Len() > defaultValue.Len() {
		i := defaultValue.Len()
		defaultValue.SetCap(userValue.Cap())
		defaultValue.SetLen(userValue.Len())
		for i < defaultValue.Len() {
			defaultValue.Index(i).Set(userValue.Index(i))
			i++
		}
	}
	return mergeArray(defaultValue, userValue, op)
}

func mergeMap(defaultValue, userValue reflect.Value, op mergeOption) error {
	iter := userValue.MapRange()
	for iter.Next() {
		userK := iter.Key()
		userV := iter.Value()
		defaultV := defaultValue.MapIndex(userK)
		if defaultV.IsValid() == false {
			defaultValue.SetMapIndex(userK, userV)
		} else {
			if err := mergeValue(defaultV, userV, op); err != nil {
				return err
			}
		}
	}
	return nil
}

func mergeStruct(defaultValue, userValue reflect.Value, op mergeOption) error {
	defaultc := defaultValue.NumField()
	defaultt := defaultValue.Type()
	for i := 0; i < defaultc; i++ {
		userv := userValue.FieldByName(defaultt.Field(i).Name)
		if userv.IsValid() == false {
			continue
		}
		defaultv := defaultValue.Field(i)
		if defaultv.Type().String() != userv.Type().String() {
			return errors.New(fmt.Sprintf("default value type is %s, different from user value Type %s!", defaultv.Type().String(), userv.Type().String()))
		}
		if err := mergeValue(defaultv, userv, op); err != nil {
			return err
		}
	}
	return nil
}

func mergeStructWithMap(defaultValue, userValue reflect.Value, op mergeOption) error {
	defaultc := defaultValue.NumField()
	defaultt := defaultValue.Type()
	for i := 0; i < defaultc; i++ {
		name := defaultt.Field(i).Name
		if op.LookUpJson == true {
			if tag, exist := defaultt.Field(i).Tag.Lookup("json"); exist == true {
				name = tag
			}
		}
		userv := userValue.MapIndex(reflect.ValueOf(name))
		if userv.IsValid() == false {
			continue
		}
		if userv.Kind() == reflect.Interface {
			userv = userv.Elem()
		}
		defaultv := defaultValue.Field(i)
		if (defaultv.Type().Kind() == reflect.Struct && userv.Type().Kind() == reflect.Map) {
			if err := mergeStructWithMap(defaultv, userv, op); err != nil {
				return err
			}
			continue
		}
		if defaultv.Type().String() != userv.Type().String() {
			return errors.New(fmt.Sprintf("default value type is %s, different from user value Type %s!", defaultv.Type().String(), userv.Type().String()))
		}
		if err := mergeValue(defaultv, userv, op); err != nil {
			return err
		}
	}
	return nil
}

func MergeArray(defaultArrayPtr, userArray interface{}, options ...func(op *mergeOption)) error {
	option := getDefaultmergeOption()
	for _, op := range options {
		op(&option)
	}
	if reflect.TypeOf(defaultArrayPtr).Kind() != reflect.Ptr {
		return errors.New("defaultArrayPtr should be a pointer to array")
	}
	defaultValue := reflect.ValueOf(defaultArrayPtr).Elem()
	if defaultValue.Kind() != reflect.Array {
		return errors.New("defaultArrayPtr should be a pointer to array")
	}
	userValue := reflect.ValueOf(userArray)
	if userValue.Kind() != reflect.Array {
		return errors.New("userArray should be a Array")
	}
	if defaultValue.Type().String() != userValue.Type().String() {
		return errors.New(fmt.Sprintf("defaultArray type is %s, different from userArray Type %s!", defaultValue.Type().String(), userValue.Type().String()))
	}
	return mergeMap(defaultValue, userValue, option)
}

func MergeSlice(defaultSlicePtr, userSlice interface{}, options ...func(op *mergeOption)) error {
	option := getDefaultmergeOption()
	for _, op := range options {
		op(&option)
	}
	if reflect.TypeOf(defaultSlicePtr).Kind() != reflect.Ptr {
		return errors.New("defaultSlicePtr should be a pointer to slice")
	}
	defaultValue := reflect.ValueOf(defaultSlicePtr).Elem()
	if defaultValue.Kind() != reflect.Slice {
		return errors.New("defaultSlicePtr should be a pointer to slice")
	}
	userValue := reflect.ValueOf(userSlice)
	if userValue.Kind() != reflect.Slice {
		return errors.New("userSlice should be a Slice")
	}
	if defaultValue.Type().String() != userValue.Type().String() {
		return errors.New(fmt.Sprintf("defaultSlice type is %s, different from userSlice Type %s!", defaultValue.Type().String(), userValue.Type().String()))
	}
	return mergeMap(defaultValue, userValue, option)
}

func MergeMap(defaultMapPtr, userMap interface{}, options ...func(op *mergeOption)) error {
	option := getDefaultmergeOption()
	for _, op := range options {
		op(&option)
	}
	if reflect.TypeOf(defaultMapPtr).Kind() != reflect.Ptr {
		return errors.New("defaultMapPtr should be a pointer to map")
	}
	defaultValue := reflect.ValueOf(defaultMapPtr).Elem()
	if defaultValue.Kind() != reflect.Map {
		return errors.New("defaultMapPtr should be a pointer to map")
	}
	userValue := reflect.ValueOf(userMap)
	if userValue.Kind() != reflect.Map {
		return errors.New("userMap should be a map")
	}
	if defaultValue.Type().String() != userValue.Type().String() {
		return errors.New(fmt.Sprintf("defaultMap type is %s, different from userMap Type %s!", defaultValue.Type().String(), userValue.Type().String()))
	}
	return mergeMap(defaultValue, userValue, option)
}

func MergeStruct(defaultStructPtr, userStruct interface{}, options ...func(op *mergeOption)) error {
	option := getDefaultmergeOption()
	for _, op := range options {
		op(&option)
	}
	if reflect.TypeOf(defaultStructPtr).Kind() != reflect.Ptr {
		return errors.New("defaultStructPtr should be a pointer to struct")
	}
	defaultValue := reflect.ValueOf(defaultStructPtr).Elem() //to get the struct
	if defaultValue.Kind() != reflect.Struct {
		return errors.New("defaultStructPtr should be a pointer to struct")
	}
	userValue := reflect.ValueOf(userStruct)
	if userValue.Kind() != reflect.Struct {
		return errors.New("userStruct should be a struct")
	}
	return mergeStruct(defaultValue, userValue, option)
}

//struct field can have tag as key in map
func MergeStructWithMap(defaultStructPtr, userMap interface{}, options ...func(op *mergeOption)) error {
	option := getDefaultmergeOption()
	for _, op := range options {
		op(&option)
	}
	if reflect.TypeOf(defaultStructPtr).Kind() != reflect.Ptr {
		return errors.New("defaultStructPtr should be a pointer to struct")
	}
	defaultValue := reflect.ValueOf(defaultStructPtr).Elem() //to get the struct
	if defaultValue.Kind() != reflect.Struct {
		return errors.New("defaultStructPtr should be a pointer to struct")
	}
	userValue := reflect.ValueOf(userMap)
	if userValue.Kind() != reflect.Map {
		return errors.New("userMap should be a map")
	}
	if userValue.Type().Key().String() != reflect.TypeOf("a").String() {
		return errors.New("key type of userMap should be string")
	}
	return mergeStructWithMap(defaultValue, userValue, option)
}