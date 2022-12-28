package provider

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type convergeConfig struct {
	timeout    time.Duration
	timeoutRaw string
	delay      time.Duration
}

// ///////////////
// TF CRUD funcs
// ///////////////
func resourceDockerServiceCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	var err error
	client := meta.(*ProviderConfig).DockerClient

	serviceSpec, err := createServiceSpec(d)
	if err != nil {
		return diag.FromErr(err)
	}

	serviceOptions := types.ServiceCreateOptions{}
	marshalledAuth := retrieveAndMarshalAuth(d, meta, "create")
	serviceOptions.EncodedRegistryAuth = base64.URLEncoding.EncodeToString(marshalledAuth)
	serviceOptions.QueryRegistry = true
	log.Printf("[DEBUG] Passing registry auth '%s'", serviceOptions.EncodedRegistryAuth)

	service, err := client.ServiceCreate(ctx, serviceSpec, serviceOptions)
	if err != nil {
		return diag.FromErr(err)
	}
	if v, ok := d.GetOk("converge_config"); ok {
		convergeConfig := createConvergeConfig(v.([]interface{}))
		log.Printf("[INFO] Waiting for Service '%s' to be created with timeout: %v", service.ID, convergeConfig.timeoutRaw)
		timeout, _ := time.ParseDuration(convergeConfig.timeoutRaw)
		stateConf := &resource.StateChangeConf{
			Pending:    serviceCreatePendingStates,
			Target:     []string{"running", "complete"},
			Refresh:    resourceDockerServiceCreateRefreshFunc(ctx, service.ID, meta),
			Timeout:    timeout,
			MinTimeout: 5 * time.Second,
			Delay:      convergeConfig.delay,
		}

		// Wait, catching any errors
		_, err := stateConf.WaitForStateContext(ctx)
		if err != nil {
			// the service will be deleted in case it cannot be converged
			if deleteErr := deleteService(ctx, service.ID, d, client); deleteErr != nil {
				return diag.FromErr(deleteErr)
			}
			if containsIgnorableErrorMessage(err.Error(), "timeout while waiting for state") {
				return diag.FromErr(&DidNotConvergeError{ServiceID: service.ID, Timeout: convergeConfig.timeout})
			}
			return diag.FromErr(err)
		}
	}

	d.SetId(service.ID)
	return resourceDockerServiceRead(ctx, d, meta)
}

func resourceDockerServiceRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	log.Printf("[INFO] Waiting for service: '%s' to expose all fields: max '%v seconds'", d.Id(), 30)

	stateConf := &resource.StateChangeConf{
		Pending:    []string{"pending"},
		Target:     []string{"all_fields", "removed"},
		Refresh:    resourceDockerServiceReadRefreshFunc(ctx, d, meta),
		Timeout:    30 * time.Second,
		MinTimeout: 5 * time.Second,
		Delay:      2 * time.Second,
	}

	// Wait, catching any errors
	_, err := stateConf.WaitForStateContext(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	return nil
}

func resourceDockerServiceReadRefreshFunc(ctx context.Context,
	d *schema.ResourceData, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		client := meta.(*ProviderConfig).DockerClient
		serviceID := d.Id()

		apiService, err := fetchDockerService(ctx, serviceID, d.Get("name").(string), client)
		if err != nil {
			return nil, "", err
		}
		if apiService == nil {
			log.Printf("[WARN] Service (%s) not found, removing from state", serviceID)
			d.SetId("")
			return serviceID, "removed", nil
		}
		service, _, err := client.ServiceInspectWithRaw(ctx, apiService.ID, types.ServiceInspectOptions{})
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

		if err = d.Set("task_spec", flattenTaskSpec(service.Spec.TaskTemplate, d)); err != nil {
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

func resourceDockerServiceUpdate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient

	service, _, err := client.ServiceInspectWithRaw(ctx, d.Id(), types.ServiceInspectOptions{})
	if err != nil {
		return diag.FromErr(err)
	}

	serviceSpec, err := createServiceSpec(d)
	if err != nil {
		return diag.FromErr(err)
	}

	updateOptions := types.ServiceUpdateOptions{}
	marshalledAuth := retrieveAndMarshalAuth(d, meta, "update")
	if err != nil {
		return diag.Errorf("error creating auth config: %s", err)
	}
	updateOptions.EncodedRegistryAuth = base64.URLEncoding.EncodeToString(marshalledAuth)

	updateResponse, err := client.ServiceUpdate(ctx, d.Id(), service.Version, serviceSpec, updateOptions)
	if err != nil {
		return diag.FromErr(err)
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
			Refresh:    resourceDockerServiceUpdateRefreshFunc(ctx, service.ID, meta),
			Timeout:    timeout,
			MinTimeout: 5 * time.Second,
			Delay:      7 * time.Second,
		}

		// Wait, catching any errors
		state, err := stateConf.WaitForStateContext(ctx)
		log.Printf("[INFO] State awaited: %v with error: %v", state, err)
		if err != nil {
			if containsIgnorableErrorMessage(err.Error(), "timeout while waiting for state") {
				return diag.FromErr(&DidNotConvergeError{ServiceID: service.ID, Timeout: convergeConfig.timeout})
			}
			return diag.FromErr(err)
		}
	}

	return resourceDockerServiceRead(ctx, d, meta)
}

func resourceDockerServiceDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	client := meta.(*ProviderConfig).DockerClient

	if err := deleteService(ctx, d.Id(), d, client); err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")
	return nil
}

