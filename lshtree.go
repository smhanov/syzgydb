package syzgydb

import (
	"container/heap"
	"log"
	"math"
	"sync"
)

func normalizeVector(vector []float64) []float64 {
	norm := 0.0

	for _, v := range vector {
		norm += v * v
	}

	if norm == 0 {
		return vector
	}

	norm = math.Sqrt(norm)

	for i, v := range vector {
		vector[i] = v / norm
	}

	return vector
}

func vectorLength(vector []float64) float64 {
	sum := 0.0
	for _, v := range vector {
		sum += v * v
	}
	return math.Sqrt(sum)
}

func randomNormalizedVector(rand *myRandomType, dim int) []float64 {
	vector := make([]float64, dim)
	for i := range vector {
		vector[i] = rand.NormFloat64()
	}
	return normalizeVector(vector)
}

type lshNode struct {
	normal []float64
	b      float64
	// lmax is the maximum distance of any point in the left side of the tree to the hyperplane.
	lmin, lmax  float64
	rmin, rmax  float64
	left, right *lshNode
	ids         []uint64
}

func (n *lshNode) isLeaf() bool {
	return n.left == nil
}

func distanceToHyperplane(method int, vector []float64, length float64, normal []float64, b float64) (dist float64, right bool) {
	dist = angularDistance(vector, normal)
	if dist < 0.5 {
		right = true
	} else {
		dist = 1 - dist
	}
	return

	dist = dotProduct(vector, normal) - b
	if method == Euclidean {
		if dist > 0 {
			right = true
		} else {
			dist = -dist
		}
		return
	}

	// angular distance of two already-normalized vectors
	dist = math.Acos(dist/length) / math.Pi
	if dist < 0.5 {
		right = true
	} else {
		dist = 1 - dist
	}
	return
}

type lshTree struct {
	roots     []*lshNode // Change from a single root to a slice of roots
	threshold int
	c         *Collection

	// random source
	rand *myRandomType
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
		rand:      myRandom.ThreadsafeNew(),
	}
}

func (tree *lshTree) addPoint(docid uint64, vector []float64) {
	length := vectorLength(vector)
	var wg sync.WaitGroup
	wg.Add(len(tree.roots))

	for i, root := range tree.roots {
		go func(i int, root *lshNode) {
			tree.roots[i] = tree.insert(root, docid, vector, length)
			wg.Done()
		}(i, root)
	}

	wg.Wait()
}

