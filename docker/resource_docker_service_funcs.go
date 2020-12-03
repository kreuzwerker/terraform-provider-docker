package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"encoding/base64"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/helper/schema"
)

type convergeConfig struct {
	timeout    time.Duration
	timeoutRaw string
	delay      time.Duration
}

/////////////////
// TF CRUD funcs
/////////////////
func resourceDockerServiceExists(d *schema.ResourceData, meta interface{}) (bool, error) {
	client := meta.(*ProviderConfig).DockerClient
	if client == nil {
		return false, nil
	}

	apiService, err := fetchDockerService(d.Id(), d.Get("name").(string), client)
	if err != nil {
		return false, err
	}
	if apiService == nil {
		return false, nil
	}

	return true, nil
}

func resourceDockerServiceCreate(d *schema.ResourceData, meta interface{}) error {
	var err error
	client := meta.(*ProviderConfig).DockerClient

	serviceSpec, err := createServiceSpec(d)
	if err != nil {
		return err
	}

	serviceOptions := types.ServiceCreateOptions{}
	marshalledAuth := retrieveAndMarshalAuth(d, meta, "create")
	serviceOptions.EncodedRegistryAuth = base64.URLEncoding.EncodeToString(marshalledAuth)
	serviceOptions.QueryRegistry = true
	log.Printf("[DEBUG] Dummy log\n")
	log.Printf("[DEBUG] Passing registry auth '%s'", serviceOptions.EncodedRegistryAuth)

	service, err := client.ServiceCreate(context.Background(), serviceSpec, serviceOptions)
	if err != nil {
		return err
	}
	if v, ok := d.GetOk("converge_config"); ok {
		convergeConfig := createConvergeConfig(v.([]interface{}))
		log.Printf("[INFO] Waiting for Service '%s' to be created with timeout: %v", service.ID, convergeConfig.timeoutRaw)
		timeout, _ := time.ParseDuration(convergeConfig.timeoutRaw)
		stateConf := &resource.StateChangeConf{
			Pending:    serviceCreatePendingStates,
			Target:     []string{"running", "complete"},
			Refresh:    resourceDockerServiceCreateRefreshFunc(service.ID, meta),
			Timeout:    timeout,
			MinTimeout: 5 * time.Second,
			Delay:      convergeConfig.delay,
		}

		// Wait, catching any errors
		_, err := stateConf.WaitForState()
		if err != nil {
			// the service will be deleted in case it cannot be converged
			if deleteErr := deleteService(service.ID, d, client); deleteErr != nil {
				return deleteErr
			}
			if strings.Contains(err.Error(), "timeout while waiting for state") {
				return &DidNotConvergeError{ServiceID: service.ID, Timeout: convergeConfig.timeout}
			}
			return err
		}
	}

	d.SetId(service.ID)
	return resourceDockerServiceRead(d, meta)
}

func resourceDockerServiceRead(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Waiting for service: '%s' to expose all fields: max '%v seconds'", d.Id(), 30)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     []string{"all_fields", "removed"},
		Refresh:    resourceDockerServiceReadRefreshFunc(d, meta),
		Timeout:    30 * time.Second,
		MinTimeout: 5 * time.Second,
		Delay:      2 * time.Second,
	}

	// Wait, catching any errors
	_, err := stateConf.WaitForState()
	if err != nil {
		return err
	}

	return nil
}

func resourceDockerServiceReadRefreshFunc(
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		client := meta.(*ProviderConfig).DockerClient
		serviceID := d.Id()

		apiService, err := fetchDockerService(serviceID, d.Get("name").(string), client)
		if err != nil {
			return nil, "", err
		}
		if apiService == nil {
			log.Printf("[WARN] Service (%s) not found, removing from state", serviceID)
			d.SetId("")
			return serviceID, "removed", nil
		}
		service, _, err := client.ServiceInspectWithRaw(context.Background(), apiService.ID, types.ServiceInspectOptions{})
		if err != nil {
			return serviceID, "", fmt.Errorf("Error inspecting service %s: %s", apiService.ID, err)
		}

		jsonObj, _ := json.MarshalIndent(service, "", "\t")
		log.Printf("[DEBUG] Docker service inspect: %s", jsonObj)

		if string(service.Endpoint.Spec.Mode) == "" && string(service.Spec.EndpointSpec.Mode) == "" {
			log.Printf("[DEBUG] Service %s does not expose endpoint spec yet", apiService.ID)
			return serviceID, "pending", nil
		}

		d.SetId(service.ID)
		d.Set("name", service.Spec.Name)
		d.Set("labels", mapToLabelSet(service.Spec.Labels))

		if err = d.Set("task_spec", flattenTaskSpec(service.Spec.TaskTemplate)); err != nil {
			log.Printf("[WARN] failed to set task spec from API: %s", err)
		}
		if err = d.Set("mode", flattenServiceMode(service.Spec.Mode)); err != nil {
			log.Printf("[WARN] failed to set mode from API: %s", err)
		}
		if err := d.Set("update_config", flattenServiceUpdateOrRollbackConfig(service.Spec.UpdateConfig)); err != nil {
			log.Printf("[WARN] failed to set update_config from API: %s", err)
		}
		if err = d.Set("rollback_config", flattenServiceUpdateOrRollbackConfig(service.Spec.RollbackConfig)); err != nil {
			log.Printf("[WARN] failed to set rollback_config from API: %s", err)
		}

		if service.Endpoint.Spec.Mode != "" {
			if err = d.Set("endpoint_spec", flattenServiceEndpoint(service.Endpoint)); err != nil {
				log.Printf("[WARN] failed to set endpoint spec from API: %s", err)
			}
		} else if service.Spec.EndpointSpec.Mode != "" {
			if err = d.Set("endpoint_spec", flattenServiceEndpointSpec(service.Spec.EndpointSpec)); err != nil {
				log.Printf("[WARN] failed to set endpoint spec from API: %s", err)
			}
		} else {
			return serviceID, "", fmt.Errorf("Error no endpoint spec for service %s", apiService.ID)
		}

		return serviceID, "all_fields", nil
	}
}

