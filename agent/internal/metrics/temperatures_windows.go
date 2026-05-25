//go:build windows

package metrics

import (
	"math"
	"strings"

	"github.com/yusufpapurcu/wmi"
)

type hardwareMonitorSensor struct {
	Name       string
	SensorType string
	Value      float64
	Identifier string
	Parent     string
}

type acpiThermalZone struct {
	InstanceName       string
	CurrentTemperature uint32
}

func collectTemperatures() *TempInfo {
	sensors := collectHardwareMonitorTemperatures()
	if len(sensors) == 0 {
		sensors = collectACPITemperatures()
	}
	if len(sensors) == 0 {
		return nil
	}

	info := &TempInfo{Sensors: sensors}
	if cpu, ok := pickTemperature(sensors, isCPUTempSensor); ok {
		info.CPUCelsius = &cpu
	}
	if gpu, ok := pickTemperature(sensors, isGPUTempSensor); ok {
		info.GPUCelsius = &gpu
	}
	return info
}

func collectHardwareMonitorTemperatures() []TempSensor {
	namespaces := []string{
		`root\LibreHardwareMonitor`,
		`root\OpenHardwareMonitor`,
	}
	var sensors []TempSensor
	for _, namespace := range namespaces {
		var rows []hardwareMonitorSensor
		err := wmi.QueryNamespace("SELECT Name, SensorType, Value, Identifier, Parent FROM Sensor WHERE SensorType = 'Temperature'", &rows, namespace)
		if err != nil {
			rows = nil
			err = wmi.QueryNamespace("SELECT Name, SensorType, Value, Identifier FROM Sensor WHERE SensorType = 'Temperature'", &rows, namespace)
			if err != nil {
				continue
			}
		}
		for _, row := range rows {
			if !validTemperature(row.Value) {
				continue
			}
			sensors = append(sensors, TempSensor{
				Name:     strings.TrimSpace(row.Name),
				Type:     strings.TrimSpace(row.SensorType),
				Value:    round1(row.Value),
				Source:   namespace,
				Hardware: strings.TrimSpace(row.Identifier + " " + row.Parent),
			})
		}
	}
	return sensors
}

func collectACPITemperatures() []TempSensor {
	var rows []acpiThermalZone
	if err := wmi.QueryNamespace("SELECT InstanceName, CurrentTemperature FROM MSAcpi_ThermalZoneTemperature", &rows, `root\WMI`); err != nil {
		return nil
	}
	sensors := make([]TempSensor, 0, len(rows))
	for _, row := range rows {
		celsius := float64(row.CurrentTemperature)/10 - 273.15
		if !validTemperature(celsius) {
			continue
		}
		sensors = append(sensors, TempSensor{
			Name:   strings.TrimSpace(row.InstanceName),
			Type:   "Temperature",
			Value:  round1(celsius),
			Source: `root\WMI`,
		})
	}
	return sensors
}

func pickTemperature(sensors []TempSensor, match func(TempSensor) bool) (float64, bool) {
	var fallback *float64
	for _, sensor := range sensors {
		if !match(sensor) {
			continue
		}
		value := sensor.Value
		if isPreferredTemperatureName(sensor.Name) {
			return value, true
		}
		if fallback == nil {
			fallback = &value
		}
	}
	if fallback != nil {
		return *fallback, true
	}
	return 0, false
}

func isCPUTempSensor(sensor TempSensor) bool {
	text := strings.ToLower(sensor.Name + " " + sensor.Hardware)
	return strings.Contains(text, "cpu") ||
		strings.Contains(text, "amdcpu") ||
		strings.Contains(text, "intelcpu") ||
		strings.Contains(text, "tctl") ||
		strings.Contains(text, "tdie")
}

func isGPUTempSensor(sensor TempSensor) bool {
	text := strings.ToLower(sensor.Name + " " + sensor.Hardware)
	return strings.Contains(text, "gpu") ||
		strings.Contains(text, "nvidia") ||
		strings.Contains(text, "geforce") ||
		strings.Contains(text, "radeon")
}

func isPreferredTemperatureName(name string) bool {
	name = strings.ToLower(name)
	return strings.Contains(name, "package") ||
		strings.Contains(name, "core") ||
		strings.Contains(name, "tctl") ||
		strings.Contains(name, "tdie") ||
		strings.Contains(name, "hot spot")
}

func validTemperature(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0) && value > -40 && value < 130
}
