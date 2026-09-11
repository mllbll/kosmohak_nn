package run

import (
	"github.com/mllbll/kosmohak_nn/internal/model"
)

func Route(sc model.Scenario, snap model.Snapshot, clientID string) model.RouteResult {
	gateways := map[string]struct{}{}
	clients := map[string]struct{}{}
	sats := map[string]struct{}{}

	for _, g := range sc.GroundSites {
		if g.Role == "gateway" {
			gateways[g.ID] = struct{}{}
		}
		if g.Role == "client" {
			clients[g.ID] = struct{}{}
		}
	}
	for _, sat := range snap.Satellites {
		if sat.Active {
			sats[sat.ID] = struct{}{}
		}
	}

	adj := map[string][]string{}
	clientHasUplink := false
	gatewayHasDownlink := false
	for _, e := range snap.Edges {
		adj[e.A] = append(adj[e.A], e.B)
		adj[e.B] = append(adj[e.B], e.A)
		if e.A == clientID || e.B == clientID {
			clientHasUplink = true
		}
		if _, ok := gateways[e.A]; ok {
			gatewayHasDownlink = true
		}
		if _, ok := gateways[e.B]; ok {
			gatewayHasDownlink = true
		}
	}

	path := bfs(clientID, adj, sats, gateways, clients)
	if len(path) > 0 {
		return model.RouteResult{Path: path, Hops: len(path) - 1}
	}

	return model.RouteResult{
		Path:   []string{},
		Reason: classifyGap(sc, snap.TS, clientHasUplink, gatewayHasDownlink, gateways),
	}
}

func allowedHop(from, to string, sats, gateways, clients map[string]struct{}) bool {
	_, fromSat := sats[from]
	_, fromClient := clients[from]
	_, toSat := sats[to]
	_, toGateway := gateways[to]
	_, toClient := clients[to]

	if toClient {
		return false
	}
	if fromClient {
		return toSat
	}
	if fromSat {
		return toSat || toGateway
	}
	return false
}

func bfs(start string, adj map[string][]string, sats, gateways, clients map[string]struct{}) []string {
	parent := map[string]string{start: ""}
	queue := []string{start}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, nb := range adj[cur] {
			if _, seen := parent[nb]; seen {
				continue
			}
			if !allowedHop(cur, nb, sats, gateways, clients) {
				continue
			}
			parent[nb] = cur
			if _, ok := gateways[nb]; ok {
				return restorePath(parent, start, nb)
			}
			queue = append(queue, nb)
		}
	}
	return nil
}

func restorePath(parent map[string]string, start, end string) []string {
	cur := end
	rev := []string{cur}
	for cur != start {
		cur = parent[cur]
		rev = append(rev, cur)
	}
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	return rev
}

func classifyGap(sc model.Scenario, t float64, clientHasUplink, gatewayHasDownlink bool, gateways map[string]struct{}) model.GapReason {
	allGatewaysDown := len(gateways) > 0
	for id := range gateways {
		if !sc.IsGatewayOutaged(id, t) {
			allGatewaysDown = false
			break
		}
	}
	if allGatewaysDown {
		return model.GapGatewayOutage
	}
	if !clientHasUplink {
		return model.GapNoVisibleSat
	}
	if !gatewayHasDownlink {
		return model.GapNoGatewayContact
	}
	return model.GapISLPartition
}
