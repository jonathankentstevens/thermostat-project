package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sort"
	"strconv"
	"sync"
	"time"

	"github.com/buaazp/fasthttprouter"
	"github.com/valyala/fasthttp"
)

// thermostat holds all data pertaining to a single unit. The fields must be exported in order to be
// handled by the json Unmarshaler/Marshaler interfaces
type thermostat struct {
	Id            int       `json:"id"`
	Name          string    `json:"name"`
	CurrentTemp   int       `json:"currentTemp"`
	PreviousTemp  int       `json:"previousTemp"`
	OperatingMode string    `json:"mode"`
	CoolSetPoint  int       `json:"coolSetPoint"`
	HeatSetPoint  int       `json:"heatSetPoint"`
	FanMode       string    `json:"fan"`
	LastChanged   time.Time `json:"lastChanged"`
}

// updateThermostat is the desired thermostat state sent in through the api @ /v1/thermostats/:id
type updateThermostat struct {
	Name          string `json:"name"`
	Temperature   int    `json:"currentTemp"` // only included to provide proper error if included
	OperatingMode string `json:"mode"`
	CoolSetPoint  int    `json:"coolSetPoint"`
	HeatSetPoint  int    `json:"heatSetPoint"`
	FanMode       string `json:"fan"`
}

// currentState provides safe concurrent access for reads. It is held in a map to provide a faster lookups of
// the desired thermostat based on the id given
type currentState struct {
	sync.Mutex
	thermostats map[int]*thermostat
}

// errResponse is the structure of any errors that may be returned to the client
type errResponse struct {
	Code        int    `json:"code"`
	Msg         string `json:"message"`
	Description string `json:"description"`
}

const (
	// specify default values for first two thermostats when the app starts up

	defaultName1 = "Downstairs Thermostat"
	defaultName2 = "Upstairs Thermostat"

	defaultCurrentTemp1 = 71
	defaultCurrentTemp2 = 72

	defaultOpMode1 = "heat"
	defaultOpMode2 = "cool"

	defaultCoolSetPt1 = 68
	defaultCoolSetPt2 = 69

	defaultHeatSetPt1 = 72
	defaultHeatSetPt2 = 73

	defaultFan1 = "auto"
	defaultFan2 = "on"
)

var (
	home          currentState
	validOpModes  []string
	validFanModes []string
	validFields   []string
	minCoolSetPt  int = 30
	maxCoolSetPt  int = 100
	minHeatSetPt  int = 30
	maxHeatSetPt  int = 100
)

func init() {
	// initialize the initial state of the home with generic values for both thermostats
	home.thermostats = make(map[int]*thermostat)
	home.thermostats[1] = &thermostat{
		Id:            1,
		Name:          defaultName1,
		CurrentTemp:   71,
		OperatingMode: defaultOpMode1,
		CoolSetPoint:  defaultCoolSetPt1,
		HeatSetPoint:  defaultHeatSetPt1,
		FanMode:       defaultFan1,
		LastChanged:   time.Now(),
	}
	home.thermostats[2] = &thermostat{
		Id:            2,
		Name:          defaultName2,
		CurrentTemp:   72,
		OperatingMode: defaultOpMode2,
		CoolSetPoint:  defaultCoolSetPt2,
		HeatSetPoint:  defaultHeatSetPt2,
		FanMode:       defaultFan2,
		LastChanged:   time.Now(),
	}

	// set the valid modes
	validOpModes = []string{"cool", "heat", "off"}
	validFanModes = []string{"auto", "on"}
	validFields = []string{"name", "currentTemp", "mode", "coolSetPoint", "heatSetPoint", "fan"}
}

// Thermostat is a getter to provide safe concurrent read access to a specific thermostat
func (home *currentState) Thermostat(id int) (*thermostat, *errResponse) {
	home.Lock()
	defer home.Unlock()

	t, ok := home.thermostats[id]
	if !ok {
		return nil, &errResponse{
			Code:        http.StatusNotFound,
			Msg:         "Not Found",
			Description: "No thermostat found for id: " + strconv.Itoa(id),
		}
	}

	return t, nil
}

