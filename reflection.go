package main

import (
	"reflect"
	"unsafe"

	wz "github.com/diamondo25/go-wz"
)

// Access unexported fields using unsafe reflection
// This is needed because the go-wz library doesn't export all data fields

func getSoundDataViaReflection(sound *wz.WZSoundDX8) []byte {
	val := reflect.ValueOf(sound).Elem()
	field := val.FieldByName("soundData")
	if field.IsValid() {
		// Use unsafe to access unexported field
		return *(*[]byte)(unsafe.Pointer(field.UnsafeAddr()))
	}
	return []byte{}
}

func getHeaderDataViaReflection(sound *wz.WZSoundDX8) []byte {
	val := reflect.ValueOf(sound).Elem()
	field := val.FieldByName("headerData")
	if field.IsValid() {
		return *(*[]byte)(unsafe.Pointer(field.UnsafeAddr()))
	}
	return []byte{}
}

func getCanvasDataViaReflection(canvas *wz.WZCanvas) []byte {
	val := reflect.ValueOf(canvas).Elem()
	field := val.FieldByName("data")
	if field.IsValid() {
		return *(*[]byte)(unsafe.Pointer(field.UnsafeAddr()))
	}
	return []byte{}
}
