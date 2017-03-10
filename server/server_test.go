package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"testing"
)

func init() {
	// start webserver to be used in unit tests
	go main()
}

func get(url string, t *testing.T, v interface{}) {
	client := http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("failed to create new get request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %s", err)
	}
	defer resp.Body.Close()

	if resp.Body == nil {
		t.Fatal("no response body returned")
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("body read failed: %s", err)
	}

	err = json.Unmarshal(b, &v)
	if err != nil {
		t.Fatalf("unmarshal failed: %s", err)
	}
}

func TestGetThermostats(t *testing.T) {
	var th []*thermostat
	get("http://localhost:8080/v1/thermostats", t, &th)
	if len(th) < 2 {
		t.Fatalf("received unexpected number of thermostats, expected %d, got %d", 2, len(th))
	}
}

func TestGetThermostat(t *testing.T) {
	cases := map[string]struct {
		num string
	}{
		"thermostat #1": {num: "1"},
		"thermostat #2": {num: "2"},
	}

	for key, tc := range cases {
		var th *thermostat
		get("http://localhost:8080/v1/thermostats/"+tc.num, t, &th)
		if th == nil {
			t.Fatalf("[%s]: thermostat received was nil", key)
		}
	}
}

func TestGetField(t *testing.T) {
	stringCases := map[string]struct {
		field, num, expected string
	}{
		"name1": {field: "name", num: "1", expected: defaultName1},
		"mode1": {field: "mode", num: "1", expected: defaultOpMode1},
		"fan1":  {field: "fan", num: "1", expected: defaultFan1},
		"name2": {field: "name", num: "2", expected: defaultName2},
		"mode2": {field: "mode", num: "2", expected: defaultOpMode2},
		"fan2":  {field: "fan", num: "2", expected: defaultFan2},
	}

	for key, tc := range stringCases {
		var s string
		get("http://localhost:8080/v1/thermostats/"+tc.num+"/"+tc.field, t, &s)
		if s != tc.expected {
			t.Fatalf("[%s]: received field '%s' is incorrect. expected %s, got %s", key, tc.field, tc.expected, s)
		}
	}

	intCases := map[string]struct {
		field, num string
		expected   int
	}{
		"currentTemp1":  {field: "currentTemp", num: "1", expected: defaultCurrentTemp1},
		"coolSetPoint1": {field: "coolSetPoint", num: "1", expected: defaultCoolSetPt1},
		"heatSetPoint1": {field: "heatSetPoint", num: "1", expected: defaultHeatSetPt1},
		"currentTemp2":  {field: "currentTemp", num: "2", expected: defaultCurrentTemp2},
		"coolSetPoint2": {field: "coolSetPoint", num: "2", expected: defaultCoolSetPt2},
		"heatSetPoint2": {field: "heatSetPoint", num: "2", expected: defaultHeatSetPt2},
	}

	for key, tc := range intCases {
		var i int
		get("http://localhost:8080/v1/thermostats/"+tc.num+"/"+tc.field, t, &i)
		if i != tc.expected {
			t.Fatalf("[%s]: received field '%s' is incorrect. expected %d, got %d", key, tc.field, tc.expected, i)
		}
	}

	err := new(errResponse)
	get("http://localhost:8080/v1/thermostats/1/other", t, &err)
	if err == nil {
		t.Fatal("err response expected was empty")
	}
}

func TestPutThermostatBulk(t *testing.T) {
	jsn := `{
		"name": "Other Thermostat",
		"coolSetPoint": 74,
		"heatSetPoint": 71,
		"mode": "cool",
		"fan": "auto"
	}`

	client := http.Client{}
	req, err := http.NewRequest("PUT", "http://localhost:8080/v1/thermostats/1", bytes.NewBuffer([]byte(jsn)))
	if err != nil {
		t.Fatalf("failed to create new PUT request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %s", err)
	}
	defer resp.Body.Close()

	var th *thermostat
	get("http://localhost:8080/v1/thermostats/1", t, &th)

	if th.Name != "Other Thermostat" {
		t.Fatalf("expected name to be %s, got %s", "Other Thermostat", th.Name)
	}
	if th.CoolSetPoint != 74 {
		t.Fatalf("expected cool set point to be %d, got %d", 74, th.CoolSetPoint)
	}
	if th.HeatSetPoint != 71 {
		t.Fatalf("expected heat set point to be %d, got %d", 71, th.HeatSetPoint)
	}
	if th.OperatingMode != "cool" {
		t.Fatalf("expected operating mode to be %s, got %s", "cool", th.OperatingMode)
	}
	if th.FanMode != "auto" {
		t.Fatalf("expected fan mode to be %s, got %s", "auto", th.FanMode)
	}
}

