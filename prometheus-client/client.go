package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.gatech.edu/cs-epl/mesh/measurement"
)

func get(url string) (string, error) {
	c := http.Client{Timeout: time.Duration(1) * time.Second}
	resp, err := c.Get(url)
	if err != nil {
		fmt.Printf("Error %s", err)
		return "", err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return string(body[:]), err
}

func main() {
	url := "http://10.43.191.57:9090/api/v1/query?query=node_network_receive_bytes_total"
	response, err := get(url)
	if err == nil {
		fmt.Println("Parsing..")
		measurements, err := measurement.Parse_measurement_json(response)
		if err != nil {
			fmt.Printf("Got %d measurements\n", len(measurements))
		}
	}
}
