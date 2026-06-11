//go:build !windows

package printer

import "fmt"

func List() ([]string, error)              { return nil, fmt.Errorf("only supported on Windows") }
func Detect() (string, error)              { return "", fmt.Errorf("only supported on Windows") }
func Print(name string, data []byte) error { return fmt.Errorf("only supported on Windows") }
