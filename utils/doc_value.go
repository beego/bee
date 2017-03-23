package utils

import "fmt"

type DocValue string

func (d *DocValue) String() string {
	return fmt.Sprint(*d)
}

func (d *DocValue) Set(value string) error {
	*d = DocValue(value)
	return nil
}