func (tree *lshTree) insert(node *lshNode, docid uint64, vector []float64, length float64) *lshNode {
	if node.isLeaf() {
		node.ids = append(node.ids, docid)
		if len(node.ids) > tree.threshold {
			node = tree.split(node)
		}
		return node
	}

	distance, right := distanceToHyperplane(tree.c.DistanceMethod, vector, length, node.normal, node.b)
	if right {
		node.rmax = math.Max(node.rmax, distance)
		node.rmin = math.Min(node.rmin, distance)
		node.right = tree.insert(node.right, docid, vector, length)
	} else {
		node.lmax = math.Max(node.lmax, distance)
		node.lmin = math.Min(node.lmin, distance)
		node.left = tree.insert(node.left, docid, vector, length)
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
	randomIndex1 := tree.rand.Intn(len(node.ids))
	var randomIndex2 int
	for {
		randomIndex2 = tree.rand.Intn(len(node.ids))
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
	var normal []float64
	var b float64

	if tree.c.DistanceMethod == Euclidean {
		normal = randomNormalizedVector(tree.rand, len(pointChosen))
		b = math.Sqrt(dotProduct(pointChosen, pointChosen))
	} else {
		//normal = normalizeVector(pointChosen)
		normal = randomNormalizedVector(tree.rand, len(pointChosen))
	}

	leftIDs := []uint64{}
	rightIDs := []uint64{}
	var lmax float64
	var rmax float64
	lmin := math.MaxFloat64
	rmin := math.MaxFloat64

	for _, id := range node.ids {
		doc, err := tree.c.getDocument(id)
		if err != nil {
			log.Panicf("error getting document: %v", err)
		}

		v := doc.Vector
		length := vectorLength(v)

		distance, right := distanceToHyperplane(tree.c.DistanceMethod, v, length, normal, b)
		if right {
			rmax = math.Max(lmax, distance)
			rmin = math.Min(lmin, distance)
			rightIDs = append(rightIDs, id)
		} else {
			lmax = math.Max(lmax, distance)
			lmin = math.Min(lmin, distance)
			leftIDs = append(leftIDs, id)
		}
	}

	//log.Printf("    leftCount: %v, rightCount: %v", len(leftIDs), len(rightIDs))

	if len(leftIDs) == 0 || len(rightIDs) == 0 {
		return node // Avoid splitting if all vectors are on one side
	}

	return &lshNode{
		normal: normal,
		b:      b,
		lmax:   lmax,
		lmin:   lmin,
		rmax:   rmax,
		rmin:   rmin,
		left:   &lshNode{ids: leftIDs},
		right:  &lshNode{ids: rightIDs},
	}
}

func (tree *lshTree) removePoint(docid uint64, vector []float64) {
	length := vectorLength(vector)
	for i, root := range tree.roots {
		tree.roots[i] = tree.remove(root, docid, vector, length)
	}
}

func (tree *lshTree) remove(node *lshNode, docid uint64, vector []float64, length float64) *lshNode {
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
	_, right := distanceToHyperplane(tree.c.DistanceMethod, vector, length, node.normal, node.b)
	if right {
		node.right = tree.remove(node.right, docid, vector, length)
	} else {
		node.left = tree.remove(node.left, docid, vector, length)
	}
	return node
}

func (tree *lshTree) search(vector []float64, radius float64, callback searchCallback) {
	length := vectorLength(vector)
	visited := make(map[uint64]bool)
	const search_k = 200 // how many points do we search beyond what is required in hopes of finding a better result.
	k_counter := 0       // number of times we visited a point and it didn't yield any better results.
	pointAccepted := false

	// Initialize the priority queue
	pq := &nodePriorityQueue{}
	heap.Init(pq)

	// Add all roots to the priority queue
	for _, node := range tree.roots {
		heap.Push(pq, &nodePriorityItem{node: node})
	}

	for pq.Len() > 0 {
		item := heap.Pop(pq).(*nodePriorityItem)
		node := item.node
		log.Printf("Node with priority %v (tau=%v, k=%v) leaf=%v", item.distance, radius, k_counter, node.isLeaf())

		if k_counter >= search_k {
			break
		}

		if node.isLeaf() {
			for _, id := range node.ids {
				if visited[id] {
					continue
				}
				visited[id] = true
				var signal int
				signal, radius = callback(id, radius)
				switch signal {
				case StopSearch:
					return
				case PointAccepted:
					k_counter = 0
					pointAccepted = true
				case PointChecked:
					if pointAccepted {
						k_counter++
					}
				case PointIgnored:
				}
			}
			//log.Printf("Search radius now %v k=%v", radius, k_counter)
		} else {
			// Calculate the distance to the hyperplane
			dist, right := distanceToHyperplane(tree.c.DistanceMethod, vector, length, node.normal, node.b)
			log.Printf("Distance to hyperplane: right=%v %v (%v-%v, %v-%v)", right, dist, node.rmax, node.rmin, node.lmax, node.lmin)

			// Add child nodes to the priority queue
			if right {
				heap.Push(pq, &nodePriorityItem{node: node.right, distance: dist - node.rmax})
				heap.Push(pq, &nodePriorityItem{node: node.left, distance: dist})
			} else {
				heap.Push(pq, &nodePriorityItem{node: node.left, distance: dist - node.lmax})
				heap.Push(pq, &nodePriorityItem{node: node.right, distance: dist})
			}
		}
	}
}

type nodePriorityItem struct {
	node     *lshNode
	distance float64
}

type nodePriorityQueue []*nodePriorityItem

func (pq nodePriorityQueue) Len() int { return len(pq) }

func (pq nodePriorityQueue) Less(i, j int) bool {
	return pq[i].distance < pq[j].distance
	//return pq[i].priority > pq[j].priority // Max-heap based on priority
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
