// Copyright 2017 Google LLC. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package maphasher provides hashing for maps.
package maphasher

import (
	"crypto"
	_ "crypto/sha256" // Default hasher uses SHA256.
	"fmt"

	"github.com/golang/glog"
	"github.com/google/trillian/storage/tree"
)

// Domain separation prefixes
const (
	leafHashPrefix = 0
	nodeHashPrefix = 1
)

// Default is a SHA256 based MapHasher for maps.
var Default = New(crypto.SHA256)

// MapHasher implements a sparse merkle tree hashing algorithm. For testing only.
// It matches the test vectors generated by other sparse map implementations,
// but it does not offer the full N bit security of the underlying hash function.
type MapHasher struct {
	crypto.Hash
	nullHashes [][]byte
}

// New creates a new MapHasher using the passed in hash function.
func New(h crypto.Hash) *MapHasher {
	m := &MapHasher{Hash: h}
	m.initNullHashes()
	return m
}

// String returns a string representation for debugging.
func (m *MapHasher) String() string {
	return fmt.Sprintf("MapHasher{%v}", m.Hash)
}

// HashEmpty returns the hash of an empty subtree with the given root. For this
// hasher, the result depends only on the height of the subtree.
func (m *MapHasher) HashEmpty(treeID int64, root tree.NodeID2) []byte {
	if depth := root.BitLen(); depth >= uint(len(m.nullHashes)*8) {
		panic(fmt.Sprintf("HashEmpty(%d) out of bounds", depth))
	}
	height := m.BitLen() - int(root.BitLen())
	if glog.V(5) {
		glog.Infof("HashEmpty(%v): %x", root, m.nullHashes[height])
	}
	return m.nullHashes[height]
}

// HashLeaf returns the Merkle tree leaf hash of the data passed in through
// leaf. The hashed structure is leafHashPrefix||leaf.
func (m *MapHasher) HashLeaf(treeID int64, id tree.NodeID2, leaf []byte) []byte {
	h := m.New()
	h.Write([]byte{leafHashPrefix})
	h.Write(leaf)
	r := h.Sum(nil)
	if glog.V(5) {
		glog.Infof("HashLeaf(%v): %x", id, r)
	}
	return r
}

// HashChildren returns the internal Merkle tree node hash of the the two child nodes l and r.
// The hashed structure is NodeHashPrefix||l||r.
func (m *MapHasher) HashChildren(l, r []byte) []byte {
	h := m.New()
	h.Write([]byte{nodeHashPrefix})
	h.Write(l)
	h.Write(r)
	p := h.Sum(nil)
	if glog.V(5) {
		glog.Infof("HashChildren(%x, %x): %x", l, r, p)
	}
	return p
}

// BitLen returns the number of bits in the hash function.
func (m *MapHasher) BitLen() int {
	return m.Size() * 8
}

// initNullHashes sets the cache of empty hashes, one for each level in the sparse tree,
// starting with the hash of an empty leaf, all the way up to the root hash of an empty tree.
// These empty branches are not stored on disk in a sparse tree. They are computed since their
// values are well-known.
func (m *MapHasher) initNullHashes() {
	// Leaves are stored at depth 0. Root is at Size()*8.
	// There are Size()*8 edges, and Size()*8 + 1 nodes in the tree.
	nodes := m.Size()*8 + 1
	r := make([][]byte, nodes)
	r[0] = m.HashLeaf(0, tree.NodeID2{}, nil)
	for i := 1; i < nodes; i++ {
		r[i] = m.HashChildren(r[i-1], r[i-1])
	}
	m.nullHashes = r
}
