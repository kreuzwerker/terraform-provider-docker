package provider

import (
	"log"

	"github.com/docker/docker/api/types/network"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// TODO 2: seems like we can replace the set hash generation with plain lists -> #74 (import resources)
func flattenIpamConfigSpec(in []network.IPAMConfig) *schema.Set { // []interface{} {
	out := make([]interface{}, len(in))
	for i, v := range in {
		log.Printf("[DEBUG] flatten ipam %d: %#v", i, v)
		m := make(map[string]interface{})
		if len(v.Subnet) > 0 {
			m["subnet"] = v.Subnet
		}
		if len(v.IPRange) > 0 {
			m["ip_range"] = v.IPRange
		}
		if len(v.Gateway) > 0 {
			m["gateway"] = v.Gateway
		}
		if len(v.AuxAddress) > 0 {
			aux := make(map[string]interface{}, len(v.AuxAddress))
			for ka, va := range v.AuxAddress {
				aux[ka] = va
			}
			m["aux_address"] = aux
		}
		out[i] = m
	}
	// log.Printf("[INFO] flatten ipam out: %#v", out)
	imapConfigsResource := resourceDockerNetwork().Schema["ipam_config"].Elem.(*schema.Resource)
	f := schema.HashResource(imapConfigsResource)
	return schema.NewSet(f, out)
	// return out
}
