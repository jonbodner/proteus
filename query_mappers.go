package proteus

import (
	"github.com/rickar/props"
	"os"
)

type MapMapper map[string]string

func (mm MapMapper) Map(name string) string {
	return mm[name]
}

type propFileMapper struct {
	properties *props.Properties
}

func (pm propFileMapper) Map(name string) string {
	return pm.properties.Get(name)
}

func PropFileToQueryMapper(name string) (QueryMapper, error) {
	file, err := os.Open(name)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	properties, err := props.Read(file)
	if err != nil {
		return nil, err
	}
	return propFileMapper{properties}, nil
}
