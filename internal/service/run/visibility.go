package run

import "github.com/mllbll/kosmohak_nn/internal/model"

func clientVisible(sc model.Scenario, snap model.Snapshot, clientID string) bool {
	elev, ok := snap.ElevationDeg[clientID]
	if !ok {
		return hasUplink(snap, clientID)
	}
	minEl := sc.Environment.MinElevationDeg
	for satID, el := range elev {
		if el < minEl {
			continue
		}
		if satelliteActive(snap, satID) {
			return true
		}
	}
	return false
}

func satelliteActive(snap model.Snapshot, satID string) bool {
	for _, sat := range snap.Satellites {
		if sat.ID == satID {
			return sat.Active
		}
	}
	return false
}

func hasUplink(snap model.Snapshot, clientID string) bool {
	for _, e := range snap.Edges {
		if e.A == clientID || e.B == clientID {
			return true
		}
	}
	return false
}

func islEdgeCount(snap model.Snapshot, satIDs map[string]struct{}) int {
	n := 0
	for _, e := range snap.Edges {
		_, aSat := satIDs[e.A]
		_, bSat := satIDs[e.B]
		if aSat && bSat {
			n++
		}
	}
	return n
}

func satelliteIDs(sc model.Scenario) map[string]struct{} {
	out := map[string]struct{}{}
	for _, sat := range sc.Design.Satellites {
		out[sat.ID] = struct{}{}
	}
	return out
}
