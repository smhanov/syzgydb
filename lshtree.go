package syzgydb

import (
	"log"
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
	normal      []float64
	b           float64
	left, right *lshNode
	ids         []uint64
}

func (n *lshNode) isLeaf() bool {
	return n.left == nil
}

func distanceToHyperplane(vector, normal []float64, b float64) float64 {
	return dotProduct(vector, normal) - b
}

type lshTree struct {
	root      *lshNode
	threshold int
	c         *Collection
}

func newLSHTree(c *Collection, threshold int) *lshTree {
	return &lshTree{
		root:      &lshNode{ids: []uint64{}},
		threshold: threshold,
		c:         c,
	}
}

func (tree *lshTree) addPoint(docid uint64, vector []float64) {
	tree.root = tree.insert(tree.root, docid, vector)
}

func (tree *lshTree) insert(node *lshNode, docid uint64, vector []float64) *lshNode {
	//log.Printf("Inserting %v", docid)
	if node.isLeaf() {
		node.ids = append(node.ids, docid)
		//log.Printf("ids: %v threshold: %v", len(node.ids), tree.threshold)
		if len(node.ids) > tree.threshold {
			node = tree.split(node)
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

func dotProduct(vector1, vector2 []float64) float64 {
	if len(vector1) != len(vector2) {
		panic("vectors must be of the same length")
	}
	dot := 0.0
	for i := range vector1 {
		dot += vector1[i] * vector2[i]
	}
	return dot
}

func midpoint(vector1, vector2 []float64) []float64 {
	if len(vector1) != len(vector2) {
		panic("vectors must be of the same length")
	}
	mid := make([]float64, len(vector1))
	for i := range vector1 {
		mid[i] = (vector1[i] + vector2[i]) / 2
	}
	return mid
}

const tolerance = 1e-9

func aboutEqual(vector1, vector2 []float64) bool {
	if len(vector1) != len(vector2) {
		return false
	}
	for i := range vector1 {
		if math.Abs(vector1[i]-vector2[i]) > tolerance {
			return false
		}
	}
	return true
}

func (tree *lshTree) split(node *lshNode) *lshNode {
	randomIndex1 := rand.Intn(len(node.ids))
	var randomIndex2 int
	for {
		randomIndex2 = rand.Intn(len(node.ids))
		if randomIndex2 != randomIndex1 {
			break
		}
	}

	//log.Printf("Splitting on %v/%v", node.ids[randomIndex1], node.ids[randomIndex2])

	doc1, err := tree.c.getDocument(node.ids[randomIndex1])

	if err != nil {
		log.Panicf("error getting document: %v", err)
	}

	doc2, err := tree.c.getDocument(node.ids[randomIndex2])

	if err != nil {
		log.Panicf("error getting document: %v", err)
	}

	if aboutEqual(doc1.Vector, doc2.Vector) {
		// Avoid splitting if the two vectors are the same
		// Maybe all of them are the same? In any case we will try again next time.
		return node
	}

	pointChosen := midpoint(doc1.Vector, doc2.Vector)
	normal := randomNormalizedVector(len(pointChosen))

	var b float64
	if tree.c.DistanceMethod == Euclidean {
		b = math.Sqrt(dotProduct(pointChosen, pointChosen))
	}

	leftIDs := []uint64{}
	rightIDs := []uint64{}

	for _, id := range node.ids {
		doc, err := tree.c.getDocument(id)
		if err != nil {
			log.Panicf("error getting document: %v", err)
		}

		v := doc.Vector

		d := distanceToHyperplane(v, normal, b)
		if d < 0 {
			leftIDs = append(leftIDs, id)
		} else {
			rightIDs = append(rightIDs, id)
		}
	}

	//log.Printf("    leftCount: %v, rightCount: %v", len(leftIDs), len(rightIDs))

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
	tree.root = tree.remove(tree.root, docid, vector)
}

func (tree *lshTree) remove(node *lshNode, docid uint64, vector []float64) *lshNode {
	if node.isLeaf() {
		// Remove the document ID from the list of IDs
		for i, id := range node.ids {
			if id == docid {
				node.ids = append(node.ids[:i], node.ids[i+1:]...)
				break
			}
		}
		// Optionally handle empty nodes here
		return node
	}

	// Traverse the tree based on the vector's position relative to the hyperplane
	distance := distanceToHyperplane(vector, node.normal, node.b)
	if distance < 0 {
		node.left = tree.remove(node.left, docid, vector)
	} else {
		node.right = tree.remove(node.right, docid, vector)
	}
	return node
}

func (tree *lshTree) search(vector []float64, callback func(docid uint64) float64) {
	tau := math.MaxFloat64
	tree.searchNode(tree.root, vector, callback, &tau)
}

func (tree *lshTree) searchNode(node *lshNode, vector []float64, callback func(docid uint64) float64, bestDistance *float64) {
	if node.isLeaf() {
		for _, id := range node.ids {
			*bestDistance = callback(id)
			if *bestDistance <= 0 {
				break
			}
		}
		return
	}

	side := distanceToHyperplane(vector, node.normal, node.b)
	distance := math.Abs(side)
	if tree.c.DistanceMethod == Cosine {
		distance = cosineDistance(vector, node.normal)
	}

	if side < 0 {
		tree.searchNode(node.left, vector, callback, bestDistance)
		if distance < *bestDistance {
			tree.searchNode(node.right, vector, callback, bestDistance)
		}
	} else {
		tree.searchNode(node.right, vector, callback, bestDistance)
		if distance < *bestDistance {
			tree.searchNode(node.left, vector, callback, bestDistance)
		}
	}
}