// ///////////////
// Helpers
// ///////////////
// fetchDockerService fetches a service by its name or id
func fetchDockerService(ctx context.Context, ID string, name string, client *client.Client) (*swarm.Service, error) {
	apiServices, err := client.ServiceList(ctx, types.ServiceListOptions{})
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
func deleteService(ctx context.Context, serviceID string, d *schema.ResourceData, client *client.Client) error {
	// get containerIDs of the running service because they do not exist after the service is deleted
	serviceContainerIds := make([]string, 0)
	if v, ok := d.GetOk("task_spec.0.container_spec.0.stop_grace_period"); ok && v.(string) != "0s" {
		filters := filters.NewArgs()
		filters.Add("service", d.Get("name").(string))
		tasks, err := client.TaskList(ctx, types.TaskListOptions{
			Filters: filters,
		})
		if err != nil {
			return err
		}
		for _, t := range tasks {
			task, _, _ := client.TaskInspectWithRaw(ctx, t.ID)
			containerID := ""
			if task.Status.ContainerStatus != nil {
				containerID = task.Status.ContainerStatus.ContainerID
			}
			log.Printf("[INFO] Found container with ID ['%s'] in state '%s' for destroying", containerID, task.Status.State)
			if strings.TrimSpace(containerID) != "" && task.Status.State != swarm.TaskStateShutdown {
				serviceContainerIds = append(serviceContainerIds, containerID)
			}
		}
	}

	// delete the service
	log.Printf("[INFO] Deleting service with ID: '%s'", serviceID)
	if err := client.ServiceRemove(ctx, serviceID); err != nil {
		return fmt.Errorf("Error deleting service with ID '%s': %s", serviceID, err)
	}

	// destroy each container after a grace period if specified
	if v, ok := d.GetOk("task_spec.0.container_spec.0.stop_grace_period"); ok && v.(string) != "0s" {
		for _, containerID := range serviceContainerIds {
			destroyGraceTime, _ := time.ParseDuration(v.(string))
			log.Printf("[INFO] Waiting for container with ID: '%s' to exit: max %v", containerID, destroyGraceTime)
			ctx, cancel := context.WithTimeout(ctx, destroyGraceTime)
			// We defer explicitly to avoid context leaks
			defer cancel()

			containerWaitChan, containerWaitErrChan := client.ContainerWait(ctx, containerID, container.WaitConditionRemoved)
			select {
			case containerWaitResult := <-containerWaitChan:
				if containerWaitResult.Error != nil {
					// We ignore those types of errors because the container might be already removed before
					// the containerWait returns
					if !(containsIgnorableErrorMessage(containerWaitResult.Error.Message, "No such container")) {
						return fmt.Errorf("failed to wait for container with ID '%s': '%v'", containerID, containerWaitResult.Error.Message)
					}
				}
				log.Printf("[INFO] Container with ID '%s' exited with code '%v'", containerID, containerWaitResult.StatusCode)
			case containerWaitErrResult := <-containerWaitErrChan:
				// We ignore those types of errors because the container might be already removed before
				// the containerWait returns
				if !(containsIgnorableErrorMessage(containerWaitErrResult.Error(), "No such container")) {
					return fmt.Errorf("error on wait for container with ID '%s': %v", containerID, containerWaitErrResult)
				}
			}

			removeOpts := types.ContainerRemoveOptions{
				RemoveVolumes: true,
				Force:         true,
			}

			log.Printf("[INFO] Removing container with ID: '%s'", containerID)
			if err := client.ContainerRemove(ctx, containerID, removeOpts); err != nil {
				// We ignore those types of errors because the container might be already removed of the removal is in progress
				// before the containerRemove call happens
				if !containsIgnorableErrorMessage(err.Error(), "No such container", "is already in progress") {
					return fmt.Errorf("Error deleting container with ID '%s': %s", containerID, err)
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
func resourceDockerServiceCreateRefreshFunc(ctx context.Context,
	serviceID string, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		client := meta.(*ProviderConfig).DockerClient

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
func resourceDockerServiceUpdateRefreshFunc(ctx context.Context,
	serviceID string, meta interface{}) resource.StateRefreshFunc {
	return func() (interface{}, string, error) {
		client := meta.(*ProviderConfig).DockerClient

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

// authToServiceAuth maps the auth to AuthConfiguration
func authToServiceAuth(auths []interface{}) types.AuthConfig {
	if len(auths) == 0 {
		return types.AuthConfig{}
	}
	// it's maxItems = 1
	auth := auths[0].(map[string]interface{})
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
	// DevSkim: ignore DS137138
	image = strings.Replace(strings.Replace(image, "http://", "", 1), "https://", "", 1)
	// Get the registry with optional port
	lastBin := strings.Index(image, "/")
	// No auth given and image name has no slash like 'alpine:3.1'
	if lastBin != -1 {
		serverAddress := image[0:lastBin]
		if fromRegistryAuth, ok := authConfigs[serverAddress]; ok {
			return fromRegistryAuth
		}
	}

	return types.AuthConfig{}
}

// retrieveAndMarshalAuth retrieves and marshals the service registry auth
func retrieveAndMarshalAuth(d *schema.ResourceData, meta interface{}, stageType string) []byte {
	var auth types.AuthConfig
	// when a service is updated/set for the first time the auth is set but empty
	// this is why we need this additional check
	if rawAuth, ok := d.GetOk("auth"); ok && len(rawAuth.([]interface{})) != 0 {
		log.Printf("[DEBUG] Getting configs from service auth '%v'", rawAuth)
		auth = authToServiceAuth(rawAuth.([]interface{}))
	} else {
		authConfigs := meta.(*ProviderConfig).AuthConfigs.Configs
		log.Printf("[DEBUG] Getting configs from provider auth '%v'", authConfigs)
		auth = fromRegistryAuth(d.Get("task_spec.0.container_spec.0.image").(string), authConfigs)
	}

	marshalledAuth, _ := json.Marshal(auth) // https://docs.docker.com/engine/api/v1.37/#section/Versioning
	return marshalledAuth
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
