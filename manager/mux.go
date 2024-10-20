package main

import (
	"encoding/json"
	"fmt"
	"github.com/darkit/plugins/manager/dist"
	"net/http"

	pluginmanager "github.com/darkit/plugins"
	"github.com/gorilla/mux"
)

type MuxServer struct {
	manager *pluginmanager.Manager
}

func NewMuxServer(manager *pluginmanager.Manager) *MuxServer {
	return &MuxServer{manager: manager}
}

func (s *MuxServer) Start() error {
	r := mux.NewRouter()

	r.HandleFunc("/plugins", s.listPlugins).Methods("GET")
	r.HandleFunc("/plugins/{name}", s.getPlugin).Methods("GET")
	r.HandleFunc("/plugins/{name}/load", s.loadPlugin).Methods("POST")
	r.HandleFunc("/plugins/{name}/unload", s.unloadPlugin).Methods("POST")
	r.HandleFunc("/plugins/{name}/execute", s.executePlugin).Methods("POST")
	r.HandleFunc("/plugins/{name}/update", s.updatePlugin).Methods("POST")
	r.HandleFunc("/plugins/{name}/rollback", s.rollbackPlugin).Methods("POST")
	r.HandleFunc("/market", s.listMarketPlugins).Methods("GET")
	r.HandleFunc("/market/{name}/install", s.installPlugin).Methods("POST")

	// 静态文件服务
	r.Handle("/layui/", http.StripPrefix("/", http.FileServer(http.FS(dist.WebDist))))

	http.Handle("/", r)
	return http.ListenAndServe(":8080", nil)
}

func (s *MuxServer) listPlugins(w http.ResponseWriter, r *http.Request) {
	plugins := s.manager.ListPlugins()
	json.NewEncoder(w).Encode(plugins)
}

func (s *MuxServer) getPlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	plugin, err := s.manager.GetPluginConfig(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(plugin)
}

func (s *MuxServer) loadPlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	err := s.manager.LoadPlugin(fmt.Sprintf("./plugins/%s.so", name))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *MuxServer) unloadPlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	err := s.manager.UnloadPlugin(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *MuxServer) executePlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	var data interface{}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	result, err := s.manager.ExecutePlugin(name, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(result)
}

func (s *MuxServer) updatePlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	var version string
	err := json.NewDecoder(r.Body).Decode(&version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = s.manager.HotUpdatePlugin(name, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *MuxServer) rollbackPlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	var version string
	err := json.NewDecoder(r.Body).Decode(&version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = s.manager.RollbackPlugin(name, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *MuxServer) listMarketPlugins(w http.ResponseWriter, r *http.Request) {
	plugins := s.manager.ListAvailablePlugins()
	json.NewEncoder(w).Encode(plugins)
}

func (s *MuxServer) installPlugin(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	var version string
	err := json.NewDecoder(r.Body).Decode(&version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = s.manager.DownloadAndInstallPlugin(name, version)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}
