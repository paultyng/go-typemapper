// +build typemapper

package consul

import (
	"github.com/hashicorp/consul/agent/structs"

	"github.com/paultyng/go-typemapper"
)

// https://github.com/hashicorp/consul/blob/5457bca10c1f8a2ac0e338fdc06c95fd5cff49c3/agent/structs/structs.go#L642-L660
func ServiceNodeToNodeService(src *structs.ServiceNode, dst *structs.NodeService) {
	typemapper.CreateMap(src, dst)
	typemapper.RecognizePrefixes("Service")
	typemapper.MapField(src.ServiceName, dst.Service)
	typemapper.IgnoreFields(
		dst.LocallyRegisteredAsSidecar,
	)
	return
}

// https://github.com/hashicorp/consul/blob/5457bca10c1f8a2ac0e338fdc06c95fd5cff49c3/agent/structs/structs.go#L893-L930