func resourceDockerServiceUpdate(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient

	service, _, err := client.ServiceInspectWithRaw(context.Background(), d.Id(), types.ServiceInspectOptions{})
	if err != nil {
		return err
	}

	serviceSpec, err := createServiceSpec(d)
	if err != nil {
		return err
	}

	updateOptions := types.ServiceUpdateOptions{}
	marshalledAuth := retrieveAndMarshalAuth(d, meta, "update")
	if err != nil {
		return fmt.Errorf("error creating auth config: %s", err)
	}
	updateOptions.EncodedRegistryAuth = base64.URLEncoding.EncodeToString(marshalledAuth)

	updateResponse, err := client.ServiceUpdate(context.Background(), d.Id(), service.Version, serviceSpec, updateOptions)
	if err != nil {
		return err
	}
	if len(updateResponse.Warnings) > 0 {
		log.Printf("[INFO] Warninig while updating Service '%s': %v", service.ID, updateResponse.Warnings)
	}

	if v, ok := d.GetOk("converge_config"); ok {
		convergeConfig := createConvergeConfig(v.([]interface{}))
		log.Printf("[INFO] Waiting for Service '%s' to be updated with timeout: %v", service.ID, convergeConfig.timeoutRaw)
		timeout, _ := time.ParseDuration(convergeConfig.timeoutRaw)
		stateConf := &resource.StateChangeConf{
			Pending:    serviceUpdatePendingStates,
			Target:     []string{"completed"},
			Refresh:    resourceDockerServiceUpdateRefreshFunc(service.ID, meta),
			Timeout:    timeout,
			MinTimeout: 5 * time.Second,
			Delay:      7 * time.Second,
		}

		// Wait, catching any errors
		state, err := stateConf.WaitForState()
		log.Printf("[INFO] State awaited: %v with error: %v", state, err)
		if err != nil {
			if strings.Contains(err.Error(), "timeout while waiting for state") {
				return &DidNotConvergeError{ServiceID: service.ID, Timeout: convergeConfig.timeout}
			}
			return err
		}
	}

	return resourceDockerServiceRead(d, meta)
}

func resourceDockerServiceDelete(d *schema.ResourceData, meta interface{}) error {
	client := meta.(*ProviderConfig).DockerClient

	if err := deleteService(d.Id(), d, client); err != nil {
		return err
	}

	d.SetId("")
	return nil
}

/////////////////
// Helpers
/////////////////
// fetchDockerService fetches a service by its name or id
func fetchDockerService(ID string, name string, client *client.Client) (*swarm.Service, error) {
	apiServices, err := client.ServiceList(context.Background(), types.ServiceListOptions{})

	if err != nil {
		return nil, fmt.Errorf("Error fetching service information from Docker: %s", err)
	}

	for _, apiService := range apiServices {
		if apiService.ID == ID || apiService.Spec.Name == name {
			return &apiService, nil
		}
	}

	return nil, nil
}

// deleteService deletes the service with the given id
func deleteService(serviceID string, d *schema.ResourceData, client *client.Client) error {
	// get containerIDs of the running service because they do not exist after the service is deleted
	serviceContainerIds := make([]string, 0)
	if _, ok := d.GetOk("task_spec.0.container_spec.0.stop_grace_period"); ok {
		filters := filters.NewArgs()
		filters.Add("service", d.Get("name").(string))
		tasks, err := client.TaskList(context.Background(), types.TaskListOptions{
			Filters: filters,
		})
		if err != nil {
			return err
		}
		for _, t := range tasks {
			task, _, _ := client.TaskInspectWithRaw(context.Background(), t.ID)
			containerID := ""
			if task.Status.ContainerStatus != nil {
				containerID = task.Status.ContainerStatus.ContainerID
			}
			log.Printf("[INFO] Found container ['%s'] for destroying: '%s'", task.Status.State, containerID)
			if strings.TrimSpace(containerID) != "" && task.Status.State != swarm.TaskStateShutdown {
				serviceContainerIds = append(serviceContainerIds, containerID)
			}
		}
	}

	// delete the service
	log.Printf("[INFO] Deleting service: '%s'", serviceID)
	if err := client.ServiceRemove(context.Background(), serviceID); err != nil {
		return fmt.Errorf("Error deleting service %s: %s", serviceID, err)
	}

	// destroy each container after a grace period if specified
	if v, ok := d.GetOk("task_spec.0.container_spec.0.stop_grace_period"); ok {
		for _, containerID := range serviceContainerIds {
			destroyGraceSeconds, _ := time.ParseDuration(v.(string))
			log.Printf("[INFO] Waiting for container: '%s' to exit: max %v", containerID, destroyGraceSeconds)
			ctx, cancel := context.WithTimeout(context.Background(), destroyGraceSeconds)
			// TODO why defer? see container_resource with handling return channels! why not remove then wait?
			defer cancel()
			exitCode, _ := client.ContainerWait(ctx, containerID, container.WaitConditionRemoved)
			log.Printf("[INFO] Container exited with code [%v]: '%s'", exitCode, containerID)

			removeOpts := types.ContainerRemoveOptions{
				RemoveVolumes: true,
				Force:         true,
			}

			log.Printf("[INFO] Removing container: '%s'", containerID)
			if err := client.ContainerRemove(context.Background(), containerID, removeOpts); err != nil {
				if !(strings.Contains(err.Error(), "No such container") || strings.Contains(err.Error(), "is already in progress")) {
					return fmt.Errorf("Error deleting container %s: %s", containerID, err)
				}
			}
		}
	}

	return nil
}

//////// Convergers

// DidNotConvergeError is the error returned when a the service does not converge in
// the defined time
type DidNotConvergeError struct {
	ServiceID string
	Timeout   time.Duration
	Err       error
}

// Error the custom error if a service does not converge
func (err *DidNotConvergeError) Error() string {
	if err.Err != nil {
		return err.Err.Error()
	}
	return "Service with ID (" + err.ServiceID + ") did not converge after " + err.Timeout.String()
}

// resourceDockerServiceCreateRefreshFunc refreshes the state of a service when it is created and needs to converge
func resourceDockerServiceCreateRefreshFunc(
	serviceID string, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		client := meta.(*ProviderConfig).DockerClient
		ctx := context.Background()

		var updater progressUpdater

		if updater == nil {
			updater = &replicatedConsoleLogUpdater{}
		}

		filters := filters.NewArgs()
		filters.Add("service", serviceID)
		filters.Add("desired-state", "running")

		getUpToDateTasks := func() ([]swarm.Task, error) {
			return client.TaskList(ctx, types.TaskListOptions{
				Filters: filters,
			})
		}

		service, _, err := client.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
		if err != nil {
			return nil, "", err
		}

		tasks, err := getUpToDateTasks()
		if err != nil {
			return nil, "", err
		}

		activeNodes, err := getActiveNodes(ctx, client)
		if err != nil {
			return nil, "", err
		}

		serviceCreateStatus, err := updater.update(&service, tasks, activeNodes, false)
		if err != nil {
			return nil, "", err
		}

		if serviceCreateStatus {
			return service.ID, "running", nil
		}

		return service.ID, "creating", nil
	}
}

