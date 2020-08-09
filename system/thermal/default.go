package thermal

import "strings"

// GetDefaultThermalProfiles will return the default list of Profiles
func GetDefaultThermalProfiles() []Profile {
	defaultProfiles := make([]Profile, 0, 3)
	defaults := []struct {
		name             string
		windowsPowerPlan string
		throttlePlan     byte
		cpuFanCurve      string
		gpuFanCurve      string
	}{
		{
			name:             "Fanless",
			windowsPowerPlan: "Power saver",
			throttlePlan:     throttlePlanSilent,
			cpuFanCurve:      "39c:0%,49c:0%,59c:0%,69c:0%,79c:31%,89c:49%,99c:56%,109c:56%",
			gpuFanCurve:      "39c:0%,49c:0%,59c:0%,69c:0%,79c:34%,89c:51%,99c:61%,109c:61%",
		},
		{
			name:             "Quiet",
			windowsPowerPlan: "Power saver",
			throttlePlan:     throttlePlanSilent,
			cpuFanCurve:      "39c:10%,49c:10%,59c:10%,69c:10%,79c:31%,89c:49%,99c:56%,109c:56%",
			gpuFanCurve:      "39c:0%,49c:0%,59c:0%,69c:0%,79c:34%,89c:51%,99c:61%,109c:61%",
		},
		{
			name:             "Silent",
			windowsPowerPlan: "Power saver",
			throttlePlan:     throttlePlanSilent,
		},
		{
			name:             "Performance",
			windowsPowerPlan: "High performance",
			throttlePlan:     throttlePlanPerformance,
		},
		/*{
			name:             "Turbo",
			windowsPowerPlan: "High performance",
			throttlePlan:     throttlePlanTurbo,
		},*/
	}
	for _, d := range defaults {
		var cpuTable, gpuTable *fanTable
		var err error
		profile := Profile{
			Name:             d.name,
			ThrottlePlan:     d.throttlePlan,
			WindowsPowerPlan: strings.ToLower(d.windowsPowerPlan),
		}
		if d.cpuFanCurve != "" {
			cpuTable, err = newFanTable(d.cpuFanCurve)
			if err != nil {
				panic(err)
			}
			profile.CPUFanCurve = cpuTable
		}
		if d.gpuFanCurve != "" {
			gpuTable, err = newFanTable(d.gpuFanCurve)
			if err != nil {
				panic(err)
			}
			profile.GPUFanCurve = gpuTable
		}
		defaultProfiles = append(defaultProfiles, profile)
	}
	return defaultProfiles
}
