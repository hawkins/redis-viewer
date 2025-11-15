package util

import jsoniter "github.com/json-iterator/go"

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func TryPrettyJSON(input string) string {
	var raw interface{}
	if err := json.Unmarshal([]byte(input), &raw); err != nil {
		// Not a valid JSON, return original string
		return input
	}
	// It's a valid JSON, pretty print it
	pretty, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return input
	}
	return string(pretty)
}

func JsonMarshal(data interface{}) ([]byte, error) {
	return json.Marshal(&data)
}

func JsonMarshalIndent(data interface{}) ([]byte, error) {
	return json.MarshalIndent(&data, "", "  ")
}

func JsonUnmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

