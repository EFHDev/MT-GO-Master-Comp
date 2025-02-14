package tools

import (
	"fmt"
	"github.com/goccy/go-json"
	"log"
)

func GetJSONRawMessage(path string) json.RawMessage {
	b, err := ReadFile(path)
	if err != nil {
		log.Fatalln(err)
	}

	var rawMap map[string]json.RawMessage
	if err := json.UnmarshalNoEscape(b, &rawMap); err != nil {
		return b
	}

	if data, exists := rawMap["data"]; exists {
		return data
	}

	return b
}

func CheckParsingError(b []byte, err error) error {
	var msg error
	switch t := err.(type) {
	case *json.SyntaxError:
		var start int64 = 0
		if t.Offset-50 > 0 {
			start = t.Offset - 50
		}
		jsn := string(b[start:t.Offset])
		jsn += "<--(Invalid Character)"
		msg = fmt.Errorf("Invalid character at offset %v\n %s", t.Offset, jsn)
	case *json.UnmarshalTypeError:
		var start int64 = 0
		if t.Offset-50 > 0 {
			start = t.Offset - 50
		}
		jsn := string(b[start:t.Offset])
		jsn += "<--(Invalid Type)"
		msg = fmt.Errorf("Invalid value at offset %v\n %s", t.Offset, jsn)
	default:
		msg = err
	}
	return msg
}