// resourceDockerServiceUpdateRefreshFunc refreshes the state of a service when it is updated and needs to converge
func resourceDockerServiceUpdateRefreshFunc(
	serviceID string, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		client := meta.(*ProviderConfig).DockerClient
		ctx := context.Background()

		var (
			updater  progressUpdater
			rollback bool
		)

		if updater == nil {
			updater = &replicatedConsoleLogUpdater{}
		}
		rollback = false

		filters := filters.NewArgs()
		filters.Add("service", serviceID)
		filters.Add("desired-state", "running")

		getUpToDateTasks := func() ([]swarm.Task, error) {
			return client.TaskList(ctx, types.TaskListOptions{
				Filters: filters,
			})
		}

		service, _, err := client.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
		if err != nil {
			return nil, "", err
		}

		if service.UpdateStatus != nil {
			log.Printf("[DEBUG] update status: %v", service.UpdateStatus.State)
			switch service.UpdateStatus.State {
			case swarm.UpdateStateUpdating:
				rollback = false
			case swarm.UpdateStateCompleted:
				return service.ID, "completed", nil
			case swarm.UpdateStateRollbackStarted:
				rollback = true
			case swarm.UpdateStateRollbackCompleted:
				return nil, "", fmt.Errorf("service rollback completed: %s", service.UpdateStatus.Message)
			case swarm.UpdateStatePaused:
				return nil, "", fmt.Errorf("service update paused: %s", service.UpdateStatus.Message)
			case swarm.UpdateStateRollbackPaused:
				return nil, "", fmt.Errorf("service rollback paused: %s", service.UpdateStatus.Message)
			}
		}

		tasks, err := getUpToDateTasks()
		if err != nil {
			return nil, "", err
		}

		activeNodes, err := getActiveNodes(ctx, client)
		if err != nil {
			return nil, "", err
		}

		isUpdateCompleted, err := updater.update(&service, tasks, activeNodes, rollback)
		if err != nil {
			return nil, "", err
		}

		if isUpdateCompleted {
			if rollback {
				return nil, "", fmt.Errorf("service rollback completed: %s", service.UpdateStatus.Message)
			}
			return service.ID, "completed", nil
		}

		return service.ID, "updating", nil
	}
}

// getActiveNodes gets the actives nodes withon a swarm
func getActiveNodes(ctx context.Context, client *client.Client) (map[string]struct{}, error) {
	nodes, err := client.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return nil, err
	}

	activeNodes := make(map[string]struct{})
	for _, n := range nodes {
		if n.Status.State != swarm.NodeStateDown {
			activeNodes[n.ID] = struct{}{}
		}
	}
	return activeNodes, nil
}

// progressUpdater interface for progressive task updates
type progressUpdater interface {
	update(service *swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, rollback bool) (bool, error)
}

// replicatedConsoleLogUpdater console log updater for replicated services
type replicatedConsoleLogUpdater struct {
	// used for mapping slots to a contiguous space
	// this also causes progress bars to appear in order
	slotMap map[int]int

	initialized bool
	done        bool
}

// update is the concrete implementation of updating replicated services
func (u *replicatedConsoleLogUpdater) update(service *swarm.Service, tasks []swarm.Task, activeNodes map[string]struct{}, rollback bool) (bool, error) {
	if service.Spec.Mode.Replicated == nil || service.Spec.Mode.Replicated.Replicas == nil {
		return false, fmt.Errorf("no replica count")
	}
	replicas := *service.Spec.Mode.Replicated.Replicas

	if !u.initialized {
		u.slotMap = make(map[int]int)
		u.initialized = true
	}

	// get the task for each slot. there can be multiple slots on one node
	tasksBySlot := u.tasksBySlot(tasks, activeNodes)

	// if a converged state is reached, check if is still converged.
	if u.done {
		for _, task := range tasksBySlot {
			if task.Status.State != swarm.TaskStateRunning {
				u.done = false
				break
			}
		}
	}

	running := uint64(0)

	// map the slots to keep track of their state individually
	for _, task := range tasksBySlot {
		mappedSlot := u.slotMap[task.Slot]
		if mappedSlot == 0 {
			mappedSlot = len(u.slotMap) + 1
			u.slotMap[task.Slot] = mappedSlot
		}

		// if a task is in the desired state count it as running
		if !terminalState(task.DesiredState) && task.Status.State == swarm.TaskStateRunning {
			running++
		}
	}

	// check if all tasks the same amount of tasks is running than replicas defined
	if !u.done {
		log.Printf("[INFO] ... progress: [%v/%v] - rollback: %v", running, replicas, rollback)
		if running == replicas {
			log.Printf("[INFO] DONE: all %v replicas running", running)
			u.done = true
		}
	}

	return running == replicas, nil
}

// tasksBySlot maps the tasks to slots on active nodes. There can be multiple slots on active nodes.
// A task is analogous to a “slot” where (on a node) the scheduler places a container.
func (u *replicatedConsoleLogUpdater) tasksBySlot(tasks []swarm.Task, activeNodes map[string]struct{}) map[int]swarm.Task {
	// if there are multiple tasks with the same slot number, favor the one
	// with the *lowest* desired state. This can happen in restart
	// scenarios.
	tasksBySlot := make(map[int]swarm.Task)
	for _, task := range tasks {
		if numberedStates[task.DesiredState] == 0 || numberedStates[task.Status.State] == 0 {
			continue
		}
		if existingTask, ok := tasksBySlot[task.Slot]; ok {
			if numberedStates[existingTask.DesiredState] < numberedStates[task.DesiredState] {
				continue
			}
			// if the desired states match, observed state breaks
			// ties. This can happen with the "start first" service
			// update mode.
			if numberedStates[existingTask.DesiredState] == numberedStates[task.DesiredState] &&
				numberedStates[existingTask.Status.State] <= numberedStates[task.Status.State] {
				continue
			}
		}
		// if the task is on a node and this node is active, then map this task to a slot
		if task.NodeID != "" {
			if _, nodeActive := activeNodes[task.NodeID]; !nodeActive {
				continue
			}
		}
		tasksBySlot[task.Slot] = task
	}

	return tasksBySlot
}

// terminalState determines if the given state is a terminal state
// meaninig 'higher' than running (see numberedStates)
func terminalState(state swarm.TaskState) bool {
	return numberedStates[state] > numberedStates[swarm.TaskStateRunning]
}

