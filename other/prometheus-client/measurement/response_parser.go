package measurement

import (
	"encoding/json"
	"errors"
	"fmt"
)

func Parse_measurement_json(json_data string) ([]NodeMeasurement, error) {
	var measurements []NodeMeasurement
	//var rawMeasurements []NodeMeasurementRaw
	var result map[string]interface{}
	json.Unmarshal([]byte(json_data), &result)

	status, exists := result["status"]
	fmt.Println(status)
	if !exists || status != "success" {
		return measurements, errors.New("unsuccessful query")
	}
	data_bytes, _ := result["data"]
	data := data_bytes.(map[string]interface{})

	measurement_type, exists := data["resultType"]
	fmt.Println(measurement_type)
	if !exists || measurement_type != "vector" {
		return measurements, errors.New("Unknown result type")
	}
	measurement_json, exists := data["result"]

	measurement_data := measurement_json.([]interface{})

	for _, m := range measurement_data {
		metric_data := m.(map[string]interface{})
		metricInfoStr, err := json.Marshal(metric_data)
		if err != nil {
			panic(err)
		}
		var metricInfo NodeMeasurementRaw
		json.Unmarshal([]byte(metricInfoStr), &metricInfo)
		value := NodeMetricValue{UnixTime: metricInfo.Value[0].(float64), Value: metricInfo.Value[1].(string)}
		measurements = append(measurements, NodeMeasurement{Metric: metricInfo.Metric, Value: value})

	}
	fmt.Printf("Have %d measurements\n", len(measurements))
	return measurements, nil
}
