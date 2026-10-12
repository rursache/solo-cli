package client

import (
	"time"
	_ "time/tzdata" // Europe/Bucharest must resolve on systems without zoneinfo
)

// bucharest is the time zone SOLO.ro dates and deadlines are in. The embedded
// tzdata means it only stays nil if the name itself stops resolving
var bucharest = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Bucharest")
	if err != nil {
		return nil
	}
	return loc
}()
