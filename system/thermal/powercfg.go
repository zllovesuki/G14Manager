package thermal

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
	GUID     string
	Name     string
	IsActive bool
}

type powercfg struct {
	plansMap   map[string]plan
	toggleMap  map[string]string
	activePlan plan
}

// NewPowerCfg will return an implementation of Powercfg to toggle power plan
func NewPowerCfg(toggleMap map[string]string) (*powercfg, error) {
	cfg := &powercfg{
		plansMap:  make(map[string]plan, 0),
		toggleMap: toggleMap,
	}
	err := cfg.listPowerPlans()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (p *powercfg) Default() {
	p.toggleMap = map[string]string{
		"Power saver":      "High performance",
		"High performance": "Power saver",
	}
}

func (p *powercfg) listPowerPlans() error {
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
			GUID:     match[1],
			Name:     match[3],
			IsActive: match[4] == "*",
		}
		p.plansMap[match[3]] = currentPlan
		if currentPlan.IsActive {
			p.activePlan = currentPlan
		}
	}
	return nil
}

func (p *powercfg) setPowerPlan(active plan) error {
	_, err := run("powercfg", "/S", active.GUID)
	if err != nil {
		log.Printf("cannot set active power plan: %s\n", err)
		return errors.New("Cannot set active power plan")
	}
	p.activePlan = active
	return nil
}

func (p *powercfg) Set(planName string) (nextPlan string, err error) {
	propose, ok := p.plansMap[planName]
	if !ok {
		err = errors.New("Cannot find target power plan")
		return
	}
	err = p.setPowerPlan(propose)
	if err != nil {
		err = errors.New("Cannot set power plan")
		return
	}

	nextPlan = propose.Name

	return
}

func (p *powercfg) Toggle() (nextPlan string, err error) {
	next, ok := p.toggleMap[p.activePlan.Name]
	if !ok {
		err = errors.New("Toggle not found")
		return
	}
	propose, ok := p.plansMap[next]
	if !ok {
		err = errors.New("Cannot find target power plan")
		return
	}
	err = p.setPowerPlan(propose)
	if err != nil {
		err = errors.New("Cannot set power plan")
		return
	}

	nextPlan = propose.Name

	return
}

func run(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	return cmd.Output()
}
