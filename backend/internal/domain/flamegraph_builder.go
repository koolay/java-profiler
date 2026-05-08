package domain

import (
	"sort"

	"github.com/koolay/java-profiler/backend/internal/clickhouse"
)

type FlamegraphNode struct {
	Name     string           `json:"name"`
	Value    uint64           `json:"value"`
	Children []FlamegraphNode `json:"children,omitempty"`
}

type FlamegraphMetadata struct {
	Partial        bool     `json:"partial"`
	Reasons        []string `json:"reasons,omitempty"`
	ScannedSamples int      `json:"scanned_samples"`
	OmittedNodes   int      `json:"omitted_nodes"`
}

type FlamegraphResult struct {
	Root     FlamegraphNode     `json:"root"`
	Metadata FlamegraphMetadata `json:"metadata"`
}

func BuildFlamegraph(samples []clickhouse.ProfileSample, nodeLimit int) FlamegraphResult {
	if nodeLimit <= 0 {
		nodeLimit = 2048
	}
	root := FlamegraphNode{Name: "root"}
	nodeCount := 1
	omitted := 0
	for _, sample := range samples {
		root.Value += sample.Value
		children := &root.Children
		for _, frame := range sample.Frames {
			idx := findChild(*children, frame)
			if idx == -1 {
				if nodeCount >= nodeLimit {
					omitted++
					break
				}
				*children = append(*children, FlamegraphNode{Name: frame})
				idx = len(*children) - 1
				nodeCount++
			}
			(*children)[idx].Value += sample.Value
			children = &(*children)[idx].Children
		}
	}
	sortNode(root.Children)
	metadata := FlamegraphMetadata{ScannedSamples: len(samples), OmittedNodes: omitted}
	if omitted > 0 {
		metadata.Partial = true
		metadata.Reasons = []string{"node_limit"}
	}
	return FlamegraphResult{Root: root, Metadata: metadata}
}

func findChild(children []FlamegraphNode, name string) int {
	for i, child := range children {
		if child.Name == name {
			return i
		}
	}
	return -1
}

func sortNode(children []FlamegraphNode) {
	sort.Slice(children, func(i, j int) bool { return children[i].Value > children[j].Value })
	for i := range children {
		sortNode(children[i].Children)
	}
}