//////// Mappers
// createServiceSpec creates the service spec: https://docs.docker.com/engine/api/v1.32/#operation/ServiceCreate
func createServiceSpec(d *schema.ResourceData) (swarm.ServiceSpec, error) {

	serviceSpec := swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: d.Get("name").(string),
		},
	}

	labels, err := createServiceLabels(d)
	if err != nil {
		return serviceSpec, err
	}
	serviceSpec.Labels = labels

	taskTemplate, err := createServiceTaskSpec(d)
	if err != nil {
		return serviceSpec, err
	}
	serviceSpec.TaskTemplate = taskTemplate

	mode, err := createServiceMode(d)
	if err != nil {
		return serviceSpec, err
	}
	serviceSpec.Mode = mode

	updateConfig, err := createServiceUpdateConfig(d)
	if err != nil {
		return serviceSpec, err
	}
	serviceSpec.UpdateConfig = updateConfig

	rollbackConfig, err := createServiceRollbackConfig(d)
	if err != nil {
		return serviceSpec, err
	}
	serviceSpec.RollbackConfig = rollbackConfig

	endpointSpec, err := createServiceEndpointSpec(d)
	if err != nil {
		return serviceSpec, err
	}
	serviceSpec.EndpointSpec = endpointSpec

	return serviceSpec, nil
}

// createServiceLabels creates the labels for the service
func createServiceLabels(d *schema.ResourceData) (map[string]string, error) {
	if v, ok := d.GetOk("labels"); ok {
		return labelSetToMap(v.(*schema.Set)), nil
	}
	return nil, nil
}

// == start taskSpec
// createServiceTaskSpec creates the task template for the service
func createServiceTaskSpec(d *schema.ResourceData) (swarm.TaskSpec, error) {
	taskSpec := swarm.TaskSpec{}
	if v, ok := d.GetOk("task_spec"); ok {
		if len(v.([]interface{})) > 0 {
			for _, rawTaskSpec := range v.([]interface{}) {
				rawTaskSpec := rawTaskSpec.(map[string]interface{})

				if rawContainerSpec, ok := rawTaskSpec["container_spec"]; ok {
					containerSpec, err := createContainerSpec(rawContainerSpec)
					if err != nil {
						return taskSpec, err
					}
					taskSpec.ContainerSpec = containerSpec
				}

				if rawResourcesSpec, ok := rawTaskSpec["resources"]; ok {
					resources, err := createResources(rawResourcesSpec)
					if err != nil {
						return taskSpec, err
					}
					taskSpec.Resources = resources
				}
				if rawRestartPolicySpec, ok := rawTaskSpec["restart_policy"]; ok {
					restartPolicy, err := createRestartPolicy(rawRestartPolicySpec)
					if err != nil {
						return taskSpec, err
					}
					taskSpec.RestartPolicy = restartPolicy
				}
				if rawPlacementSpec, ok := rawTaskSpec["placement"]; ok {
					placement, err := createPlacement(rawPlacementSpec)
					if err != nil {
						return taskSpec, err
					}
					taskSpec.Placement = placement
				}
				if rawForceUpdate, ok := rawTaskSpec["force_update"]; ok {
					taskSpec.ForceUpdate = uint64(rawForceUpdate.(int))
				}
				if rawRuntimeSpec, ok := rawTaskSpec["runtime"]; ok {
					taskSpec.Runtime = swarm.RuntimeType(rawRuntimeSpec.(string))
				}
				if rawNetworksSpec, ok := rawTaskSpec["networks"]; ok {
					networks, err := createServiceNetworks(rawNetworksSpec)
					if err != nil {
						return taskSpec, err
					}
					taskSpec.Networks = networks
				}
				if rawLogDriverSpec, ok := rawTaskSpec["log_driver"]; ok {
					logDriver, err := createLogDriver(rawLogDriverSpec)
					if err != nil {
						return taskSpec, err
					}
					taskSpec.LogDriver = logDriver
				}
			}
		}
	}
	return taskSpec, nil
}

