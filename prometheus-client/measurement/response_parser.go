package measurement

import (
	"encoding/json"
	"errors"
	"fmt"
)

func Parse_measurement_json(json_data string) ([]NodeMeasurement, error) {
	var measurements []NodeMeasurement
	var rawMeasurements []NodeMeasurementRaw
	var result map[string]interface{}
	json.Unmarshal([]byte(json_data), &result)

	status, exists := result["status"]
	fmt.Println(status)
	if !exists || status != "success" {
		return measurements, errors.New("unsuccessful query")
	}
	data_bytes, _ := result["data"]
	json.Unmarshal([]byte(data_bytes.(string)), &data)
	measurement_type, exists := data["resultType"]
	fmt.Println(measurement_type)
	if !exists || measurement_type != "vector" {
		return measurements, errors.New("Unknown result type")
	}
	measurement_json, exists := data["result"]

	measurement_json_str, ok := measurement_json.(string)
	if !exists || !ok {
		return measurements, errors.New("No measurements found")
	}
	json.Unmarshal([]byte(measurement_json_str), &rawMeasurements)
	return measurements, nil
}
