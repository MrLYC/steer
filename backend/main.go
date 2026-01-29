/*
Steer - Kubernetes Helm Test Operator Demo
===========================================

This is a minimal runnable demo version of the Steer operator.
It simulates the core functionality without requiring an actual Kubernetes cluster.

Architecture:
1. CRD Models: Define HelmRelease and HelmTestJob structures
2. Mock Storage: In-memory storage for CRD resources
3. Mock Controller: Simulates the reconciliation loop
4. Web API: RESTful API for managing CRDs
5. WebSocket: Real-time updates for UI

Usage:
  go run main.go

The server will start on http://localhost:8080
*/

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// ============================================================================
// CRD Type Definitions
// ============================================================================

type ObjectMeta struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// HelmRelease CRD
type HelmRelease struct {
	APIVersion string            `json:"apiVersion"`
	Kind       string            `json:"kind"`
	Metadata   ObjectMeta        `json:"metadata"`
	Spec       HelmReleaseSpec   `json:"spec"`
	Status     HelmReleaseStatus `json:"status"`
}

type HelmReleaseSpec struct {
	Chart      ChartSpec      `json:"chart"`
	Values     interface{}    `json:"values,omitempty"`
	Deployment DeploymentSpec `json:"deployment"`
	Cleanup    CleanupSpec    `json:"cleanup,omitempty"`
}

type ChartSpec struct {
	Name       string     `json:"name"`
	Version    string     `json:"version,omitempty"`
	Repository string     `json:"repository,omitempty"`
	Git        *GitSource `json:"git,omitempty"`
}

type GitSource struct {
	URL    string `json:"url"`
	Ref    string `json:"ref,omitempty"`
	Path   string `json:"path,omitempty"`
	Branch string `json:"branch,omitempty"`
}

type DeploymentSpec struct {
	Namespace           string `json:"namespace"`
	Timeout             string `json:"timeout,omitempty"`
	MaxRetries          int    `json:"maxRetries,omitempty"`
	WaitAfterDeployment string `json:"waitAfterDeployment,omitempty"`
	AutoUninstallAfter  string `json:"autoUninstallAfter,omitempty"`
}

type CleanupSpec struct {
	DeleteNamespace bool `json:"deleteNamespace,omitempty"`
	DeleteImages    bool `json:"deleteImages,omitempty"`
}

type HelmReleaseStatus struct {
	Phase      string    `json:"phase"` // Pending, Installing, Installed, Failed
	Message    string    `json:"message,omitempty"`
	DeployedAt time.Time `json:"deployedAt,omitempty"`
}

// HelmTestJob CRD
type HelmTestJob struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Metadata   ObjectMeta         `json:"metadata"`
	Spec       HelmTestJobSpec    `json:"spec"`
	Status     HelmTestJobStatus  `json:"status"`
}

type HelmTestJobSpec struct {
	HelmReleaseRef HelmReleaseRef `json:"helmReleaseRef"`
	Schedule       ScheduleSpec   `json:"schedule"`
	Test           TestSpec       `json:"test"`
	Hooks          HooksSpec      `json:"hooks,omitempty"`
	Cleanup        *CleanupSpec   `json:"cleanup,omitempty"`
}

type HelmReleaseRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

type ScheduleSpec struct {
	Type     string `json:"type"` // once, cron
	Delay    string `json:"delay,omitempty"`
	Cron     string `json:"cron,omitempty"`
	Timezone string `json:"timezone,omitempty"`
}

type TestSpec struct {
	Timeout string `json:"timeout,omitempty"`
	Logs    bool   `json:"logs,omitempty"`
	Filter  string `json:"filter,omitempty"`
}

type HooksSpec struct {
	PreTest  []Hook `json:"preTest,omitempty"`
	PostTest []Hook `json:"postTest,omitempty"`
}

type Hook struct {
	Name   string   `json:"name"`
	Type   string   `json:"type"` // script, kubernetes
	Env    []EnvVar `json:"env,omitempty"`
	Script string   `json:"script,omitempty"`
}

type EnvVar struct {
	Name      string        `json:"name"`
	Value     string        `json:"value,omitempty"`
	ValueFrom *EnvVarSource `json:"valueFrom,omitempty"`
}

type EnvVarSource struct {
	FieldPath      string                    `json:"fieldPath,omitempty"`
	HelmReleaseRef *HelmReleaseFieldSelector `json:"helmReleaseRef,omitempty"`
}

