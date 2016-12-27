package gtreap

type Treap struct {
	compare Compare
	root    *node
}

// Compare returns an integer comparing the two items
// lexicographically. The result will be 0 if a==b, -1 if a < b, and
// +1 if a > b.
type Compare func(a, b interface{}) int

// Item can be anything.
type Item interface{}

type node struct {
	item     Item
	priority int
	left     *node
	right    *node
}

func NewTreap(c Compare) *Treap {
	return &Treap{compare: c, root: nil}
}

func (t *Treap) Min() Item {
	n := t.root
	if n == nil {
		return nil
	}
	for n.left != nil {
		n = n.left
	}
	return n.item
}

func (t *Treap) Max() Item {
	n := t.root
	if n == nil {
		return nil
	}
	for n.right != nil {
		n = n.right
	}
	return n.item
}

func (t *Treap) Get(target Item) Item {
	n := t.root
	for n != nil {
		c := t.compare(target, n.item)
		if c < 0 {
			n = n.left
		} else if c > 0 {
			n = n.right
		} else {
			return n.item
		}
	}
	return nil
}

// Note: only the priority of the first insert of an item is used.
// Priorities from future updates on already existing items are
// ignored.  To change the priority for an item, you need to do a
// Delete then an Upsert.
func (t *Treap) Upsert(item Item, itemPriority int) *Treap {
	r := t.union(t.root, &node{item: item, priority: itemPriority})
	return &Treap{compare: t.compare, root: r}
}

func (t *Treap) union(this *node, that *node) *node {
	if this == nil {
		return that
	}
	if that == nil {
		return this
	}
	if this.priority > that.priority {
		left, middle, right := t.split(that, this.item)
		if middle == nil {
			return &node{
				item:     this.item,
				priority: this.priority,
				left:     t.union(this.left, left),
				right:    t.union(this.right, right),
			}
		}
		return &node{
			item:     middle.item,
			priority: this.priority,
			left:     t.union(this.left, left),
			right:    t.union(this.right, right),
		}
	}
	// We don't use middle because the "that" has precendence.
	left, _, right := t.split(this, that.item)
	return &node{
		item:     that.item,
		priority: that.priority,
		left:     t.union(left, that.left),
		right:    t.union(right, that.right),
	}
}

// Splits a treap into two treaps based on a split item "s".
// The result tuple-3 means (left, X, right), where X is either...
// nil - meaning the item s was not in the original treap.
// non-nil - returning the node that had item s.
// The tuple-3's left result treap has items < s,
// and the tuple-3's right result treap has items > s.
func (t *Treap) split(n *node, s Item) (*node, *node, *node) {
	if n == nil {
		return nil, nil, nil
	}
	c := t.compare(s, n.item)
	if c == 0 {
		return n.left, n, n.right
	}
	if c < 0 {
		left, middle, right := t.split(n.left, s)
		return left, middle, &node{
			item:     n.item,
			priority: n.priority,
			left:     right,
			right:    n.right,
		}
	}
	left, middle, right := t.split(n.right, s)
	return &node{
		item:     n.item,
		priority: n.priority,
		left:     n.left,
		right:    left,
	}, middle, right
}

func (t *Treap) Delete(target Item) *Treap {
	left, _, right := t.split(t.root, target)
	return &Treap{compare: t.compare, root: t.join(left, right)}
}

// All the items from this are < items from that.
func (t *Treap) join(this *node, that *node) *node {
	if this == nil {
		return that
	}
	if that == nil {
		return this
	}
	if this.priority > that.priority {
		return &node{
			item:     this.item,
			priority: this.priority,
			left:     this.left,
			right:    t.join(this.right, that),
		}
	}
	return &node{
		item:     that.item,
		priority: that.priority,
		left:     t.join(this, that.left),
		right:    that.right,
	}
}

type ItemVisitor func(i Item) bool

// Iterator returns an ascending Iterator instance that is bound to this Treap.
// The iterator begins at "pivot" and iterates through the end of the Treap.
func (t *Treap) Iterator(pivot Item) *Iterator {
	return newIterator(t, pivot)
}

// Visit items greater-than-or-equal to the pivot.
func (t *Treap) VisitAscend(pivot Item, visitor ItemVisitor) {
	t.visitAscend(t.root, pivot, visitor)
}

func (t *Treap) visitAscend(n *node, pivot Item, visitor ItemVisitor) bool {
	if n == nil {
		return true
	}
	if t.compare(pivot, n.item) <= 0 {
		if !t.visitAscend(n.left, pivot, visitor) {
			return false
		}
		if !visitor(n.item) {
			return false
		}
	}
	return t.visitAscend(n.right, pivot, visitor)
}

// Iterator supports iterative ascending traversal of the Treap. An Iterator is
// instantiated by calling a Treap's Iterator method.
type Iterator struct {
	t     *Treap
	pivot Item
	stack []stackNode
}

type stackNode struct {
	n       *node
	visited bool
}

func newIterator(t *Treap, pivot Item) *Iterator {
	it := Iterator{t: t, pivot: pivot}
	it.pushStack(t.root, false)
	return &it
}

// Next returns the next Item in the iteration sequence.
//
// If another item exists in the iteration sequence, true will be returned as
// the second return value; if not, false will be returned, indicating end of
// iteration. Additional calls to Next after end of iteration will continue
// to return false.
func (it *Iterator) Next() (Item, bool) {
	for {
		n, visited := it.popStack()
		if n == nil {
			return nil, false
		}

		if visited {
			// Only nodes that have already satisfied comparison will be placed on
			// the stack as "visited", so we can safely process them without
			// performing the comparison again.
			return n.item, true
		}

		if n.right != nil {
			it.pushStack(n.right, false)
		}
		if it.t.compare(it.pivot, n.item) <= 0 {
			// Visit our left node first. We will push "n" back onto the stack and
			// mark it visited so we don't revisit its children.
			if n.left == nil {
				// No left node, so we will skip the stack and visit "n" this round.
				return n.item, true
			}

			// Process "n" after its left child. Mark it "visited" so we don't re-push
			// entries onto the stack or re-compare.
			it.pushStack(n, true)
			it.pushStack(n.left, false)
		}
	}
}

func (it *Iterator) pushStack(n *node, visited bool) {
	it.stack = append(it.stack, stackNode{n, visited})
}

func (it *Iterator) popStack() (n *node, visited bool) {
	end := len(it.stack) - 1
	if end < 0 {
		return nil, false
	}

	sn := &it.stack[end]
	n, visited = sn.n, sn.visited
	sn.n = nil // Clear the node reference from our stack.
	it.stack = it.stack[:end]
	return
}
