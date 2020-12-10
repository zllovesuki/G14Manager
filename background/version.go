package background

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/Masterminds/semver"
	"github.com/zllovesuki/G14Manager/system/shared"
	"github.com/zllovesuki/G14Manager/util"
)

type VersionChecker struct {
	current  *semver.Version
	repo     string
	tick     chan time.Time
	notifier chan<- util.Notification
}

type release struct {
	TagName string `json:"tag_name"`
}

func NewVersionCheck(current string, repo string, notifier chan<- util.Notification) (*VersionChecker, error) {
	sem, err := semver.NewVersion(current)
	if err != nil {
		return nil, err
	}
	tick := make(chan time.Time, 1)
	tick <- time.Now()

	return &VersionChecker{
		current:  sem,
		repo:     repo,
		tick:     tick,
		notifier: notifier,
	}, nil
}

func (v *VersionChecker) String() string {
	return "VersionChecker"
}

func (v *VersionChecker) Serve(haltCtx context.Context) error {
	log.Println("[VersionChecker] starting checker loop")

	go func() {
		ticker := time.NewTicker(time.Hour * 6)
		defer ticker.Stop()
		for {
			select {
			case t := <-ticker.C:
				v.tick <- t
			case <-haltCtx.Done():
				return
			}
		}
	}()

	for {
		select {
		case <-haltCtx.Done():
			log.Println("[VersionChecker] stopping checker loop")
			return nil
		case <-v.tick:
			log.Println("[VersionChecker] checking for new version")
			latest, err := v.getLatest()
			if err != nil {
				log.Printf("[VersionChecker] error checking for new version: %+v\n", err)
			}
			if latest.GreaterThan(v.current) {
				log.Printf("[VersionChecker] new version found: %s\n", latest.String())
				v.notifier <- util.Notification{
					AppName: shared.AppName,
					Title:   "New Version Available",
					Message: fmt.Sprintf("A new version of G14Manager is available: %s", latest.String()),
				}
			}
		}
	}
}

func (v *VersionChecker) getLatest() (*semver.Version, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", v.repo)
	client := http.Client{
		Timeout: time.Second * 5,
	}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	res, getErr := client.Do(req)
	if getErr != nil {
		return nil, getErr
	}
	defer res.Body.Close()

	var r release
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		return nil, err
	}

	return semver.NewVersion(r.TagName)
}
