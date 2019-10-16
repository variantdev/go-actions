package cmd

import "strings"

type StringSlice []string

func (f *StringSlice) String() string {
	return strings.Join(*f, ", ")
}

func (f *StringSlice) Set(value string) error {
	*f = append(*f, value)
	return nil
}