// createContainerSpec creates the container spec
func createContainerSpec(v interface{}) (*swarm.ContainerSpec, error) {
	containerSpec := swarm.ContainerSpec{}
	if len(v.([]interface{})) > 0 {
		for _, rawContainerSpec := range v.([]interface{}) {
			rawContainerSpec := rawContainerSpec.(map[string]interface{})
			if value, ok := rawContainerSpec["image"]; ok {
				containerSpec.Image = value.(string)
			}
			if value, ok := rawContainerSpec["labels"]; ok {
				containerSpec.Labels = labelSetToMap(value.(*schema.Set))
			}
			if value, ok := rawContainerSpec["command"]; ok {
				containerSpec.Command = stringListToStringSlice(value.([]interface{}))
			}
			if value, ok := rawContainerSpec["args"]; ok {
				containerSpec.Args = stringListToStringSlice(value.([]interface{}))
			}
			if value, ok := rawContainerSpec["hostname"]; ok {
				containerSpec.Hostname = value.(string)
			}
			if value, ok := rawContainerSpec["env"]; ok {
				containerSpec.Env = mapTypeMapValsToStringSlice(value.(map[string]interface{}))
			}
			if value, ok := rawContainerSpec["dir"]; ok {
				containerSpec.Dir = value.(string)
			}
			if value, ok := rawContainerSpec["user"]; ok {
				containerSpec.User = value.(string)
			}
			if value, ok := rawContainerSpec["groups"]; ok {
				containerSpec.Groups = stringListToStringSlice(value.([]interface{}))
			}
			if value, ok := rawContainerSpec["privileges"]; ok {
				if len(value.([]interface{})) > 0 {
					containerSpec.Privileges = &swarm.Privileges{}

					for _, rawPrivilegesSpec := range value.([]interface{}) {
						rawPrivilegesSpec := rawPrivilegesSpec.(map[string]interface{})

						if value, ok := rawPrivilegesSpec["credential_spec"]; ok {
							if len(value.([]interface{})) > 0 {
								containerSpec.Privileges.CredentialSpec = &swarm.CredentialSpec{}
								for _, rawCredentialSpec := range value.([]interface{}) {
									rawCredentialSpec := rawCredentialSpec.(map[string]interface{})
									if value, ok := rawCredentialSpec["file"]; ok {
										containerSpec.Privileges.CredentialSpec.File = value.(string)
									}
									if value, ok := rawCredentialSpec["registry"]; ok {
										containerSpec.Privileges.CredentialSpec.File = value.(string)
									}
								}
							}
						}
						if value, ok := rawPrivilegesSpec["se_linux_context"]; ok {
							if len(value.([]interface{})) > 0 {
								containerSpec.Privileges.SELinuxContext = &swarm.SELinuxContext{}
								for _, rawSELinuxContext := range value.([]interface{}) {
									rawSELinuxContext := rawSELinuxContext.(map[string]interface{})
									if value, ok := rawSELinuxContext["disable"]; ok {
										containerSpec.Privileges.SELinuxContext.Disable = value.(bool)
									}
									if value, ok := rawSELinuxContext["user"]; ok {
										containerSpec.Privileges.SELinuxContext.User = value.(string)
									}
									if value, ok := rawSELinuxContext["role"]; ok {
										containerSpec.Privileges.SELinuxContext.Role = value.(string)
									}
									if value, ok := rawSELinuxContext["type"]; ok {
										containerSpec.Privileges.SELinuxContext.Type = value.(string)
									}
									if value, ok := rawSELinuxContext["level"]; ok {
										containerSpec.Privileges.SELinuxContext.Level = value.(string)
									}
								}
							}
						}
					}
				}
			}
			if value, ok := rawContainerSpec["read_only"]; ok {
				containerSpec.ReadOnly = value.(bool)
			}
			if value, ok := rawContainerSpec["mounts"]; ok {
				mounts := []mount.Mount{}

				for _, rawMount := range value.(*schema.Set).List() {
					rawMount := rawMount.(map[string]interface{})
					mountType := mount.Type(rawMount["type"].(string))
					mountInstance := mount.Mount{
						Type:   mountType,
						Target: rawMount["target"].(string),
						Source: rawMount["source"].(string),
					}
					if value, ok := rawMount["read_only"]; ok {
						mountInstance.ReadOnly = value.(bool)
					}

					if mountType == mount.TypeBind {
						if value, ok := rawMount["bind_options"]; ok {
							if len(value.([]interface{})) > 0 {
								mountInstance.BindOptions = &mount.BindOptions{}
								for _, rawBindOptions := range value.([]interface{}) {
									rawBindOptions := rawBindOptions.(map[string]interface{})
									if value, ok := rawBindOptions["propagation"]; ok {
										mountInstance.BindOptions.Propagation = mount.Propagation(value.(string))
									}
								}
							}
						}
					} else if mountType == mount.TypeVolume {
						if value, ok := rawMount["volume_options"]; ok {
							if len(value.([]interface{})) > 0 {
								mountInstance.VolumeOptions = &mount.VolumeOptions{}
								for _, rawVolumeOptions := range value.([]interface{}) {
									rawVolumeOptions := rawVolumeOptions.(map[string]interface{})
									if value, ok := rawVolumeOptions["no_copy"]; ok {
										mountInstance.VolumeOptions.NoCopy = value.(bool)
									}
									if value, ok := rawVolumeOptions["labels"]; ok {
										mountInstance.VolumeOptions.Labels = labelSetToMap(value.(*schema.Set))
									}
									// because it is not possible to nest maps
									if value, ok := rawVolumeOptions["driver_name"]; ok {
										if mountInstance.VolumeOptions.DriverConfig == nil {
											mountInstance.VolumeOptions.DriverConfig = &mount.Driver{}
										}
										mountInstance.VolumeOptions.DriverConfig.Name = value.(string)
									}
									if value, ok := rawVolumeOptions["driver_options"]; ok {
										if mountInstance.VolumeOptions.DriverConfig == nil {
											mountInstance.VolumeOptions.DriverConfig = &mount.Driver{}
										}
										mountInstance.VolumeOptions.DriverConfig.Options = mapTypeMapValsToString(value.(map[string]interface{}))
									}
								}
							}
						}
					} else if mountType == mount.TypeTmpfs {
						if value, ok := rawMount["tmpfs_options"]; ok {
							if len(value.([]interface{})) > 0 {
								mountInstance.TmpfsOptions = &mount.TmpfsOptions{}
								for _, rawTmpfsOptions := range value.([]interface{}) {
									rawTmpfsOptions := rawTmpfsOptions.(map[string]interface{})
									if value, ok := rawTmpfsOptions["size_bytes"]; ok {
										mountInstance.TmpfsOptions.SizeBytes = value.(int64)
									}
									if value, ok := rawTmpfsOptions["mode"]; ok {
										mountInstance.TmpfsOptions.Mode = os.FileMode(value.(int))
									}
								}
							}
						}
					}

					mounts = append(mounts, mountInstance)
				}

				containerSpec.Mounts = mounts
			}
			if value, ok := rawContainerSpec["stop_signal"]; ok {
				containerSpec.StopSignal = value.(string)
			}
			if value, ok := rawContainerSpec["stop_grace_period"]; ok {
				parsed, _ := time.ParseDuration(value.(string))
				containerSpec.StopGracePeriod = &parsed
			}
			if value, ok := rawContainerSpec["healthcheck"]; ok {
				containerSpec.Healthcheck = &container.HealthConfig{}
				if len(value.([]interface{})) > 0 {
					for _, rawHealthCheck := range value.([]interface{}) {
						rawHealthCheck := rawHealthCheck.(map[string]interface{})
						if testCommand, ok := rawHealthCheck["test"]; ok {
							containerSpec.Healthcheck.Test = stringListToStringSlice(testCommand.([]interface{}))
						}
						if rawInterval, ok := rawHealthCheck["interval"]; ok {
							containerSpec.Healthcheck.Interval, _ = time.ParseDuration(rawInterval.(string))
						}
						if rawTimeout, ok := rawHealthCheck["timeout"]; ok {
							containerSpec.Healthcheck.Timeout, _ = time.ParseDuration(rawTimeout.(string))
						}
						if rawStartPeriod, ok := rawHealthCheck["start_period"]; ok {
							containerSpec.Healthcheck.StartPeriod, _ = time.ParseDuration(rawStartPeriod.(string))
						}
						if rawRetries, ok := rawHealthCheck["retries"]; ok {
							containerSpec.Healthcheck.Retries, _ = rawRetries.(int)
						}
					}
				}
			}
			if value, ok := rawContainerSpec["hosts"]; ok {
				containerSpec.Hosts = extraHostsSetToDockerExtraHosts(value.(*schema.Set))
			}
			if value, ok := rawContainerSpec["dns_config"]; ok {
				containerSpec.DNSConfig = &swarm.DNSConfig{}
				if len(v.([]interface{})) > 0 {
					for _, rawDNSConfig := range value.([]interface{}) {
						if rawDNSConfig != nil {
							rawDNSConfig := rawDNSConfig.(map[string]interface{})
							if nameservers, ok := rawDNSConfig["nameservers"]; ok {
								containerSpec.DNSConfig.Nameservers = stringListToStringSlice(nameservers.([]interface{}))
							}
							if search, ok := rawDNSConfig["search"]; ok {
								containerSpec.DNSConfig.Search = stringListToStringSlice(search.([]interface{}))
							}
							if options, ok := rawDNSConfig["options"]; ok {
								containerSpec.DNSConfig.Options = stringListToStringSlice(options.([]interface{}))
							}
						}
					}
				}
			}
			if value, ok := rawContainerSpec["secrets"]; ok {
				secrets := []*swarm.SecretReference{}

				for _, rawSecret := range value.(*schema.Set).List() {
					rawSecret := rawSecret.(map[string]interface{})
					rawFilemode := rawSecret["file_mode"].(int)
					secret := swarm.SecretReference{
						SecretID: rawSecret["secret_id"].(string),
						File: &swarm.SecretReferenceFileTarget{
							Name: rawSecret["file_name"].(string),
							UID:  rawSecret["file_uid"].(string),
							GID:  rawSecret["file_gid"].(string),
							Mode: os.FileMode(uint32(rawFilemode)),
						},
					}
					if value, ok := rawSecret["secret_name"]; ok {
						secret.SecretName = value.(string)
					}
					secrets = append(secrets, &secret)
				}
				containerSpec.Secrets = secrets
			}
			if value, ok := rawContainerSpec["configs"]; ok {
				configs := []*swarm.ConfigReference{}

				for _, rawConfig := range value.(*schema.Set).List() {
					rawConfig := rawConfig.(map[string]interface{})
					rawFilemode := rawConfig["file_mode"].(int)
					config := swarm.ConfigReference{
						ConfigID: rawConfig["config_id"].(string),
						File: &swarm.ConfigReferenceFileTarget{
							Name: rawConfig["file_name"].(string),
							UID:  rawConfig["file_uid"].(string),
							GID:  rawConfig["file_gid"].(string),
							Mode: os.FileMode(uint32(rawFilemode)),
						},
					}
					if value, ok := rawConfig["config_name"]; ok {
						config.ConfigName = value.(string)
					}
					configs = append(configs, &config)
				}
				containerSpec.Configs = configs
			}
			if value, ok := rawContainerSpec["isolation"]; ok {
				containerSpec.Isolation = container.Isolation(value.(string))
			}
		}
	}

	return &containerSpec, nil
}

