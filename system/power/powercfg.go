package power

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"os/exec"
	"regexp"
	"strings"
	"syscall"

	"github.com/zllovesuki/G14Manager/system/persist"
)

const (
	powerPersistKey = "PowerCfg"
)

var (
	powerCfgRe = regexp.MustCompile(`(?P<GUID>[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12})  (?P<Name>\((.*?)\))\s?(?P<Active>\*)?`)
)

type plan struct {
	GUID         string
	OriginalName string
	Name         string
}

// Cfg allows the caller to change the Power Plan Option in Windows
type Cfg struct {
	plansMap   map[string]plan
	activePlan plan
}

// NewCfg will return a Cfg allowing you to modify the Windows Power Option
func NewCfg() (*Cfg, error) {
	cfg := &Cfg{
		plansMap: make(map[string]plan, 0),
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
			GUID:         match[1],
			OriginalName: match[3],
			Name:         strings.ToLower(match[3]),
		}
		p.plansMap[currentPlan.Name] = currentPlan
	}
	return nil
}

func (p *Cfg) setPowerPlan(active plan) error {
	_, err := run("powercfg", "/S", active.GUID)
	if err != nil {
		log.Printf("cannot set active power plan: %s\n", err)
		return errors.New("Cannot set active power plan")
	}
	p.activePlan = active
	return nil
}

// Set will change the Windows Power Option to the given power plan name
func (p *Cfg) Set(planName string) (nextPlan string, err error) {
	propose, ok := p.plansMap[planName]
	if !ok {
		err = errors.New("Cannot find target power plan")
		return
	}

	if p.activePlan.GUID == propose.GUID {
		// don't apply unnecessary changes
		return p.activePlan.Name, nil
	}

	err = p.setPowerPlan(propose)
	if err != nil {
		err = errors.New("Cannot set power plan")
		return
	}

	nextPlan = propose.OriginalName

	log.Printf("windows power plan set to: %s\n", nextPlan)

	return
}

var _ persist.Registry = &Cfg{}

// Name satisfies persist.Registry
func (p *Cfg) Name() string {
	return powerPersistKey
}

// Value satisfies persist.Registry
func (p *Cfg) Value() []byte {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(p.activePlan); err != nil {
		return nil
	}
	return buf.Bytes()
}

// Load satisfies persist.Registry
func (p *Cfg) Load(v []byte) error {
	if len(v) == 0 {
		return nil
	}
	activePlan := plan{}
	buf := bytes.NewBuffer(v)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&activePlan); err != nil {
		return err
	}
	p.activePlan = activePlan
	return nil
}

// Apply satisfies persist.Registry
func (p *Cfg) Apply() error {
	if p.activePlan.Name == "" {
		return nil
	}
	_, err := p.Set(p.activePlan.Name)
	return err
}

// Close satisfied persist.Registry
func (p *Cfg) Close() error {
	return nil
}

// run will attempt to execute in command line without showing the console window
func run(command string, args ...string) ([]byte, error) {
	cmd := exec.Command(command, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x08000000}
	return cmd.Output()
}