type HelmReleaseFieldSelector struct {
	FieldPath string `json:"fieldPath"`
}

type HelmTestJobStatus struct {
	Phase          string       `json:"phase"` // Pending, Running, Succeeded, Failed
	Message        string       `json:"message,omitempty"`
	StartTime      time.Time    `json:"startTime,omitempty"`
	CompletionTime time.Time    `json:"completionTime,omitempty"`
	TestResults    []TestResult `json:"testResults,omitempty"`
	HookResults    []HookResult `json:"hookResults,omitempty"`
}

type TestResult struct {
	Name        string    `json:"name"`
	Phase       string    `json:"phase"`
	StartedAt   time.Time `json:"startedAt"`
	CompletedAt time.Time `json:"completedAt"`
	Message     string    `json:"message,omitempty"`
}

type HookResult struct {
	Name    string `json:"name"`
	Phase   string `json:"phase"`
	Message string `json:"message,omitempty"`
}

// ============================================================================
// Mock Storage
// ============================================================================

type Storage struct {
	mu           sync.RWMutex
	helmReleases map[string]*HelmRelease
	helmTestJobs map[string]*HelmTestJob
}

func NewStorage() *Storage {
	return &Storage{
		helmReleases: make(map[string]*HelmRelease),
		helmTestJobs: make(map[string]*HelmTestJob),
	}
}

func (s *Storage) CreateHelmRelease(hr *HelmRelease) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key := hr.Metadata.Namespace + "/" + hr.Metadata.Name
	if _, exists := s.helmReleases[key]; exists {
		return fmt.Errorf("HelmRelease already exists: %s", key)
	}
	
	hr.Status.Phase = "Pending"
	s.helmReleases[key] = hr
	return nil
}

func (s *Storage) GetHelmRelease(namespace, name string) (*HelmRelease, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	key := namespace + "/" + name
	hr, exists := s.helmReleases[key]
	if !exists {
		return nil, fmt.Errorf("HelmRelease not found: %s", key)
	}
	return hr, nil
}

func (s *Storage) ListHelmReleases() []*HelmRelease {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	list := make([]*HelmRelease, 0, len(s.helmReleases))
	for _, hr := range s.helmReleases {
		list = append(list, hr)
	}
	return list
}

func (s *Storage) UpdateHelmRelease(hr *HelmRelease) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key := hr.Metadata.Namespace + "/" + hr.Metadata.Name
	s.helmReleases[key] = hr
	return nil
}

func (s *Storage) DeleteHelmRelease(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key := namespace + "/" + name
	delete(s.helmReleases, key)
	return nil
}

func (s *Storage) CreateHelmTestJob(tj *HelmTestJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key := tj.Metadata.Namespace + "/" + tj.Metadata.Name
	if _, exists := s.helmTestJobs[key]; exists {
		return fmt.Errorf("HelmTestJob already exists: %s", key)
	}
	
	tj.Status.Phase = "Pending"
	s.helmTestJobs[key] = tj
	return nil
}

func (s *Storage) GetHelmTestJob(namespace, name string) (*HelmTestJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	key := namespace + "/" + name
	tj, exists := s.helmTestJobs[key]
	if !exists {
		return nil, fmt.Errorf("HelmTestJob not found: %s", key)
	}
	return tj, nil
}

func (s *Storage) ListHelmTestJobs() []*HelmTestJob {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	list := make([]*HelmTestJob, 0, len(s.helmTestJobs))
	for _, tj := range s.helmTestJobs {
		list = append(list, tj)
	}
	return list
}

func (s *Storage) UpdateHelmTestJob(tj *HelmTestJob) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key := tj.Metadata.Namespace + "/" + tj.Metadata.Name
	s.helmTestJobs[key] = tj
	return nil
}

func (s *Storage) DeleteHelmTestJob(namespace, name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	key := namespace + "/" + name
	delete(s.helmTestJobs, key)
	return nil
}

// ============================================================================
// Mock Controller
// ============================================================================

type Controller struct {
	storage *Storage
}

func NewController(storage *Storage) *Controller {
	return &Controller{storage: storage}
}

func (c *Controller) Start() {
	go c.reconcileLoop()
}

func (c *Controller) reconcileLoop() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		c.reconcileHelmReleases()
		c.reconcileHelmTestJobs()
	}
}