// createResources creates the resource requirements for the service
func createResources(v interface{}) (*swarm.ResourceRequirements, error) {
	resources := swarm.ResourceRequirements{}
	if len(v.([]interface{})) > 0 {
		for _, rawResourcesSpec := range v.([]interface{}) {
			if rawResourcesSpec != nil {
				rawResourcesSpec := rawResourcesSpec.(map[string]interface{})
				if value, ok := rawResourcesSpec["limits"]; ok {
					if len(value.([]interface{})) > 0 {
						resources.Limits = &swarm.Resources{}
						for _, rawLimitsSpec := range value.([]interface{}) {
							rawLimitsSpec := rawLimitsSpec.(map[string]interface{})
							if value, ok := rawLimitsSpec["nano_cpus"]; ok {
								resources.Limits.NanoCPUs = int64(value.(int))
							}
							if value, ok := rawLimitsSpec["memory_bytes"]; ok {
								resources.Limits.MemoryBytes = int64(value.(int))
							}
							if value, ok := rawLimitsSpec["generic_resources"]; ok {
								resources.Limits.GenericResources, _ = createGenericResources(value)
							}
						}
					}
				}
				if value, ok := rawResourcesSpec["reservation"]; ok {
					if len(value.([]interface{})) > 0 {
						resources.Reservations = &swarm.Resources{}
						for _, rawReservationSpec := range value.([]interface{}) {
							rawReservationSpec := rawReservationSpec.(map[string]interface{})
							if value, ok := rawReservationSpec["nano_cpus"]; ok {
								resources.Reservations.NanoCPUs = int64(value.(int))
							}
							if value, ok := rawReservationSpec["memory_bytes"]; ok {
								resources.Reservations.MemoryBytes = int64(value.(int))
							}
							if value, ok := rawReservationSpec["generic_resources"]; ok {
								resources.Reservations.GenericResources, _ = createGenericResources(value)
							}
						}
					}
				}
			}
		}
	}
	return &resources, nil
}

// createGenericResources creates generic resources for a container
func createGenericResources(value interface{}) ([]swarm.GenericResource, error) {
	genericResources := make([]swarm.GenericResource, 0)
	if len(value.([]interface{})) > 0 {
		for _, rawGenericResource := range value.([]interface{}) {
			rawGenericResource := rawGenericResource.(map[string]interface{})

			if rawNamedResources, ok := rawGenericResource["named_resources_spec"]; ok {
				for _, rawNamedResource := range rawNamedResources.(*schema.Set).List() {
					namedGenericResource := &swarm.NamedGenericResource{}
					splitted := strings.Split(rawNamedResource.(string), "=")
					namedGenericResource.Kind = splitted[0]
					namedGenericResource.Value = splitted[1]

					genericResource := swarm.GenericResource{}
					genericResource.NamedResourceSpec = namedGenericResource
					genericResources = append(genericResources, genericResource)
				}
			}
			if rawDiscreteResources, ok := rawGenericResource["discrete_resources_spec"]; ok {
				for _, rawDiscreteResource := range rawDiscreteResources.(*schema.Set).List() {
					discreteGenericResource := &swarm.DiscreteGenericResource{}
					splitted := strings.Split(rawDiscreteResource.(string), "=")
					discreteGenericResource.Kind = splitted[0]
					discreteGenericResource.Value, _ = strconv.ParseInt(splitted[1], 10, 64)

					genericResource := swarm.GenericResource{}
					genericResource.DiscreteResourceSpec = discreteGenericResource
					genericResources = append(genericResources, genericResource)
				}
			}
		}
	}
	return genericResources, nil
}

// createRestartPolicy creates the restart poliyc of the service
func createRestartPolicy(v interface{}) (*swarm.RestartPolicy, error) {
	restartPolicy := swarm.RestartPolicy{}
	rawRestartPolicy := v.(map[string]interface{})

	if v, ok := rawRestartPolicy["condition"]; ok {
		restartPolicy.Condition = swarm.RestartPolicyCondition(v.(string))
	}
	if v, ok := rawRestartPolicy["delay"]; ok {
		parsed, _ := time.ParseDuration(v.(string))
		restartPolicy.Delay = &parsed
	}
	if v, ok := rawRestartPolicy["max_attempts"]; ok {
		parsed, _ := strconv.ParseInt(v.(string), 10, 64)
		mapped := uint64(parsed)
		restartPolicy.MaxAttempts = &mapped
	}
	if v, ok := rawRestartPolicy["window"]; ok {
		parsed, _ := time.ParseDuration(v.(string))
		restartPolicy.Window = &parsed
	}
	return &restartPolicy, nil
}

