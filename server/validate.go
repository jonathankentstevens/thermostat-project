package main

import (
	"net/http"
	"strconv"
)

// inArray determines whether or not a string is in the provided string array
func inArray(char string, strings []string) bool {
	for _, a := range strings {
		if a == char {
			return true
		}
	}
	return false
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