func TestPutThermostatSingle(t *testing.T) {
	stringCases := map[string]struct {
		field, num, val string
	}{
		"name1": {field: "name", num: "1", val: "Basement Thermostat"},
		"mode1": {field: "mode", num: "1", val: "cool"},
		"fan1":  {field: "fan", num: "1", val: "auto"},
		"name2": {field: "name", num: "2", val: "Shed Thermostat"},
		"mode2": {field: "mode", num: "2", val: "heat"},
		"fan2":  {field: "fan", num: "2", val: "on"},
		"mode3": {field: "mode", num: "2", val: "off"},
	}

	for key, tc := range stringCases {
		b := []byte(`{"` + tc.field + `": "` + tc.val + `"}`)
		client := http.Client{}
		req, err := http.NewRequest("PUT", "http://localhost:8080/v1/thermostats/"+tc.num, bytes.NewBuffer(b))
		if err != nil {
			t.Fatalf("[%s]: failed to create new PUT request: %s", key, err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("[%s]: request failed: %s", key, err)
		}
		resp.Body.Close()

		var s string
		get("http://localhost:8080/v1/thermostats/"+tc.num+"/"+tc.field, t, &s)
		if s != tc.val || s == "" {
			t.Fatalf("[%s]: setting field: '%s' failed, expected '%s', got '%s'", key, tc.field, tc.val, s)
		}
	}

	intCases := map[string]struct {
		field, num string
		val        int
	}{
		"coolSetPoint1": {field: "coolSetPoint", num: "1", val: 65},
		"heatSetPoint1": {field: "heatSetPoint", num: "1", val: 73},
		"coolSetPoint2": {field: "coolSetPoint", num: "2", val: 68},
		"heatSetPoint2": {field: "heatSetPoint", num: "2", val: 71},
	}

	for key, tc := range intCases {
		b := []byte(`{"` + tc.field + `": ` + strconv.Itoa(tc.val) + `}`)
		client := http.Client{}
		req, err := http.NewRequest("PUT", "http://localhost:8080/v1/thermostats/"+tc.num, bytes.NewBuffer(b))
		if err != nil {
			t.Fatalf("[%s]: failed to create new PUT request: %s", key, err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("[%s]: request failed: %s", key, err)
		}
		resp.Body.Close()

		var i int
		get("http://localhost:8080/v1/thermostats/"+tc.num+"/"+tc.field, t, &i)
		if i != tc.val || i == 0 {
			t.Fatalf("[%s]: setting field: '%s' failed, expected %d, got %d", key, tc.field, tc.val, i)
		}
	}

	errStringCases := map[string]struct {
		field, val, errMsg string
	}{
		"BadOpMode":  {field: "mode", val: "on", errMsg: "Invalid Operating Mode"},
		"BadFanMode": {field: "fan", val: "off", errMsg: "Invalid Fan Mode"},
	}

	for key, tc := range errStringCases {
		b := []byte(`{"` + tc.field + `": "` + tc.val + `"}`)
		client := http.Client{}
		req, err := http.NewRequest("PUT", "http://localhost:8080/v1/thermostats/1", bytes.NewBuffer(b))
		if err != nil {
			t.Fatalf("[%s]: failed to create new PUT request: %s", key, err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("[%s]: request failed: %s", key, err)
		}
		errRes := new(errResponse)
		b, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("[%s]: failed to read response body: %s", key, err)
		}
		resp.Body.Close()

		err = json.Unmarshal(b, &errRes)
		if err != nil {
			t.Fatalf("[%s]: failed to unmarshal response into *errResponse: %s", key, err)
		}

		if errRes == nil {
			t.Fatalf("[%s]: expected error response, got nothing", key)
		}

		if errRes.Msg != tc.errMsg {
			t.Fatalf("[%s]: incorrect error msg, expected %s, got %s", key, tc.errMsg, errRes.Msg)
		}
	}
}

func TestPostThermostat(t *testing.T) {
	jsn := `{
		"name": "Basement Thermostat",
		"coolSetPoint": 72,
		"heatSetPoint": 68,
		"mode": "heat",
		"fan": "on"
	}`

	client := http.Client{}
	req, err := http.NewRequest("POST", "http://localhost:8080/v1/thermostats", bytes.NewBuffer([]byte(jsn)))
	if err != nil {
		t.Fatalf("failed to create new POST request: %s", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %s", err)
	}

	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to get response body: %s", err)
	}
	resp.Body.Close()

	var th *thermostat
	err = json.Unmarshal(b, &th)
	if err != nil {
		t.Fatalf("failed to unmarshal response body into thermostat: %s", err)
	}

	var thCheck *thermostat
	get("http://localhost:8080/v1/thermostats/"+strconv.Itoa(th.ID), t, &thCheck)

	if thCheck.Name != "Basement Thermostat" {
		t.Fatalf("expected name to be %s, got %s", "Basement Thermostat", thCheck.Name)
	}
	if thCheck.CoolSetPoint != 72 {
		t.Fatalf("expected cool set point to be %d, got %d", 72, thCheck.CoolSetPoint)
	}
	if thCheck.HeatSetPoint != 68 {
		t.Fatalf("expected heat set point to be %d, got %d", 68, thCheck.HeatSetPoint)
	}
	if thCheck.OperatingMode != "heat" {
		t.Fatalf("expected operating mode to be %s, got %s", "heat", thCheck.OperatingMode)
	}
	if thCheck.FanMode != "on" {
		t.Fatalf("expected fan mode to be %s, got %s", "on", thCheck.FanMode)
	}
}
