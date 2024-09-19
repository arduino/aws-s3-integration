// This file is part of arduino aws-sitewise-integration.
//
// Copyright 2024 ARDUINO SA (http://www.arduino.cc/)
//
// This software is released under the Mozilla Public License Version 2.0,
// which covers the main part of aws-sitewise-integration.
// The terms of this license can be found at:
// https://www.mozilla.org/media/MPL/2.0/index.815ca599c9df.txt
//
// You can be released from the requirements of the above licenses by purchasing
// a commercial license. Buying such a license is mandatory if you want to
// modify or otherwise use the software for commercial activities involving the
// Arduino software without disclosing the source code of your own applications.
// To purchase a commercial license, send an email to license@arduino.cc.

package iot

type Type string

const (
	Analog                     Type = "ANALOG"
	CharString                 Type = "CHARSTRING"
	Float                      Type = "FLOAT"
	Int                        Type = "INT"
	LenghtC                    Type = "LENGHT_C"
	LenghtI                    Type = "LENGHT_I"
	LenghtM                    Type = "LENGHT_M"
	Percentage                 Type = "PERCENTAGE"
	Status                     Type = "STATUS"
	TemperatureC               Type = "TEMPERATURE_C"
	TemperatureF               Type = "TEMPERATURE_F"
	Meter                      Type = "METER"
	Kilogram                   Type = "KILOGRAM"
	Gram                       Type = "GRAM"
	Second                     Type = "SECOND"
	Ampere                     Type = "AMPERE"
	Kelvin                     Type = "KELVIN"
	Candela                    Type = "CANDELA"
	Mole                       Type = "MOLE"
	Hertz                      Type = "HERTZ"
	Radian                     Type = "RADIAN"
	Steradian                  Type = "STERADIAN"
	Newton                     Type = "NEWTON"
	Pascal                     Type = "PASCAL"
	Joule                      Type = "JOULE"
	Watt                       Type = "WATT"
	Coulomb                    Type = "COULOMB"
	Volt                       Type = "VOLT"
	Farad                      Type = "FARAD"
	Ohm                        Type = "OHM"
	Siemens                    Type = "SIEMENS"
	Weber                      Type = "WEBER"
	Tesla                      Type = "TESLA"
	Henry                      Type = "HENRY"
	DegreesCelsius             Type = "DEGREES_CELSIUS"
	Lumen                      Type = "LUMEN"
	Lux                        Type = "LUX"
	Becquerel                  Type = "BECQUEREL"
	Gray                       Type = "GRAY"
	Sievert                    Type = "SIEVERT"
	Katal                      Type = "KATAL"
	SquareMeter                Type = "SQUARE_METER"
	CubicMeter                 Type = "CUBIC_METER"
	Liter                      Type = "LITER"
	MeterPerSecond             Type = "METER_PER_SECOND"
	MeterPerSquareSecond       Type = "METER_PER_SQUARE_SECOND"
	CubicMeterPerSecond        Type = "CUBIC_METER_PER_SECOND"
	LiterPerSecond             Type = "LITER_PER_SECOND"
	WattPerSquareMeter         Type = "WATT_PER_SQUARE_METER"
	CandelaPerSquareMeter      Type = "CANDELA_PER_SQUARE_METER"
	Bit                        Type = "BIT"
	BitPerSecond               Type = "BIT_PER_SECOND"
	DegreesLatitude            Type = "DEGREES_LATITUDE"
	DegreesLongitude           Type = "DEGREES_LONGITUDE"
	PhValue                    Type = "PH_VALUE"
	Decibel                    Type = "DECIBEL"
	Decibel1w                  Type = "DECIBEL_1W"
	Bel                        Type = "BEL"
	Count                      Type = "COUNT"
	RatioDiv                   Type = "RATIO_DIV"
	RatioMod                   Type = "RATIO_MOD"
	PercentageRelativeHumidity Type = "PERCENTAGE_RELATIVE_HUMIDITY"
	PercentageBatteryLevel     Type = "PERCENTAGE_BATTERY_LEVEL"
	SecondsBatteryLevel        Type = "SECONDS_BATTERY_LEVEL"
	EventRateSecond            Type = "EVENT_RATE_SECOND"
	EventRateMinute            Type = "EVENT_RATE_MINUTE"
	HeartRate                  Type = "HEART_RATE"
	HeartBeats                 Type = "HEART_BEATS"
	SiemensPerMeter            Type = "SIEMENS_PER_METER"
	// Complex properties
	Location               Type = "LOCATION"
	ColorHSB               Type = "COLOR_HSB"
	ColorRGB               Type = "COLOR_RGB"
	GenericComplexProperty      = "GENERIC_COMPLEX_PROPERTY"
	Schedule               Type = "SCHEDULE"
	// Alexa Properties
	HomeColoredLight       = "HOME_COLORED_LIGHT"
	HomeDimmedLight        = "HOME_DIMMED_LIGHT"
	HomeLight         Type = "HOME_LIGHT"
	HomeContactSensor      = "HOME_CONTACT_SENSOR"
	HomeMotionSensor       = "HOME_MOTION_SENSOR"
	HomeSmartPlugType      = "HOME_SMART_PLUG"
	HomeTemperature        = "HOME_TEMPERATURE"
	HomeTemperatureC       = "HOME_TEMPERATURE_C"
	HomeTemperatureF       = "HOME_TEMPERATURE_F"
	HomeSwitch        Type = "HOME_SWITCH"
	HomeTelevision         = "HOME_TELEVISION"
	// New Types based on dimensions
	Energy               Type = "ENERGY"
	Force                Type = "FORCE"
	Temperature          Type = "TEMPERATURE"
	Power                Type = "POWER"
	ElectricCurrent      Type = "ELECTRIC_CURRENT"
	ElectricPotential         = "ELECTRIC_POTENTIAL"
	ElectricalResistance      = "ELECTRICAL_RESISTANCE"
	Capacitance          Type = "CAPACITANCE"
	Time                 Type = "TIME"
	Frequency            Type = "FREQUENCY"
	DataRate             Type = "DATA_RATE"
	Acceleration         Type = "ACCELERATION"
	Area                 Type = "AREA"
	Length               Type = "LENGTH"
	Velocity             Type = "VELOCITY"
	Mass                 Type = "MASS"
	Volume               Type = "VOLUME"
	FlowRate             Type = "FLOW_RATE"
	Angle                Type = "ANGLE"
	Illuminance          Type = "ILLUMINANCE"
	LuminousFlux         Type = "LUMINOUS_FLUX"
	Luminance            Type = "LUMINANCE"
	LuminousIntensity         = "LUMINOUS_INTENSITY"
	LogarithmicQuantity       = "LOGARITHMIC_QUANTITY"
	Pressure             Type = "PRESSURE"
	InformationContent        = "INFORMATION_CONTENT"
)

