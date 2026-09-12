package run

import (
	"sort"
	"strconv"

	"github.com/mllbll/kosmohak_nn/internal/model"
)

const (
	maxMinHopPaths = 4
	maxBackupPaths = 2
)

func Route(sc model.Scenario, snap model.Snapshot, clientID string) model.RouteResult {
	g := buildRouteGraph(sc, snap)
	algo := routeAlgorithm()

	clientHasUplink := false
	gatewayHasDownlink := false
	for _, e := range snap.Edges {
		if e.A == clientID || e.B == clientID {
			clientHasUplink = true
		}
		if _, ok := g.gateways[e.A]; ok {
			gatewayHasDownlink = true
		}
		if _, ok := g.gateways[e.B]; ok {
			gatewayHasDownlink = true
		}
	}

	minHops := shortestHops(clientID, g)
	if minHops < 0 {
		return model.RouteResult{
			Path:      []string{},
			Reason:    classifyGap(sc, snap.TS, clientHasUplink, gatewayHasDownlink, g.gateways),
			Algorithm: algo,
		}
	}

	primary := enumerateAtHops(clientID, g, minHops, maxMinHopPaths)
	backups := enumerateAtHops(clientID, g, minHops+1, maxBackupPaths)
	if len(primary) == 0 {
		return model.RouteResult{
			Path:      []string{},
			Reason:    classifyGap(sc, snap.TS, clientHasUplink, gatewayHasDownlink, g.gateways),
			Algorithm: algo,
		}
	}

	alts := make([][]string, 0, len(primary)+len(backups)-1)
	alts = append(alts, primary[1:]...)
	alts = append(alts, backups...)
	algo.MinHops = minHops

	return model.RouteResult{
		Path:         copyPath(primary[0]),
		Hops:         minHops,
		Alternatives: alts,
		Algorithm:    algo,
	}
}

func routeAlgorithm() model.RouteAlgorithm {
	return model.RouteAlgorithm{
		Name:      "bfs_min_hops",
		Objective: "минимальное число hops от клиента до шлюза",
		Constraints: []string{
			"клиент соединяется только со спутником",
			"ISL только между активными спутниками",
			"шлюз — стоп маршрута",
			"наземные пункты не ретранслируют трафик",
		},
		Rationale: "На каждом шаге сетки граф меняется целиком, поэтому допустимые маршруты ищутся заново BFS. Критерий — hops, а не километры: каждый hop это отдельный радиоканал и задержка. Запасные пути той же длины и +1 hop показывают обход при отказе узла.",
	}
}

type routeGraph struct {
	adj      map[string][]string
	sats     map[string]struct{}
	gateways map[string]struct{}
	clients  map[string]struct{}
}

func buildRouteGraph(sc model.Scenario, snap model.Snapshot) routeGraph {
	g := routeGraph{
		adj:      map[string][]string{},
		sats:     map[string]struct{}{},
		gateways: map[string]struct{}{},
		clients:  map[string]struct{}{},
	}
	for _, site := range sc.GroundSites {
		if site.Role == "gateway" {
			g.gateways[site.ID] = struct{}{}
		}
		if site.Role == "client" {
			g.clients[site.ID] = struct{}{}
		}
	}
	for _, sat := range snap.Satellites {
		if sat.Active {
			g.sats[sat.ID] = struct{}{}
		}
	}
	for _, e := range snap.Edges {
		g.adj[e.A] = append(g.adj[e.A], e.B)
		g.adj[e.B] = append(g.adj[e.B], e.A)
	}
	for id := range g.adj {
		sort.Strings(g.adj[id])
	}
	return g
}

func allowedHop(from, to string, g routeGraph) bool {
	_, fromSat := g.sats[from]
	_, fromClient := g.clients[from]
	_, toSat := g.sats[to]
	_, toGateway := g.gateways[to]
	_, toClient := g.clients[to]

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

func shortestHops(start string, g routeGraph) int {
	dist := map[string]int{start: 0}
	queue := []string{start}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if _, gw := g.gateways[cur]; gw && cur != start {
			return dist[cur]
		}
		for _, nb := range g.adj[cur] {
			if _, seen := dist[nb]; seen {
				continue
			}
			if !allowedHop(cur, nb, g) {
				continue
			}
			dist[nb] = dist[cur] + 1
			queue = append(queue, nb)
		}
	}
	return -1
}