// UpdateThermostat provides a type safe way to perform updates on a specific thermostat
func (home *currentState) UpdateThermostat(target *thermostat, desired updateThermostat) {
	home.Lock()

	updated := &thermostat{
		Id: target.Id,
	}

	// make sure new name isn't empty before changing
	if desired.Name != "" {
		updated.Name = desired.Name
	} else {
		updated.Name = target.Name
	}

	// make sure new operating mode isn't empty before changing
	if desired.OperatingMode != "" {
		updated.OperatingMode = desired.OperatingMode
	} else {
		updated.OperatingMode = target.OperatingMode
	}

	// make sure cool set point isn't empty before changing
	if desired.CoolSetPoint != 0 {
		updated.CoolSetPoint = desired.CoolSetPoint
	} else {
		updated.CoolSetPoint = target.CoolSetPoint
	}

	// make sure heat set point isn't empty before changing
	if desired.HeatSetPoint != 0 {
		updated.HeatSetPoint = desired.HeatSetPoint
	} else {
		updated.HeatSetPoint = target.HeatSetPoint
	}

	// make sure new fan mode isn't empty before changing
	if desired.FanMode != "" {
		updated.FanMode = desired.FanMode
	} else {
		updated.FanMode = target.FanMode
	}

	// make sure that the previousTemp only gets changed if the currentTemp does
	temp := (updated.CoolSetPoint + updated.HeatSetPoint) / 2
	if temp != target.CurrentTemp {
		updated.CurrentTemp = temp
		updated.PreviousTemp = target.CurrentTemp
	}

	// set the last time the thermostat's settings were changed to now
	updated.LastChanged = time.Now()
	home.thermostats[target.Id] = updated

	home.Unlock()
}

// AddThermostat takes the desired thermostat state and adds it to our map of thermostats
func (home *currentState) AddThermostat(desired updateThermostat) int {
	home.Lock()

	// find the next id to use as the identifier for the new thermostat
	var ints []int
	for key := range home.thermostats {
		ints = append(ints, key)
	}
	sort.Ints(ints)
	newId := ints[len(ints)-1] + 1 // +1 because our first id starts at 1, not 0

	updated := &thermostat{
		Id: newId,
	}

	// set default name if not provided
	if desired.Name != "" {
		updated.Name = desired.Name
	} else {
		updated.Name = "Thermostat #" + strconv.Itoa(newId)
	}

	// set operating mode to 'off' if not provided
	if desired.OperatingMode != "" {
		updated.OperatingMode = desired.OperatingMode
	} else {
		updated.OperatingMode = "off"
	}

	// set cool set point to 71 if not provided
	if desired.CoolSetPoint != 0 {
		updated.CoolSetPoint = desired.CoolSetPoint
	} else {
		updated.CoolSetPoint = 71
	}

	// set heat set point to 71 if not provided
	if desired.HeatSetPoint != 0 {
		updated.HeatSetPoint = desired.HeatSetPoint
	} else {
		updated.HeatSetPoint = 71
	}

	// set fan mode to 'auto' if not provided
	if desired.FanMode != "" {
		updated.FanMode = desired.FanMode
	} else {
		updated.FanMode = "auto"
	}

	if updated.CoolSetPoint != 0 && updated.HeatSetPoint != 0 {
		updated.CurrentTemp = (updated.CoolSetPoint + updated.HeatSetPoint) / 2
	} else {
		updated.CurrentTemp = 71
	}

	// set the last time the thermostat's settings were changed to now
	updated.LastChanged = time.Now()
	home.thermostats[newId] = updated
	home.Unlock()

	return newId
}

// inArray determines whether or not a string is in the provided string array
func inArray(char string, strings []string) bool {
	for _, a := range strings {
		if a == char {
			return true
		}
	}
	return false
}

// sendJSON sends the provided data back to the client as a json byte array
func sendJSON(req *fasthttp.RequestCtx, v interface{}) error {
	jsn, err := json.Marshal(v)
	if err != nil {
		return err
	}

	req.Response.Header.Set("Content-Type", "application/json")
	_, err = req.Write(jsn)
	if err != nil {
		return err
	}

	return nil
}

// validateOpMode makes sure the operating mode passed is a valid option
func validateOpMode(val string) *errResponse {
	if val != "" && !inArray(val, validOpModes) {
		return &errResponse{
			Code:        http.StatusBadRequest,
			Msg:         "Invalid Operating Mode",
			Description: "The operating mode provided is not valid. Valid choices are: 'cool', 'heat', or 'off'.",
		}
	}

	return nil
}

// validateFanMode makes sure the fan mode passed is a valid option
func validateFanMode(val string) *errResponse {
	if val != "" && !inArray(val, validFanModes) {
		return &errResponse{
			Code:        http.StatusBadRequest,
			Msg:         "Invalid Fan Mode",
			Description: "The fan mode provided is not valid. Valid choices are: 'auto' or 'on'.",
		}
	}

	return nil
}

