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
	GUID         string
	OriginalName string
	Name         string
	IsActive     bool
}

type Powercfg struct {
	plansMap   map[string]plan
	activePlan plan
}

// NewPowerCfg will return an implementation of Powercfg to change power plan
func NewPowerCfg() (*Powercfg, error) {
	cfg := &Powercfg{
		plansMap: make(map[string]plan, 0),
	}
	err := cfg.listPowerPlans()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (p *Powercfg) listPowerPlans() error {
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
			GUID:         match[1],
			OriginalName: match[3],
			Name:         strings.ToLower(match[3]),
			IsActive:     match[4] == "*",
		}
		p.plansMap[currentPlan.Name] = currentPlan
		if currentPlan.IsActive {
			p.activePlan = currentPlan
		}
	}
	return nil
}

func (p *Powercfg) setPowerPlan(active plan) error {
	_, err := run("powercfg", "/S", active.GUID)
	if err != nil {
		log.Printf("cannot set active power plan: %s\n", err)
		return errors.New("Cannot set active power plan")
	}
	p.activePlan = active
	return nil
}

func (p *Powercfg) Set(planName string) (nextPlan string, err error) {
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

	nextPlan = propose.OriginalName

	return
}

func run(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	return cmd.Output()
}
