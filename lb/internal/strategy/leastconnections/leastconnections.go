package leastconnectionsgo

type LeastConnections struct {
	connections map[string]int
}

func NewLeastConnections() *LeastConnections {
	return &LeastConnections{
		connections: make(map[string]int),
	}
}

func (lc *LeastConnections) Next(instances []string) string {
	if len(instances) == 0 {
		return ""
	}

	for _, inst := range instances {
		if _, exists := lc.connections[inst]; !exists {
			lc.connections[inst] = 0
		}
	}

	minInst := instances[0]
	minConn := lc.connections[minInst]

	for _, inst := range instances[1:] {
		if lc.connections[inst] < minConn {
			minConn = lc.connections[inst]
			minInst = inst
		}
	}

	lc.connections[minInst]++

	return minInst
}

func (lc *LeastConnections) Done(instance string) {
	if lc.connections[instance] > 0 {
		lc.connections[instance]--
	}
}