func enumerateAtHops(start string, g routeGraph, targetHops, limit int) [][]string {
	if targetHops < 1 || limit <= 0 {
		return nil
	}
	out := make([][]string, 0, limit)
	used := map[string]bool{start: true}
	var walk func(cur string, path []string)
	walk = func(cur string, path []string) {
		if len(out) >= limit {
			return
		}
		hops := len(path) - 1
		if hops == targetHops {
			if _, gw := g.gateways[cur]; gw && cur != start {
				out = append(out, append([]string{}, path...))
			}
			return
		}
		for _, nb := range g.adj[cur] {
			if used[nb] || !allowedHop(cur, nb, g) {
				continue
			}
			used[nb] = true
			walk(nb, append(path, nb))
			used[nb] = false
		}
	}
	walk(start, []string{start})
	return out
}

func pathStillValid(path []string, g routeGraph) bool {
	if len(path) < 2 {
		return false
	}
	if _, ok := g.clients[path[0]]; !ok {
		return false
	}
	if _, ok := g.gateways[path[len(path)-1]]; !ok {
		return false
	}
	for i := 0; i < len(path)-1; i++ {
		from, to := path[i], path[i+1]
		if !allowedHop(from, to, g) || !adjacent(g.adj, from, to) {
			return false
		}
	}
	return true
}

func adjacent(adj map[string][]string, a, b string) bool {
	for _, nb := range adj[a] {
		if nb == b {
			return true
		}
	}
	return false
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

func networkDelta(sc model.Scenario, snap model.Snapshot, prev []string, prevFound bool, current model.RouteResult) model.NetworkDelta {
	delta := model.NetworkDelta{
		PreviousPath: copyPath(prev),
		CurrentPath:  copyPath(current.Path),
	}
	if !prevFound {
		if len(current.Path) == 0 {
			delta.Explanation = "Первый шаг сетки: допустимого маршрута нет (" + gapTitle(current.Reason) + "). Алгоритм: BFS min hops, маршруты ищутся заново на каждом t"
			return delta
		}
		delta.Explanation = "Первый шаг сетки: маршрут построен с нуля BFS (min hops). Запасные пути: " + strconv.Itoa(len(current.Alternatives))
		return delta
	}
	if len(prev) == 0 {
		delta.Changed = len(current.Path) > 0
		if len(current.Path) == 0 {
			delta.Explanation = "На предыдущем шаге маршрута не было, разрыв сохраняется (" + gapTitle(current.Reason) + ")"
			return delta
		}
		delta.Explanation = "На предыдущем шаге маршрута не было, сейчас BFS нашёл допустимый путь"
		return delta
	}

	g := buildRouteGraph(sc, snap)
	delta.PreviousStillValid = pathStillValid(prev, g)
	same := pathsEqual(prev, current.Path)
	delta.Changed = !same

	switch {
	case same && delta.PreviousStillValid:
		delta.Explanation = "Маршрут совпадает с предыдущим шагом и остаётся допустимым"
	case delta.PreviousStillValid && !same:
		delta.Explanation = "Предыдущий путь ещё допустим, но BFS выбрал другой min-hop маршрут"
	case !delta.PreviousStillValid && len(current.Path) > 0:
		delta.Explanation = "Предыдущий путь стал недопустимым (ребро или узел пропали). Маршрут перестроен BFS по текущему снимку"
	case !delta.PreviousStillValid && len(current.Path) == 0:
		delta.Explanation = "Предыдущий путь разорван, нового допустимого маршрута нет (" + gapTitle(current.Reason) + ")"
	default:
		delta.Explanation = "Маршрут пересчитан BFS на текущем состоянии сети"
	}
	return delta
}

func pathsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