func (c *Controller) reconcileHelmReleases() {
	releases := c.storage.ListHelmReleases()
	for _, hr := range releases {
		if hr.Status.Phase == "Pending" {
			// Simulate deployment
			hr.Status.Phase = "Installing"
			hr.Status.Message = "Deploying Helm chart..."
			c.storage.UpdateHelmRelease(hr)
			
			// Simulate async deployment
			go func(hr *HelmRelease) {
				time.Sleep(3 * time.Second)
				hr.Status.Phase = "Installed"
				hr.Status.Message = "Helm chart deployed successfully"
				hr.Status.DeployedAt = time.Now()
				c.storage.UpdateHelmRelease(hr)
			}(hr)
		}
	}
}

func (c *Controller) reconcileHelmTestJobs() {
	jobs := c.storage.ListHelmTestJobs()
	for _, tj := range jobs {
		if tj.Status.Phase == "Pending" {
			// Check if HelmRelease is ready
			hr, err := c.storage.GetHelmRelease(tj.Spec.HelmReleaseRef.Namespace, tj.Spec.HelmReleaseRef.Name)
			if err != nil || hr.Status.Phase != "Installed" {
				continue
			}
			
			// Check delay
			if tj.Spec.Schedule.Type == "once" && tj.Spec.Schedule.Delay != "" {
				// Simplified: just start immediately in demo
			}
			
			// Start test
			tj.Status.Phase = "Running"
			tj.Status.Message = "Running tests..."
			tj.Status.StartTime = time.Now()
			c.storage.UpdateHelmTestJob(tj)
			
			// Simulate async test execution
			go func(tj *HelmTestJob, hr *HelmRelease) {
				// Execute preTest hooks
				for _, hook := range tj.Spec.Hooks.PreTest {
					result := c.executeHook(hook, tj, hr)
					tj.Status.HookResults = append(tj.Status.HookResults, result)
				}
				
				time.Sleep(2 * time.Second)
				
				// Simulate test execution
				tj.Status.TestResults = []TestResult{
					{
						Name:        "test-connection",
						Phase:       "Succeeded",
						StartedAt:   time.Now().Add(-2 * time.Second),
						CompletedAt: time.Now(),
						Message:     "Connection test passed",
					},
				}
				
				// Execute postTest hooks
				for _, hook := range tj.Spec.Hooks.PostTest {
					result := c.executeHook(hook, tj, hr)
					tj.Status.HookResults = append(tj.Status.HookResults, result)
				}
				
				tj.Status.Phase = "Succeeded"
				tj.Status.Message = "All tests passed"
				tj.Status.CompletionTime = time.Now()
				c.storage.UpdateHelmTestJob(tj)
			}(tj, hr)
		}
	}
}

func (c *Controller) executeHook(hook Hook, tj *HelmTestJob, hr *HelmRelease) HookResult {
	log.Printf("Executing hook: %s (type: %s)", hook.Name, hook.Type)
	
	// Resolve environment variables
	envVars := c.resolveEnvVars(hook.Env, tj, hr)
	log.Printf("Hook %s environment variables: %v", hook.Name, envVars)
	
	// Simulate hook execution
	time.Sleep(1 * time.Second)
	
	return HookResult{
		Name:    hook.Name,
		Phase:   "Succeeded",
		Message: fmt.Sprintf("Hook %s executed successfully", hook.Name),
	}
}

func (c *Controller) resolveEnvVars(envs []EnvVar, tj *HelmTestJob, hr *HelmRelease) map[string]string {
	result := make(map[string]string)
	
	for _, env := range envs {
		if env.Value != "" {
			result[env.Name] = env.Value
		} else if env.ValueFrom != nil {
			if env.ValueFrom.FieldPath != "" {
				// Simplified: extract from HelmTestJob
				switch env.ValueFrom.FieldPath {
				case "status.phase":
					result[env.Name] = tj.Status.Phase
				case "status.startTime":
					result[env.Name] = tj.Status.StartTime.Format(time.RFC3339)
				case "metadata.name":
					result[env.Name] = tj.Metadata.Name
				default:
					result[env.Name] = "unknown"
				}
			} else if env.ValueFrom.HelmReleaseRef != nil {
				// Extract from HelmRelease
				switch env.ValueFrom.HelmReleaseRef.FieldPath {
				case "metadata.name":
					result[env.Name] = hr.Metadata.Name
				case "metadata.namespace":
					result[env.Name] = hr.Metadata.Namespace
				case "spec.deployment.namespace":
					result[env.Name] = hr.Spec.Deployment.Namespace
				default:
					result[env.Name] = "unknown"
				}
			}
		}
	}
	
	return result
}