// validateCoolSetPt makes sure the cool set point passed is between the min and max allowed
func validateCoolSetPt(val int) *errResponse {
	if val != 0 && (val > maxCoolSetPt || val < minCoolSetPt) {
		return &errResponse{
			Code:        http.StatusBadRequest,
			Msg:         "Invalid Cool Set Point",
			Description: "The cool set point provided is not within the allowed range. It must be between " + strconv.Itoa(minCoolSetPt) + " and " + strconv.Itoa(maxCoolSetPt) + " degrees Fahrenheit.",
		}
	}
	return nil
}

// validateHeatSetPt makes sure the heat set point passed is between the min and max allowed
func validateHeatSetPt(val int) *errResponse {
	if val != 0 && (val > maxHeatSetPt || val < minHeatSetPt) {
		return &errResponse{
			Code:        http.StatusBadRequest,
			Msg:         "Invalid Heat Set Point",
			Description: "The heat set point provided is not within the allowed range. It must be between " + strconv.Itoa(minHeatSetPt) + " and " + strconv.Itoa(maxHeatSetPt) + " degrees Fahrenheit.",
		}
	}
	return nil
}

// validateData takes in the desired new state of a thermostat and makes sure all fields pass
// their specific validation
func validateData(desired updateThermostat) *errResponse {
	if desired.Temperature != 0 {
		return &errResponse{
			Code:        http.StatusBadRequest,
			Msg:         "Non-Writable Field",
			Description: "The field 'currentTemp' is not a writable field. You must set the cool or heat set point (coolSetPoint/heatSetPoint) instead.",
		}
	}

	// verify the operating mode passed in is a valid operating mode
	if err := validateOpMode(desired.OperatingMode); err != nil {
		return err
	}

	// verify the fan mode passed in is a valid fan mode
	if err := validateFanMode(desired.FanMode); err != nil {
		return err
	}

	// verify cool set point is within the allowed range if not empty
	if err := validateCoolSetPt(desired.CoolSetPoint); err != nil {
		return err
	}

	// verify heat set point is within the allowed range if not empty
	if err := validateHeatSetPt(desired.HeatSetPoint); err != nil {
		return err
	}

	return nil
}

// HandleRoute is middleware that sets the content type to json and performs validation of the desired
// thermostat if an id is present in the query string before continuing on to any routes
func HandleRoute(h fasthttp.RequestHandler) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(req *fasthttp.RequestCtx) {
		req.SetContentType("application/json")

		// if there is an :id param in the query string, we validate that the id provided is a
		// valid integer and that we can find a thermostat based on that id
		idCheck := req.UserValue("id")
		if idCheck != nil {
			id, err := strconv.Atoi(idCheck.(string))
			if err != nil {
				res := &errResponse{
					Code:        http.StatusBadRequest,
					Msg:         "Invalid identifier provided",
					Description: err.Error(),
				}
				req.SetStatusCode(http.StatusBadRequest)
				sendJSON(req, res)
			}

			// verify id passed exists in our map of thermostats
			t, errRes := home.Thermostat(id)
			if errRes != nil {
				req.SetStatusCode(http.StatusNotFound)
				sendJSON(req, errRes)
				return
			}

			req.SetUserValue("thermostat", t)
		}

		h(req)
	})
}

// Index serves the index of the api
func Index(req *fasthttp.RequestCtx) {
	req.SetStatusCode(http.StatusOK)
	req.SetBodyString("Index Page")
}

// GetThermostats is the handler to return the information about all of the thermostats in the home
func GetThermostats(req *fasthttp.RequestCtx) {
	var therms []*thermostat
	for _, thermostat := range home.thermostats {
		therms = append(therms, thermostat)
	}

	if len(therms) == 0 {
		res := &errResponse{
			Code:        http.StatusNotFound,
			Msg:         "Not Found",
			Description: "No thermostats were found.",
		}
		req.SetStatusCode(http.StatusNotFound)
		sendJSON(req, res)
		return
	}

	req.SetStatusCode(http.StatusOK)
	sendJSON(req, therms)
}

// GetThermostat is the handler to return all information about a specific thermostat based on the id given
func GetThermostat(req *fasthttp.RequestCtx) {
	req.SetStatusCode(http.StatusOK)
	sendJSON(req, req.UserValue("thermostat").(*thermostat))
}

