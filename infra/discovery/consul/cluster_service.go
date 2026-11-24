package consul

import (
	"context"
	"net"
	"strconv"

	"github.com/hashicorp/consul/api"
)

// ServiceNode is one instance of a service with the node it runs on.
type ServiceNode struct {
	ID      string
	Service string
	// Address is the address the instance registered; empty means the node's.
	Address string
	Port    int
	Tags    []string
	Meta    map[string]string

	Node        string
	NodeAddress string
	NodeMeta    map[string]string

	// Status is the aggregated status of the checks of the instance and its node:
	// passing, warning or critical.
	Status string
}

// HostPort returns host:port of the instance, falling back to the node's address.
func (n ServiceNode) HostPort() string {
	host := n.Address
	if host == "" {
		host = n.NodeAddress
	}

	return net.JoinHostPort(host, strconv.Itoa(n.Port))
}

// ServiceQuery narrows [Cluster.ServiceNodes].
type ServiceQuery struct {
	// PassingOnly leaves out instances whose checks are not all passing.
	PassingOnly bool
	// Tag leaves only instances with this tag; empty means any.
	Tag string
	// NodeMeta leaves only instances on nodes with all of these node_meta pairs.
	NodeMeta map[string]string
}

// ServiceNodes returns the instances of service.
func (c *Cluster) ServiceNodes(ctx context.Context, service string, q ServiceQuery) ([]ServiceNode, error) {
	var out []ServiceNode

	err := c.do(ctx, true, func(ctx context.Context, client *api.Client) error {
		opts := new(api.QueryOptions).WithContext(ctx)
		opts.NodeMeta = q.NodeMeta

		entries, _, err := client.Health().Service(service, q.Tag, q.PassingOnly, opts)
		if err != nil {
			return err
		}

		out = serviceNodes(entries)

		return nil
	})

	return out, err
}

// ExecuteQuery executes the prepared query given by name or ID. Each node carries
// its node_meta.
func (c *Cluster) ExecuteQuery(ctx context.Context, nameOrID string) ([]ServiceNode, error) {
	var out []ServiceNode

	err := c.do(ctx, true, func(ctx context.Context, client *api.Client) error {
		resp, _, err := client.PreparedQuery().Execute(nameOrID, new(api.QueryOptions).WithContext(ctx))
		if err != nil {
			return err
		}

		entries := make([]*api.ServiceEntry, len(resp.Nodes))
		for i := range resp.Nodes {
			entries[i] = &resp.Nodes[i]
		}

		out = serviceNodes(entries)

		return nil
	})

	return out, err
}

func serviceNodes(entries []*api.ServiceEntry) []ServiceNode {
	out := make([]ServiceNode, 0, len(entries))

	for _, e := range entries {
		n := ServiceNode{Status: e.Checks.AggregatedStatus()}

		if e.Service != nil {
			n.ID, n.Service, n.Address, n.Port = e.Service.ID, e.Service.Service, e.Service.Address, e.Service.Port
			n.Tags, n.Meta = e.Service.Tags, e.Service.Meta
		}

		if e.Node != nil {
			n.Node, n.NodeAddress, n.NodeMeta = e.Node.Node, e.Node.Address, e.Node.Meta
		}

		out = append(out, n)
	}

	return out
}
