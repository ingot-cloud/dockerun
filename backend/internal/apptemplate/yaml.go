package apptemplate

import "gopkg.in/yaml.v3"

func encodeYAML(value any) string {
	content, err := yaml.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(content)
}