// GetField is the handler to return a specific property of a specific thermostat
func GetField(req *fasthttp.RequestCtx) {
	field := req.UserValue("field").(string)
	if !inArray(field, validFields) {
		res := &errResponse{
			Code:        http.StatusBadRequest,
			Msg:         "Invalid Property",
			Description: "The property provided is not a valid property of a thermostat. Valid choices are: 'name', 'currentTemp', 'mode', 'coolSetPoint', 'heatSetPoint', or 'fan'.",
		}
		req.SetStatusCode(http.StatusBadRequest)
		sendJSON(req, res)
		return
	}

	// no need to check if this exists, already validated in middleware
	t := req.UserValue("thermostat").(*thermostat)

	var returnVal interface{}
	var isEmpty bool

	switch field {
	case "name":
		if t.Name == "" {
			isEmpty = true
		} else {
			returnVal = t.Name
		}
	case "currentTemp":
		if t.CurrentTemp == 0 {
			isEmpty = true
		} else {
			returnVal = t.CurrentTemp
		}
	case "mode":
		if t.OperatingMode == "" {
			isEmpty = true
		} else {
			returnVal = t.OperatingMode
		}
	case "coolSetPoint":
		if t.CoolSetPoint == 0 {
			isEmpty = true
		} else {
			returnVal = t.CoolSetPoint
		}
	case "heatSetPoint":
		if t.HeatSetPoint == 0 {
			isEmpty = true
		} else {
			returnVal = t.HeatSetPoint
		}
	case "fan":
		if t.FanMode == "" {
			isEmpty = true
		} else {
			returnVal = t.FanMode
		}
	}

	if isEmpty {
		res := &errResponse{
			Code:        http.StatusNotFound,
			Msg:         "Not Found",
			Description: "No field '" + field + "' exists for requested thermostat.",
		}
		req.SetStatusCode(http.StatusNotFound)
		sendJSON(req, res)
		return
	}

	req.SetStatusCode(http.StatusOK)
	sendJSON(req, returnVal)
}

// PutThermostat handles bulk updates for a specific thermostat
func PutThermostat(req *fasthttp.RequestCtx) {

	// verify json body was valid to api spec
	var desired updateThermostat
	if err := json.Unmarshal(req.PostBody(), &desired); err != nil {
		res := &errResponse{
			Code:        http.StatusBadRequest,
			Msg:         "Invalid JSON body provided",
			Description: err.Error(),
		}
		req.SetStatusCode(http.StatusBadRequest)
		sendJSON(req, res)
		return
	}

	// perform validation of the new desired state of the thermostat
	err := validateData(desired)
	if err != nil {
		req.SetStatusCode(http.StatusBadRequest)
		sendJSON(req, err)
		return
	}

	// retrieve our target thermostat found by the id provided in the query string
	target := req.UserValue("thermostat").(*thermostat)

	// update the thermostat once all data has been validated
	home.UpdateThermostat(target, desired)

	req.SetStatusCode(http.StatusOK)
}

// PostThermostat is the handler to add a new thermostat to the home
func PostThermostat(req *fasthttp.RequestCtx) {

	// verify json body was valid to api spec
	var desired updateThermostat
	if err := json.Unmarshal(req.PostBody(), &desired); err != nil {
		res := &errResponse{
			Code:        http.StatusBadRequest,
			Msg:         "Invalid JSON body provided",
			Description: err.Error(),
		}
		req.SetStatusCode(http.StatusBadRequest)
		sendJSON(req, res)
		return
	}

	// perform validation of the new desired state of the thermostat
	err := validateData(desired)
	if err != nil {
		req.SetStatusCode(http.StatusBadRequest)
		sendJSON(req, err)
		return
	}

	// add new thermostat based on the desired state given
	newId := home.AddThermostat(desired)

	req.SetStatusCode(http.StatusOK)

	newThermostat, err := home.Thermostat(newId)
	if err != nil {
		req.SetStatusCode(http.StatusBadRequest)
		sendJSON(req, err)
		return
	}

	// send back the new thermostat so the client has access to the new id
	sendJSON(req, newThermostat)
}

func main() {

	// initialize router
	r := fasthttprouter.New()

	// build router specs
	r.GET("/", Index)
	r.GET("/v1/thermostats", HandleRoute(GetThermostats))
	r.GET("/v1/thermostats/:id", HandleRoute(GetThermostat))
	r.GET("/v1/thermostats/:id/:field", HandleRoute(GetField))
	r.PUT("/v1/thermostats/:id", HandleRoute(PutThermostat))
	r.POST("/v1/thermostats", HandleRoute(PostThermostat))

	// serve api on port 8080
	log.Println("Serving on port :8080")
	if err := fasthttp.ListenAndServe(":8080", r.Handler); err != nil {
		log.Fatalln("failed to serve on port :8080 with error:", err)
	}
}
