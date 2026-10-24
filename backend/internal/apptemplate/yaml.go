package apptemplate

import "gopkg.in/yaml.v3"

func mustYAML(content string, value any) {
	if err := yaml.Unmarshal([]byte(content), value); err != nil {
		panic(err)
	}
}
func encodeYAML(value any) string {
	content, err := yaml.Marshal(value)
	if err != nil {
		panic(err)
	}
	return string(content)
}
