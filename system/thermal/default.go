package thermal

// GetDefaultThermalProfiles will return the default list of Profiles
func GetDefaultThermalProfiles() []Profile {
	defaultProfiles := make([]Profile, 0, 3)
	defaults := []struct {
		name             string
		windowsPowerPlan string
		throttlePlan     uint32
		cpuFanCurve      string
		gpuFanCurve      string
	}{
		{
			name:             "Fanless",
			windowsPowerPlan: "Power saver",
			throttlePlan:     ThrottlePlanPerformance,
			cpuFanCurve:      "20c:0%,50c:0%,55c:0%,60c:0%,65c:31%,70c:49%,75c:56%,98c:56%",
			gpuFanCurve:      "20c:0%,50c:0%,55c:0%,60c:0%,65c:34%,70c:51%,75c:61%,98c:61%",
		},
		{
			name:             "Quiet",
			windowsPowerPlan: "Power saver",
			throttlePlan:     ThrottlePlanPerformance,
			cpuFanCurve:      "20c:10%,50c:10%,55c:10%,60c:10%,65c:31%,70c:49%,75c:56%,98c:56%",
			gpuFanCurve:      "20c:0%,50c:0%,55c:0%,60c:0%,65c:34%,70c:51%,75c:61%,98c:61%",
		},
		{
			name:             "Power Saver",
			windowsPowerPlan: "Power saver",
			throttlePlan:     ThrottlePlanSilent,
		},
		{
			name:             "Silent Performance",
			windowsPowerPlan: "High performance",
			throttlePlan:     ThrottlePlanSilent,
		},
		{
			name:             "Performance",
			windowsPowerPlan: "High performance",
			throttlePlan:     ThrottlePlanPerformance,
		},
		{
			name:             "Turbo",
			windowsPowerPlan: "High performance",
			throttlePlan:     ThrottlePlanTurbo,
		},
	}
	for _, d := range defaults {
		var cpuTable, gpuTable *FanTable
		var err error
		profile := Profile{
			Name:             d.name,
			ThrottlePlan:     d.throttlePlan,
			WindowsPowerPlan: d.windowsPowerPlan,
		}
		if d.cpuFanCurve != "" {
			cpuTable, err = NewFanTable(d.cpuFanCurve)
			if err != nil {
				panic(err)
			}
			profile.CPUFanCurve = cpuTable
		}
		if d.gpuFanCurve != "" {
			gpuTable, err = NewFanTable(d.gpuFanCurve)
			if err != nil {
				panic(err)
			}
			profile.GPUFanCurve = gpuTable
		}
		defaultProfiles = append(defaultProfiles, profile)
	}
	return defaultProfiles
}