// createPlacement creates the placement strategy for the service
func createPlacement(v interface{}) (*swarm.Placement, error) {
	placement := swarm.Placement{}
	if len(v.([]interface{})) > 0 {
		for _, rawPlacement := range v.([]interface{}) {
			if rawPlacement != nil {
				rawPlacement := rawPlacement.(map[string]interface{})
				if v, ok := rawPlacement["constraints"]; ok {
					placement.Constraints = stringSetToStringSlice(v.(*schema.Set))
				}
				if v, ok := rawPlacement["prefs"]; ok {
					placement.Preferences = stringSetToPlacementPrefs(v.(*schema.Set))
				}
				if v, ok := rawPlacement["platforms"]; ok {
					placement.Platforms = mapSetToPlacementPlatforms(v.(*schema.Set))
				}
			}
		}
	}

	return &placement, nil
}

// createServiceNetworks creates the networks the service will be attachted to
func createServiceNetworks(v interface{}) ([]swarm.NetworkAttachmentConfig, error) {
	networks := []swarm.NetworkAttachmentConfig{}
	if len(v.(*schema.Set).List()) > 0 {
		for _, rawNetwork := range v.(*schema.Set).List() {
			network := swarm.NetworkAttachmentConfig{
				Target: rawNetwork.(string),
			}
			networks = append(networks, network)
		}
	}
	return networks, nil
}

// createLogDriver creates the log driver for the service
func createLogDriver(v interface{}) (*swarm.Driver, error) {
	logDriver := swarm.Driver{}
	if len(v.([]interface{})) > 0 {
		for _, rawLogging := range v.([]interface{}) {
			rawLogging := rawLogging.(map[string]interface{})
			if rawName, ok := rawLogging["name"]; ok {
				logDriver.Name = rawName.(string)
			}
			if rawOptions, ok := rawLogging["options"]; ok {
				logDriver.Options = mapTypeMapValsToString(rawOptions.(map[string]interface{}))
			}
			return &logDriver, nil
		}
	}
	return nil, nil
}

// == end taskSpec

// createServiceMode creates the mode the service will run in
func createServiceMode(d *schema.ResourceData) (swarm.ServiceMode, error) {
	serviceMode := swarm.ServiceMode{}
	if v, ok := d.GetOk("mode"); ok {
		// because its a list
		if len(v.([]interface{})) > 0 {
			for _, rawMode := range v.([]interface{}) {
				// with a map
				rawMode := rawMode.(map[string]interface{})

				if rawReplicatedMode, replModeOk := rawMode["replicated"]; replModeOk {
					// with a list
					if len(rawReplicatedMode.([]interface{})) > 0 {
						for _, rawReplicatedModeInt := range rawReplicatedMode.([]interface{}) {
							// which is a map
							rawReplicatedModeMap := rawReplicatedModeInt.(map[string]interface{})
							log.Printf("[INFO] Setting service mode to 'replicated'")
							serviceMode.Replicated = &swarm.ReplicatedService{}
							if testReplicas, testReplicasOk := rawReplicatedModeMap["replicas"]; testReplicasOk {
								log.Printf("[INFO] Setting %v replicas", testReplicas)
								replicas := uint64(testReplicas.(int))
								serviceMode.Replicated.Replicas = &replicas
							}
						}
					}
				}
				if rawGlobalMode, globalModeOk := rawMode["global"]; globalModeOk && rawGlobalMode.(bool) {
					log.Printf("[INFO] Setting service mode to 'global' is %v", rawGlobalMode)
					serviceMode.Global = &swarm.GlobalService{}
				}
			}
		}
	}
	return serviceMode, nil
}

// createServiceUpdateConfig creates the service update config
func createServiceUpdateConfig(d *schema.ResourceData) (*swarm.UpdateConfig, error) {
	if v, ok := d.GetOk("update_config"); ok {
		return createUpdateOrRollbackConfig(v.([]interface{}))
	}
	return nil, nil
}

// createServiceRollbackConfig create the service rollback config
func createServiceRollbackConfig(d *schema.ResourceData) (*swarm.UpdateConfig, error) {
	if v, ok := d.GetOk("rollback_config"); ok {
		return createUpdateOrRollbackConfig(v.([]interface{}))
	}
	return nil, nil
}

// == start endpointSpec
// createServiceEndpointSpec creates the spec for the endpoint
func createServiceEndpointSpec(d *schema.ResourceData) (*swarm.EndpointSpec, error) {
	endpointSpec := swarm.EndpointSpec{}
	if v, ok := d.GetOk("endpoint_spec"); ok {
		if len(v.([]interface{})) > 0 {
			for _, rawEndpointSpec := range v.([]interface{}) {
				if rawEndpointSpec != nil {
					rawEndpointSpec := rawEndpointSpec.(map[string]interface{})
					if value, ok := rawEndpointSpec["mode"]; ok {
						endpointSpec.Mode = swarm.ResolutionMode(value.(string))
					}
					if value, ok := rawEndpointSpec["ports"]; ok {
						endpointSpec.Ports = portSetToServicePorts(value)
					}
				}
			}
		}
	}

	return &endpointSpec, nil
}

// portSetToServicePorts maps a set of ports to portConfig
func portSetToServicePorts(v interface{}) []swarm.PortConfig {
	retPortConfigs := []swarm.PortConfig{}
	if len(v.([]interface{})) > 0 {
		for _, portInt := range v.([]interface{}) {
			portConfig := swarm.PortConfig{}
			rawPort := portInt.(map[string]interface{})
			if value, ok := rawPort["name"]; ok {
				portConfig.Name = value.(string)
			}
			if value, ok := rawPort["protocol"]; ok {
				portConfig.Protocol = swarm.PortConfigProtocol(value.(string))
			}
			if value, ok := rawPort["target_port"]; ok {
				portConfig.TargetPort = uint32(value.(int))
			}
			if externalPort, ok := rawPort["published_port"]; ok {
				portConfig.PublishedPort = uint32(externalPort.(int))
			}
			if value, ok := rawPort["publish_mode"]; ok {
				portConfig.PublishMode = swarm.PortConfigPublishMode(value.(string))
			}

			retPortConfigs = append(retPortConfigs, portConfig)
		}
	}

	return retPortConfigs
}

// == end endpointSpec

