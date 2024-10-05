package syzgydb

import (
	"container/heap"
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

func distanceToHyperplane(method int, vector, normal []float64, b float64) (dist float64, right bool) {
	if method == Euclidean {
		dist = dotProduct(vector, normal) - b
		if dist > 0 {
			right = true
		} else {
			dist = -dist
		}
		return
	}
	dist = angularDistance(vector, normal)
	right = dist > 0.5
	return
}

type lshTree struct {
	roots     []*lshNode // Change from a single root to a slice of roots
	threshold int
	c         *Collection
}

func newLSHTree(c *Collection, threshold int, numTrees int) *lshTree {
	roots := make([]*lshNode, numTrees)
	for i := 0; i < numTrees; i++ {
		roots[i] = &lshNode{ids: []uint64{}}
	}
	return &lshTree{
		roots:     roots,
		threshold: threshold,
		c:         c,
	}
}

func (tree *lshTree) addPoint(docid uint64, vector []float64) {
	for i, root := range tree.roots {
		tree.roots[i] = tree.insert(root, docid, vector)
	}
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

	_, right := distanceToHyperplane(tree.c.DistanceMethod, vector, node.normal, node.b)
	if !right {
		node.left = tree.insert(node.left, docid, vector)
	} else {
		node.right = tree.insert(node.right, docid, vector)
	}
	// If both children are nil, this node becomes a leaf
	if node.left == nil && node.right == nil {
		return nil
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

		_, right := distanceToHyperplane(tree.c.DistanceMethod, v, normal, b)
		if !right {
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
	normalizedVector := normalizeVector(vector)
	for i, root := range tree.roots {
		tree.roots[i] = tree.remove(root, docid, normalizedVector)
	}
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
		// If the node is empty, return nil to remove it
		if len(node.ids) == 0 {
			return nil
		}
		return node
	}

	// Traverse the tree based on the vector's position relative to the hyperplane
	_, right := distanceToHyperplane(tree.c.DistanceMethod, vector, node.normal, node.b)
	if !right {
		node.left = tree.remove(node.left, docid, vector)
	} else {
		node.right = tree.remove(node.right, docid, vector)
	}
	return node
}

func (tree *lshTree) search(vector []float64, callback func(docid uint64) float64) {
	tau := math.MaxFloat64
	normalizedVector := normalizeVector(vector)

	// Initialize the priority queue
	pq := &nodePriorityQueue{}
	heap.Init(pq)

	// Add all roots to the priority queue
	for _, root := range tree.roots {
		heap.Push(pq, &nodePriorityItem{node: root, priority: 0})
	}

	for pq.Len() > 0 {
		item := heap.Pop(pq).(*nodePriorityItem)
		node := item.node

		if item.priority > tau {
			break
		}

		if node.isLeaf() {
			for _, id := range node.ids {
				distance := callback(id)
				if distance < 0 {
					return
				}
				if distance < tau {
					tau = distance
				}
			}
		} else {
			// Calculate the distance to the hyperplane
			dist, right := distanceToHyperplane(Cosine, normalizedVector, node.normal, node.b)

			// Add child nodes to the priority queue
			if right {
				heap.Push(pq, &nodePriorityItem{node: node.right, priority: dist})
				heap.Push(pq, &nodePriorityItem{node: node.left, priority: tau - dist})
			} else {
				heap.Push(pq, &nodePriorityItem{node: node.left, priority: dist})
				heap.Push(pq, &nodePriorityItem{node: node.right, priority: tau - dist})
			}
		}
	}
}

type nodePriorityItem struct {
	node     *lshNode
	priority float64
}

type nodePriorityQueue []*nodePriorityItem

func (pq nodePriorityQueue) Len() int { return len(pq) }

func (pq nodePriorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority // Min-heap based on priority
}

func (pq nodePriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
}

func (pq *nodePriorityQueue) Push(x interface{}) {
	item := x.(*nodePriorityItem)
	*pq = append(*pq, item)
}

func (pq *nodePriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	*pq = old[0 : n-1]
	return item
}
