package domain

import (
	"sort"

	rootdomain "github.com/koolay/java-profiler/domain"
)

type FlamegraphNode struct {
	Name         string           `json:"name"`
	Value        uint64           `json:"value"`
	DisplayValue string           `json:"display_value,omitempty"`
	Children     []FlamegraphNode `json:"children,omitempty"`
	childIndex   map[string]int   `json:"-"`
}

type FlamegraphMetadata struct {
	Partial        bool     `json:"partial"`
	Reasons        []string `json:"reasons,omitempty"`
	ScannedSamples int      `json:"scanned_samples"`
	OmittedNodes   int      `json:"omitted_nodes"`
}

type FlamegraphResult struct {
	Root      FlamegraphNode                   `json:"root"`
	Metadata  FlamegraphMetadata               `json:"metadata"`
	Semantics rootdomain.ProfileValueSemantics `json:"semantics"`
}

type FlamegraphSample struct {
	Frames []string
	Value  uint64
}

func BuildFlamegraph(samples []FlamegraphSample, nodeLimit int) FlamegraphResult {
	if nodeLimit <= 0 {
		nodeLimit = 2048
	}
	root := FlamegraphNode{Name: "root", childIndex: map[string]int{}}
	nodeCount := 1
	omitted := 0
	for _, sample := range samples {
		root.Value += sample.Value
		children := &root
		for _, frame := range sample.Frames {
			idx, ok := children.childIndex[frame]
			if !ok {
				if nodeCount >= nodeLimit {
					omitted++
					break
				}
				children.Children = append(children.Children, FlamegraphNode{Name: frame, childIndex: map[string]int{}})
				idx = len(children.Children) - 1
				if children.childIndex == nil {
					children.childIndex = map[string]int{}
				}
				children.childIndex[frame] = idx
				nodeCount++
			}
			children.Children[idx].Value += sample.Value
			children = &children.Children[idx]
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

func ApplyProfileSemantics(result FlamegraphResult, profileType rootdomain.ProfileType, window rootdomain.TimeWindow) FlamegraphResult {
	result.Semantics = profileType.Semantics(window)
	applyDisplayValues(&result.Root, profileType, window)
	return result
}

func applyDisplayValues(node *FlamegraphNode, profileType rootdomain.ProfileType, window rootdomain.TimeWindow) {
	node.DisplayValue = rootdomain.FormatProfileValue(profileType, node.Value, window)
	for i := range node.Children {
		applyDisplayValues(&node.Children[i], profileType, window)
	}
}

func sortNode(children []FlamegraphNode) {
	sort.Slice(children, func(i, j int) bool { return children[i].Value > children[j].Value })
	for i := range children {
		sortNode(children[i].Children)
	}
}
