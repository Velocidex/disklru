package disklru

import (
	"encoding/json"
)

type JsonEncoder struct{}

func (self JsonEncoder) Encode(obj interface{}) ([]byte, error) {
	return json.Marshal(obj)
}

func (self JsonEncoder) Decode(in []byte) (interface{}, error) {
	var res interface{}

	err := json.Unmarshal(in, &res)

	return res, err
}
