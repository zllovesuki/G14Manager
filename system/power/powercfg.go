package power

import (
	"errors"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
)

var (
	powerCfgRe = regexp.MustCompile(`(?P<GUID>[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12})  (?P<Name>\((.*?)\))\s?(?P<Active>\*)?`)
)

type plan struct {
	GUID           string
	Name           string
	normalizedName string
}

// Cfg allows the caller to change the Power Plan Option in Windows
type Cfg struct {
	plansMap   map[string]plan
	activePlan plan
}

// NewCfg will return a Cfg allowing you to modify the Windows Power Option
func NewCfg() (*Cfg, error) {
	cfg := &Cfg{
		plansMap: make(map[string]plan),
	}
	err := cfg.loadPowerPlans()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (p *Cfg) loadPowerPlans() error {
	powerCfgOut, err := run("powercfg", "/l")
	if err != nil {
		log.Printf("cannot list power plans: %s\n", err)
		return err
	}
	lines := strings.Split(string(powerCfgOut), "\n")
	for _, line := range lines {
		match := powerCfgRe.FindStringSubmatch(line)
		if len(match) == 0 {
			continue
		}
		currentPlan := plan{
			GUID:           match[1],
			Name:           match[3],
			normalizedName: strings.ToLower(match[3]),
		}
		p.plansMap[currentPlan.normalizedName] = currentPlan
	}
	return nil
}

func (p *Cfg) setPowerPlan(active plan) error {
	_, err := run("powercfg", "/S", active.GUID)
	if err != nil {
		log.Printf("cannot set active power plan: %s\n", err)
		return errors.New("cannot set active power plan")
	}
	p.activePlan = active
	return nil
}

// Set will change the Windows Power Option to the given power plan name
func (p *Cfg) Set(planName string) (nextPlan string, err error) {
	propose, ok := p.plansMap[strings.ToLower(planName)]
	if !ok {
		err = errors.New("cannot find target power plan")
		return
	}

	if p.activePlan.GUID == propose.GUID {
		// don't apply unnecessary changes
		return p.activePlan.Name, nil
	}

	err = p.setPowerPlan(propose)
	if err != nil {
		err = errors.New("cannot set power plan")
		return
	}

	nextPlan = propose.Name

	log.Printf("windows power plan set to: %s\n", nextPlan)

	return
}

// run will attempt to execute in command line without showing the console window
func run(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	return cmd.Output()
}