var floatPropertyTypes = []Type{
	Analog,
	Float,
	LenghtC,
	LenghtI,
	LenghtM,
	Percentage,
	TemperatureC,
	TemperatureF,
	Meter,
	Kilogram,
	Gram,
	Second,
	Ampere,
	Kelvin,
	Candela,
	Mole,
	Hertz,
	Radian,
	Steradian,
	Newton,
	Pascal,
	Joule,
	Watt,
	Coulomb,
	Volt,
	Farad,
	Ohm,
	Siemens,
	Weber,
	Tesla,
	Henry,
	DegreesCelsius,
	Lumen,
	Lux,
	Becquerel,
	Gray,
	Sievert,
	Katal,
	SquareMeter,
	CubicMeter,
	Liter,
	MeterPerSecond,
	MeterPerSquareSecond,
	CubicMeterPerSecond,
	LiterPerSecond,
	WattPerSquareMeter,
	CandelaPerSquareMeter,
	Bit,
	BitPerSecond,
	DegreesLatitude,
	DegreesLongitude,
	PhValue,
	Decibel,
	Decibel1w,
	Bel,
	RatioDiv,
	RatioMod,
	PercentageRelativeHumidity,
	PercentageBatteryLevel,
	SecondsBatteryLevel,
	EventRateSecond,
	EventRateMinute,
	HeartRate,
	HeartBeats,
	SiemensPerMeter,
	HomeTemperature,
	HomeTemperatureC,
	HomeTemperatureF,
	Energy,
	Force,
	Temperature,
	Power,
	ElectricCurrent,
	ElectricPotential,
	ElectricalResistance,
	Capacitance,
	Frequency,
	DataRate,
	Acceleration,
	Area,
	Length,
	Velocity,
	Mass,
	Volume,
	FlowRate,
	Angle,
	Illuminance,
	LuminousFlux,
	Luminance,
	LuminousIntensity,
	LogarithmicQuantity,
	Pressure,
}

var intPropertyTypes = []Type{
	Int,
	Count,
	Time,
	InformationContent,
}

var booleanPropertyTypes = []Type{
	Status,
	HomeLight,
	HomeSwitch,
	HomeContactSensor,
	HomeMotionSensor,
}

func IsPropertyFloat(pType string) bool {
	for _, tpy := range floatPropertyTypes {
		if pType == string(tpy) {
			return true
		}
	}
	return false
}

func IsPropertyInt(pType string) bool {
	for _, tpy := range intPropertyTypes {
		if pType == string(tpy) {
			return true
		}
	}
	return false
}

func IsPropertyNumberType(pType string) bool {
	return IsPropertyFloat(pType) || IsPropertyInt(pType)
}

func IsPropertyString(pType string) bool {
	return pType == "CHARSTRING"
}

func IsPropertyLocation(pType string) bool {
	return Type(pType) == Location
}

func IsPropertyBool(pType string) bool {
	for _, tpy := range booleanPropertyTypes {
		if pType == string(tpy) {
			return true
		}
	}
	return false
}
