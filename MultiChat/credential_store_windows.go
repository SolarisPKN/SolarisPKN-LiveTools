//go:build windows

package main

import (
	"errors"
	"unsafe"

	"golang.org/x/sys/windows"
)

func protectLocalCredential(clear []byte) ([]byte, error) {
	if len(clear) == 0 {
		return nil, errors.New("empty credential")
	}
	in := windows.DataBlob{Size: uint32(len(clear)), Data: &clear[0]}
	var out windows.DataBlob
	err := windows.CryptProtectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	if err != nil {
		return nil, err
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.Data))))
	if out.Size == 0 || out.Data == nil {
		return nil, errors.New("DPAPI empty result")
	}
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}

func unprotectLocalCredential(encrypted []byte) ([]byte, error) {
	if len(encrypted) == 0 {
		return nil, errors.New("empty protected credential")
	}
	in := windows.DataBlob{Size: uint32(len(encrypted)), Data: &encrypted[0]}
	var out windows.DataBlob
	err := windows.CryptUnprotectData(&in, nil, nil, 0, nil, windows.CRYPTPROTECT_UI_FORBIDDEN, &out)
	if err != nil {
		return nil, errors.New("unable to decrypt credentials for current Windows user")
	}
	defer windows.LocalFree(windows.Handle(uintptr(unsafe.Pointer(out.Data))))
	if out.Size == 0 || out.Data == nil {
		return nil, errors.New("DPAPI empty result")
	}
	return append([]byte(nil), unsafe.Slice(out.Data, out.Size)...), nil
}
