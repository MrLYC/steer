package web

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	steerv1alpha1 "github.com/MrLYC/steer/operator/api/v1alpha1"
)

type Server struct {
	addr      string
	staticDir string
	k8sClient client.Client
}

func NewServer(addr string, staticDir string, k8sClient client.Client) *Server {
	return &Server{addr: addr, staticDir: staticDir, k8sClient: k8sClient}
}

func (s *Server) Start(ctx context.Context) error {
	if s.addr == "" {
		return errors.New("web addr is empty")
	}
	if s.k8sClient == nil {
		return errors.New("k8s client is nil")
	}

	router := mux.NewRouter()
	router.Use(corsMiddleware)

	api := router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/helmreleases", s.handleListHelmReleases).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/helmreleases", s.handleCreateHelmRelease).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/helmreleases/{namespace}/{name}", s.handleGetHelmRelease).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/helmreleases/{namespace}/{name}", s.handleDeleteHelmRelease).Methods(http.MethodDelete, http.MethodOptions)

	api.HandleFunc("/helmtestjobs", s.handleListHelmTestJobs).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/helmtestjobs", s.handleCreateHelmTestJob).Methods(http.MethodPost, http.MethodOptions)
	api.HandleFunc("/helmtestjobs/{namespace}/{name}", s.handleGetHelmTestJob).Methods(http.MethodGet, http.MethodOptions)
	api.HandleFunc("/helmtestjobs/{namespace}/{name}", s.handleDeleteHelmTestJob).Methods(http.MethodDelete, http.MethodOptions)

	// Static UI: keep it as a fallback, so API routes win.
	if s.staticDir != "" {
		router.PathPrefix("/").Handler(spaFileServer(s.staticDir))
	}

	httpServer := &http.Server{
		Addr:              s.addr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return fmt.Errorf("web server failed: %w", err)
	}
}

func spaFileServer(staticDir string) http.Handler {
	fs := http.FileServer(http.Dir(staticDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Vite assets / direct files: serve if exists.
		cleanPath := filepath.Clean(r.URL.Path)
		if strings.HasPrefix(cleanPath, "..") {
			writeError(w, http.StatusBadRequest, "invalid path")
			return
		}
		candidate := filepath.Join(staticDir, cleanPath)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}

		// SPA fallback: serve index.html for non-API routes.
		if strings.HasPrefix(r.URL.Path, "/api/") {
			writeError(w, http.StatusNotFound, "not found")
			return
		}
		indexPath := filepath.Join(staticDir, "index.html")
		http.ServeFile(w, r, indexPath)
	})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,DELETE,OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleListHelmReleases(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var list steerv1alpha1.HelmReleaseList
	if err := s.k8sClient.List(ctx, &list); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list.Items)
}

func (s *Server) handleCreateHelmRelease(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var obj steerv1alpha1.HelmRelease
	if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if obj.APIVersion == "" {
		obj.APIVersion = steerv1alpha1.GroupVersion.String()
	}
	if obj.Kind == "" {
		obj.Kind = "HelmRelease"
	}
	if obj.Name == "" || obj.Namespace == "" {
		writeError(w, http.StatusBadRequest, "metadata.name and metadata.namespace are required")
		return
	}
	if err := s.k8sClient.Create(ctx, &obj); err != nil {
		if apierrors.IsAlreadyExists(err) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, obj)
}

func (s *Server) handleGetHelmRelease(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	nn := types.NamespacedName{Namespace: vars["namespace"], Name: vars["name"]}
	var obj steerv1alpha1.HelmRelease
	if err := s.k8sClient.Get(ctx, nn, &obj); err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, obj)
}

func (s *Server) handleDeleteHelmRelease(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	nn := types.NamespacedName{Namespace: vars["namespace"], Name: vars["name"]}
	obj := &steerv1alpha1.HelmRelease{}
	obj.Namespace = nn.Namespace
	obj.Name = nn.Name
	if err := s.k8sClient.Delete(ctx, obj); err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleListHelmTestJobs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var list steerv1alpha1.HelmTestJobList
	if err := s.k8sClient.List(ctx, &list); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, list.Items)
}

func (s *Server) handleCreateHelmTestJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var obj steerv1alpha1.HelmTestJob
	if err := json.NewDecoder(r.Body).Decode(&obj); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if obj.APIVersion == "" {
		obj.APIVersion = steerv1alpha1.GroupVersion.String()
	}
	if obj.Kind == "" {
		obj.Kind = "HelmTestJob"
	}
	if obj.Name == "" || obj.Namespace == "" {
		writeError(w, http.StatusBadRequest, "metadata.name and metadata.namespace are required")
		return
	}
	if err := s.k8sClient.Create(ctx, &obj); err != nil {
		if apierrors.IsAlreadyExists(err) {
			writeError(w, http.StatusConflict, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, obj)
}

func (s *Server) handleGetHelmTestJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	nn := types.NamespacedName{Namespace: vars["namespace"], Name: vars["name"]}
	var obj steerv1alpha1.HelmTestJob
	if err := s.k8sClient.Get(ctx, nn, &obj); err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, obj)
}

func (s *Server) handleDeleteHelmTestJob(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	vars := mux.Vars(r)
	nn := types.NamespacedName{Namespace: vars["namespace"], Name: vars["name"]}
	obj := &steerv1alpha1.HelmTestJob{}
	obj.Namespace = nn.Namespace
	obj.Name = nn.Name
	if err := s.k8sClient.Delete(ctx, obj); err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}