// ============================================================================
// Web API Handlers
// ============================================================================

type API struct {
	storage    *Storage
	controller *Controller
	upgrader   websocket.Upgrader
}

func NewAPI(storage *Storage, controller *Controller) *API {
	return &API{
		storage:    storage,
		controller: controller,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // Allow all origins in demo
			},
		},
	}
}

// CORS middleware
func (api *API) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// HelmRelease handlers
func (api *API) listHelmReleases(w http.ResponseWriter, r *http.Request) {
	releases := api.storage.ListHelmReleases()
	json.NewEncoder(w).Encode(releases)
}

func (api *API) createHelmRelease(w http.ResponseWriter, r *http.Request) {
	var hr HelmRelease
	if err := json.NewDecoder(r.Body).Decode(&hr); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	hr.APIVersion = "steer.io/v1alpha1"
	hr.Kind = "HelmRelease"
	
	if err := api.storage.CreateHelmRelease(&hr); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(hr)
}

func (api *API) getHelmRelease(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]
	
	hr, err := api.storage.GetHelmRelease(namespace, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	json.NewEncoder(w).Encode(hr)
}

func (api *API) deleteHelmRelease(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]
	
	if err := api.storage.DeleteHelmRelease(namespace, name); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// HelmTestJob handlers
func (api *API) listHelmTestJobs(w http.ResponseWriter, r *http.Request) {
	jobs := api.storage.ListHelmTestJobs()
	json.NewEncoder(w).Encode(jobs)
}

func (api *API) createHelmTestJob(w http.ResponseWriter, r *http.Request) {
	var tj HelmTestJob
	if err := json.NewDecoder(r.Body).Decode(&tj); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	
	tj.APIVersion = "steer.io/v1alpha1"
	tj.Kind = "HelmTestJob"
	
	if err := api.storage.CreateHelmTestJob(&tj); err != nil {
		http.Error(w, err.Error(), http.StatusConflict)
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(tj)
}

func (api *API) getHelmTestJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]
	
	tj, err := api.storage.GetHelmTestJob(namespace, name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	json.NewEncoder(w).Encode(tj)
}

func (api *API) deleteHelmTestJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	namespace := vars["namespace"]
	name := vars["name"]
	
	if err := api.storage.DeleteHelmTestJob(namespace, name); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	
	w.WriteHeader(http.StatusNoContent)
}

// ============================================================================
// Main
// ============================================================================

func main() {
	storage := NewStorage()
	controller := NewController(storage)
	api := NewAPI(storage, controller)
	
	// Start controller
	controller.Start()
	log.Println("Controller started")
	
	// Setup router
	router := mux.NewRouter()
	router.Use(api.corsMiddleware)
	
	// API routes
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiRouter.HandleFunc("/helmreleases", api.listHelmReleases).Methods("GET")
	apiRouter.HandleFunc("/helmreleases", api.createHelmRelease).Methods("POST")
	apiRouter.HandleFunc("/helmreleases/{namespace}/{name}", api.getHelmRelease).Methods("GET")
	apiRouter.HandleFunc("/helmreleases/{namespace}/{name}", api.deleteHelmRelease).Methods("DELETE")
	
	apiRouter.HandleFunc("/helmtestjobs", api.listHelmTestJobs).Methods("GET")
	apiRouter.HandleFunc("/helmtestjobs", api.createHelmTestJob).Methods("POST")
	apiRouter.HandleFunc("/helmtestjobs/{namespace}/{name}", api.getHelmTestJob).Methods("GET")
	apiRouter.HandleFunc("/helmtestjobs/{namespace}/{name}", api.deleteHelmTestJob).Methods("DELETE")
	
	// Serve static files
	router.PathPrefix("/").Handler(http.FileServer(http.Dir("../frontend/dist")))
	
	// Start server
	addr := ":8080"
	log.Printf("Server starting on http://localhost%s", addr)
	log.Printf("API available at http://localhost%s/api/v1", addr)
	
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
