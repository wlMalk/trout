package trout

import (
	"net/http"
	"sort"
	"strings"
)

type node struct {
	parent *node

	methods map[string]http.Handler
	allowed []string

	order int

	index       int
	hasWildcard bool

	nodes nodes

	text  string
	param string

	paramsCount int
}

func (n *node) String() string {
	str := ""
	for _, cn := range n.nodes {
		str += "\n" + cn.String()
	}
	str = strings.Replace(str, "\n", "\n\t", -1)
	return n.text + str
}

func createNodes(parts []string, names []string, indices []int, typs []int) *node {
	var pNode *node
	var node *node
	for i, part := range parts {
		node = newNode(part)
		node.setParam(names[i], indices[i], typs[i] == 1)
		if pNode != nil {
			pNode.addNode(node)
		}
		pNode = node
	}
	return node
}

func newRouteNode(path string) *node {
	slices := splitPath(path)
	parts, names, indices, typs := processPath(slices)
	node := createNodes(parts, names, indices, typs)
	return node
}

func newNode(text string) *node {
	return &node{
		text:  text,
		index: -1,
	}
}

type nodes []*node

func (n nodes) Len() int {
	return len(n)
}

func (n nodes) Swap(i, j int) {
	n[i], n[j] = n[j], n[i]
}

func (n nodes) Less(i, j int) bool {
	if n[i].index == -1 {
		return true
	} else if len(n[i].text) > 0 && len(n[j].text) > 0 && n[i].text[0] < n[j].text[0] {
		return true
	} else if !n[i].hasWildcard && n[j].hasWildcard {
		return true
	}
	return false
}

func (n *node) addMethod(method string, h http.Handler) {
	if n.methods == nil {
		n.methods = map[string]http.Handler{}
	}
	if _, ok := n.methods[strings.ToUpper(method)]; !ok {
		n.allowed = append(n.allowed, strings.ToUpper(method))
		sort.Strings(n.allowed)
	}
	n.methods[strings.ToUpper(method)] = h
}

func (n *node) match(path string, f func() *Params) (no *node, pars *Params, found bool) {
	pars = f()

	var part string

	i := 0

	ps := 0
	cs := 0

walk:
	for i = cs + 1; i < len(path)+1; i++ {
		if i == len(path) {
			ps = cs
			cs = i
			if i != ps+1 {
				if found && len(n.nodes) == 0 {
					if n.hasWildcard {
						found = true
						no = n
						return
					}
					found = false
					no = n
					return
				}
				part = path[ps+1 : cs]
			}
			break
		}
		if path[i] == '/' {
			ps = cs
			cs = i
			if i != ps+1 {
				if found && len(n.nodes) == 0 {
					if n.hasWildcard {
						found = true
						no = n
						return
					}
					found = false
					no = n
					return
				}
				part = path[ps+1 : cs]
				break
			}
		}
	}

	if !found {
		if n.parent == nil && n.text == "" {
			goto find
		}
		if ((len(n.text) < len(part) && n.index != -1) && strings.EqualFold(part[:len(n.text)], n.text)) ||
			strings.EqualFold(part, n.text) {
			found = true
			no = n

			if cs < len(path)-1 {
				goto walk
			}

			if no == nil {
				no = n
			}
			return
		}
		found = false

		return
	}

find:
	for _, cn := range n.nodes {
		if ((len(cn.text) < len(part) && cn.index != -1) && strings.EqualFold(part[:len(cn.text)], cn.text)) ||
			strings.EqualFold(part, cn.text) {
			if cn.index != -1 {
				if cn.hasWildcard {
					*pars = append(*pars, &Param{cn.param, path[ps+1+len(cn.text):]})
				} else {
					*pars = append(*pars, &Param{cn.param, part[len(cn.text):]})
				}
			}
			found = true
			no = cn

			if cs < len(path)-1 {
				n = cn
				goto walk
			}
			if no == nil {
				no = cn
			}
			return
		}
		found = false
	}

	return
}

func (n *node) getRoot() *node {
	for n.parent != nil {
		n = n.parent
	}
	return n
}

func (n *node) setParam(param string, index int, wildcard bool) {
	if index < 0 {
		return
	}
	if n.index == -1 {
		n.paramsCount = n.paramsCount + 1
	}
	n.index = index
	n.param = param
	n.hasWildcard = wildcard
}

func (n *node) setOrder() {
	n.order = -1
	node := n
	for node != nil {
		if (node.text != "" || node.index != -1) && !(node.text != "" && node.index != -1) {
			n.order = n.order + 1
		}
		node = node.parent
	}
}

func (n *node) finalize() {
	n.setOrder()
	sort.Sort(n.nodes)
	for _, cn := range n.nodes {
		cn.finalize()
	}
}

func (n *node) addNode(on *node) {
	if n.hasWildcard {
		panic("cannot add a node to a node that has wildcard")
	}

	found := false
	for _, cn := range n.nodes {
		if sameNodes(cn, on) {
			found = true
			recursiveCompare(cn, on)
			break
		}
	}
	if !found {
		n.nodes = append(n.nodes, on)
		on.parent = n
		if on.index != -1 {
			on.paramsCount = on.parent.paramsCount + 1
		} else {
			on.paramsCount = on.parent.paramsCount
		}
	}
}

func sameNodes(cn *node, on *node) bool {
	return cn.text == on.text && cn.index == on.index && cn.hasWildcard == on.hasWildcard
}

func recursiveCompare(cn *node, on *node) {
walk:
	for _, con := range on.nodes {
		for _, ccn := range cn.nodes {
			if sameNodes(ccn, con) {
				recursiveCompare(ccn, con)
				continue walk
			}
		}
		cn.nodes = append(cn.nodes, con)
		con.parent = cn
		if con.index != -1 {
			con.paramsCount = con.parent.paramsCount + 1
		} else {
			con.paramsCount = con.parent.paramsCount
		}
	}
}
