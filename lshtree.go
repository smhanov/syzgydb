package syzgydb

import (
	"math"
	"math/rand"
)

func normalizeVector(vector []float64) []float64 {
	norm := 0.0
	for _, v := range vector {
		norm += v * v
	}
	norm = math.Sqrt(norm)

	if norm == 0 {
		return make([]float64, len(vector)) // Return a zero vector if the input is all zeros
	}

	normalized := make([]float64, len(vector))
	for i, v := range vector {
		normalized[i] = v / norm
	}
	return normalized
}

func randomNormalizedVector(dim int) []float64 {
	vector := make([]float64, dim)
	for i := range vector {
		vector[i] = rand.NormFloat64()
	}
	return normalizeVector(vector)
}

type lshNode struct {
	normal []float64
	b      float64
	left, right *lshNode
	ids    []uint64
}

func (n *lshNode) isLeaf() bool {
	return n.left == nil
}

func distanceToHyperplane(vector, normal []float64, b float64) float64 {
	dotProduct := 0.0
	for i, v := range vector {
		dotProduct += v * normal[i]
	}
	return dotProduct - b
}

type lshTree struct {
	root *lshNode
	threshold int
}

func newLSHTree(threshold int) *lshTree {
	return &lshTree{
		root: &lshNode{ids: []uint64{}},
		threshold: threshold,
	}
}

func (tree *lshTree) addPoint(docid uint64, vector []float64) {
	tree.root = tree.insert(tree.root, docid, vector)
}

func (tree *lshTree) insert(node *lshNode, docid uint64, vector []float64) *lshNode {
	if node.isLeaf() {
		node.ids = append(node.ids, docid)
		if len(node.ids) > tree.threshold {
			return tree.split(node, vector)
		}
		return node
	}

	distance := distanceToHyperplane(vector, node.normal, node.b)
	if distance < 0 {
		node.left = tree.insert(node.left, docid, vector)
	} else {
		node.right = tree.insert(node.right, docid, vector)
	}
	return node
}

func (tree *lshTree) split(node *lshNode, vector []float64) *lshNode {
	randomIndex := rand.Intn(len(node.ids))
	randomVector := vector // Assume we have a way to get the vector by ID
	normal := randomNormalizedVector(len(randomVector))
	b := math.Sqrt(dotProduct(randomVector, randomVector))

	leftIDs := []uint64{}
	rightIDs := []uint64{}

	for _, id := range node.ids {
		// Assume we have a way to get the vector by ID
		v := vector
		if distanceToHyperplane(v, normal, b) < 0 {
			leftIDs = append(leftIDs, id)
		} else {
			rightIDs = append(rightIDs, id)
		}
	}

	if len(leftIDs) == 0 || len(rightIDs) == 0 {
		return node // Avoid splitting if all vectors are on one side
	}

	return &lshNode{
		normal: normal,
		b:      b,
		left:   &lshNode{ids: leftIDs},
		right:  &lshNode{ids: rightIDs},
	}
}

func (tree *lshTree) removePoint(docid uint64, vector []float64) {
	// Implement removal logic if needed
}

func (tree *lshTree) search(vector []float64, callback func(docid uint64) float64) {
	tree.searchNode(tree.root, vector, callback, math.MaxFloat64)
}

func (tree *lshTree) searchNode(node *lshNode, vector []float64, callback func(docid uint64) float64, bestDistance float64) {
	if node.isLeaf() {
		for _, id := range node.ids {
			distance := callback(id)
			if distance < bestDistance {
				bestDistance = distance
			}
		}
		return
	}

	distance := distanceToHyperplane(vector, node.normal, node.b)
	if distance < 0 {
		tree.searchNode(node.left, vector, callback, bestDistance)
		if math.Abs(distance) < bestDistance {
			tree.searchNode(node.right, vector, callback, bestDistance)
		}
	} else {
		tree.searchNode(node.right, vector, callback, bestDistance)
		if math.Abs(distance) < bestDistance {
			tree.searchNode(node.left, vector, callback, bestDistance)
		}
	}
}