// createUpdateOrRollbackConfig create the configuration for and update or rollback
func createUpdateOrRollbackConfig(config []interface{}) (*swarm.UpdateConfig, error) {
	updateConfig := swarm.UpdateConfig{}
	if len(config) > 0 {
		sc := config[0].(map[string]interface{})
		if v, ok := sc["parallelism"]; ok {
			updateConfig.Parallelism = uint64(v.(int))
		}
		if v, ok := sc["delay"]; ok {
			updateConfig.Delay, _ = time.ParseDuration(v.(string))
		}
		if v, ok := sc["failure_action"]; ok {
			updateConfig.FailureAction = v.(string)
		}
		if v, ok := sc["monitor"]; ok {
			updateConfig.Monitor, _ = time.ParseDuration(v.(string))
		}
		if v, ok := sc["max_failure_ratio"]; ok {
			value, _ := strconv.ParseFloat(v.(string), 64)
			updateConfig.MaxFailureRatio = float32(value)
		}
		if v, ok := sc["order"]; ok {
			updateConfig.Order = v.(string)
		}
	}

	return &updateConfig, nil
}

// createConvergeConfig creates the configuration for converging
func createConvergeConfig(config []interface{}) *convergeConfig {
	plainConvergeConfig := &convergeConfig{}
	if len(config) > 0 {
		for _, rawConvergeConfig := range config {
			rawConvergeConfig := rawConvergeConfig.(map[string]interface{})
			if delay, ok := rawConvergeConfig["delay"]; ok {
				plainConvergeConfig.delay, _ = time.ParseDuration(delay.(string))
			}
			if timeout, ok := rawConvergeConfig["timeout"]; ok {
				plainConvergeConfig.timeoutRaw, _ = timeout.(string)
				plainConvergeConfig.timeout, _ = time.ParseDuration(timeout.(string))
			}
		}
	}
	return plainConvergeConfig
}

// authToServiceAuth maps the auth to AuthConfiguration
func authToServiceAuth(auth map[string]interface{}) types.AuthConfig {
	if auth["username"] != nil && len(auth["username"].(string)) > 0 && auth["password"] != nil && len(auth["password"].(string)) > 0 {
		return types.AuthConfig{
			Username:      auth["username"].(string),
			Password:      auth["password"].(string),
			ServerAddress: auth["server_address"].(string),
		}
	}

	return types.AuthConfig{}
}

// fromRegistryAuth extract the desired AuthConfiguration for the given image
func fromRegistryAuth(image string, authConfigs map[string]types.AuthConfig) types.AuthConfig {
	// Remove normalized prefixes to simplify substring
	image = strings.Replace(strings.Replace(image, "http://", "", 1), "https://", "", 1)
	// Get the registry with optional port
	lastBin := strings.Index(image, "/")
	// No auth given and image name has no slash like 'alpine:3.1'
	if lastBin != -1 {
		serverAddress := image[0:lastBin]
		if fromRegistryAuth, ok := authConfigs[normalizeRegistryAddress(serverAddress)]; ok {
			return fromRegistryAuth
		}
	}

	return types.AuthConfig{}
}

// retrieveAndMarshalAuth retrieves and marshals the service registry auth
func retrieveAndMarshalAuth(d *schema.ResourceData, meta interface{}, stageType string) []byte {
	auth := types.AuthConfig{}
	if v, ok := d.GetOk("auth"); ok {
		auth = authToServiceAuth(v.(map[string]interface{}))
	} else {
		authConfigs := meta.(*ProviderConfig).AuthConfigs.Configs
		if len(authConfigs) == 0 {
			log.Printf("[DEBUG] AuthConfigs empty on %s. Wait 3s and try again", stageType)
			// sometimes the dockerconfig is read succesfully from disk but the
			// call to create/update the service is faster. So we delay to prevent the
			// passing of an empty auth configuration in this case
			<-time.After(3 * time.Second)
			authConfigs = meta.(*ProviderConfig).AuthConfigs.Configs
		}
		log.Printf("[DEBUG] Getting configs from '%v'", authConfigs)
		auth = fromRegistryAuth(d.Get("task_spec.0.container_spec.0.image").(string), authConfigs)
	}

	marshalledAuth, _ := json.Marshal(auth) // https://docs.docker.com/engine/api/v1.37/#section/Versioning
	return marshalledAuth
}

// stringSetToPlacementPrefs maps a string set to PlacementPreference
func stringSetToPlacementPrefs(stringSet *schema.Set) []swarm.PlacementPreference {
	ret := []swarm.PlacementPreference{}
	if stringSet == nil {
		return ret
	}
	for _, envVal := range stringSet.List() {
		ret = append(ret, swarm.PlacementPreference{
			Spread: &swarm.SpreadOver{
				SpreadDescriptor: envVal.(string),
			},
		})
	}
	return ret
}

// mapSetToPlacementPlatforms maps a string set to Platform
func mapSetToPlacementPlatforms(stringSet *schema.Set) []swarm.Platform {
	ret := []swarm.Platform{}
	if stringSet == nil {
		return ret
	}

	for _, rawPlatform := range stringSet.List() {
		rawPlatform := rawPlatform.(map[string]interface{})
		ret = append(ret, swarm.Platform{
			Architecture: rawPlatform["architecture"].(string),
			OS:           rawPlatform["os"].(string),
		})
	}

	return ret
}

//////// States

// numberedStates are ascending sorted states for docker tasks
// meaning they appear internally in this order in the statemachine
var (
	numberedStates = map[swarm.TaskState]int64{
		swarm.TaskStateNew:       1,
		swarm.TaskStateAllocated: 2,
		swarm.TaskStatePending:   3,
		swarm.TaskStateAssigned:  4,
		swarm.TaskStateAccepted:  5,
		swarm.TaskStatePreparing: 6,
		swarm.TaskStateReady:     7,
		swarm.TaskStateStarting:  8,
		swarm.TaskStateRunning:   9,

		// The following states are not actually shown in progress
		// output, but are used internally for ordering.
		swarm.TaskStateComplete: 10,
		swarm.TaskStateShutdown: 11,
		swarm.TaskStateFailed:   12,
		swarm.TaskStateRejected: 13,
	}

	longestState int
)

// serviceCreatePendingStates are the pending states for the creation of a service
var serviceCreatePendingStates = []string{
	"new",
	"allocated",
	"pending",
	"assigned",
	"accepted",
	"preparing",
	"ready",
	"starting",
	"creating",
	"paused",
}

// serviceUpdatePendingStates are the pending states for the update of a service
var serviceUpdatePendingStates = []string{
	"creating",
	"updating",
}
