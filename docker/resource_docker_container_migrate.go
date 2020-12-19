package docker

import (
	"fmt"
	"log"
	"sort"

	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func resourceDockerContainerMigrateState(
	v int, is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	switch v {
	case 0:
		log.Println("[INFO] Found Docker Container State v0; migrating to v1")
		return migrateDockerContainerMigrateStateV0toV1(is, meta)
	default:
		return is, fmt.Errorf("Unexpected schema version: %d", v)
	}
}

func migrateDockerContainerMigrateStateV0toV1(is *terraform.InstanceState, meta interface{}) (*terraform.InstanceState, error) {
	if is.Empty() {
		log.Println("[DEBUG] Empty InstanceState; nothing to migrate.")
		return is, nil
	}

	log.Printf("[DEBUG] Docker Container Attributes before Migration: %#v", is.Attributes)

	err := updateV0ToV1PortsOrder(is, meta)

	log.Printf("[DEBUG] Docker Container Attributes after State Migration: %#v", is.Attributes)

	return is, err
}

type mappedPort struct {
	internal int
	external int
	ip       string
	protocol string
}

type byPort []mappedPort

func (s byPort) Len() int {
	return len(s)
}

func (s byPort) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byPort) Less(i, j int) bool {
	return s[i].internal < s[j].internal
}

func updateV0ToV1PortsOrder(is *terraform.InstanceState, meta interface{}) error {
	reader := &schema.MapFieldReader{
		Schema: resourceDockerContainer().Schema,
		Map:    schema.BasicMapReader(is.Attributes),
	}

	writer := &schema.MapFieldWriter{
		Schema: resourceDockerContainer().Schema,
	}

	result, err := reader.ReadField([]string{"ports"})
	if err != nil {
		return err
	}

	if result.Value == nil {
		return nil
	}

	// map the ports into a struct, so they can be sorted easily
	portsMapped := make([]mappedPort, 0)
	portsRaw := result.Value.([]interface{})
	for _, portRaw := range portsRaw {
		if portRaw == nil {
			continue
		}
		portTyped := portRaw.(map[string]interface{})
		portMapped := mappedPort{
			internal: portTyped["internal"].(int),
			external: portTyped["external"].(int),
			ip:       portTyped["ip"].(string),
			protocol: portTyped["protocol"].(string),
		}

		portsMapped = append(portsMapped, portMapped)
	}
	sort.Sort(byPort(portsMapped))

	// map the sorted ports to an output structure tf can write
	outputPorts := make([]interface{}, 0)
	for _, mappedPort := range portsMapped {
		outputPort := make(map[string]interface{})
		outputPort["internal"] = mappedPort.internal
		outputPort["external"] = mappedPort.external
		outputPort["ip"] = mappedPort.ip
		outputPort["protocol"] = mappedPort.protocol
		outputPorts = append(outputPorts, outputPort)
	}

	// store them back to state
	if err := writer.WriteField([]string{"ports"}, outputPorts); err != nil {
		return err
	}
	for k, v := range writer.Map() {
		is.Attributes[k] = v
	}

	return nil
}
