//go:build !windows

package main

import "errors"

func protectLocalCredential([]byte) ([]byte, error) {
	return nil, errors.New("encrypted OAuth storage is currently implemented only on Windows")
}
func unprotectLocalCredential([]byte) ([]byte, error) {
	return nil, errors.New("encrypted OAuth storage is currently implemented only on Windows")
}
