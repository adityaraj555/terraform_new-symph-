package main

import (
	"encoding/json"
	"fmt"
)

type Data struct {
	Roof map[string]interface{} `json:"roof"`
}

func main() {

	str := `{
				"roof": {
							"_countRoofFacets": {
								"marker": null,
								"value": null
							},
							"id": "b0ff589a-262f-4863-841e-1bbb0bf67ef7"
						}
		}`
	var d Data
	err := json.Unmarshal([]byte(str), &d)
	if err != nil {
		fmt.Println("Unmarshalling error")
	}
	fmt.Printf("%v\n", d.Roof["_countRoofFacets"].(map[string]interface{})["marker"] == nil)
	// if nestedData, ok := d.Roof["roof"].(map[string]interface{}); ok {
	// 	fmt.Println(nestedData)
	// }
}
