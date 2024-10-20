package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	pluginmanager "github.com/darkit/plugins"
	"github.com/darkit/plugins/manager/dist"
)

type Server struct {
	manager *pluginmanager.Manager
}

func NewServer(manager *pluginmanager.Manager) *Server {
	return &Server{manager: manager}
}

func (s *Server) Start() error {
	http.HandleFunc("/", s.serveHTML)
	http.HandleFunc("/plugins", s.handlePlugins)
	http.HandleFunc("/plugins/", s.handlePluginOperations)
	http.HandleFunc("/market", s.handleMarket)
	http.HandleFunc("/market/", s.handleMarketOperations)

	// 静态文件服务
	http.Handle("/layui/", http.StripPrefix("/", http.FileServer(http.FS(dist.WebDist))))

	return http.ListenAndServe(":715", nil)
}

func (s *Server) serveHTML(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	content, err := dist.WebDist.ReadFile("http.html")
	if err != nil {
		http.Error(w, "Could not read index.html", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write(content)
}

func (s *Server) handlePlugins(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	plugins := s.manager.ListPlugins()
	json.NewEncoder(w).Encode(plugins)
}

func (s *Server) handlePluginOperations(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Path[len("/plugins/"):]
	switch r.Method {
	case http.MethodGet:
		s.getPlugin(w, r, name)
	case http.MethodPost:
		if r.URL.Path[len("/plugins/"):] == name+"/load" {
			s.loadPlugin(w, r, name)
		} else if r.URL.Path[len("/plugins/"):] == name+"/unload" {
			s.unloadPlugin(w, r, name)
		} else if r.URL.Path[len("/plugins/"):] == name+"/execute" {
			s.executePlugin(w, r, name)
		} else if r.URL.Path[len("/plugins/"):] == name+"/update" {
			s.updatePlugin(w, r, name)
		} else if r.URL.Path[len("/plugins/"):] == name+"/rollback" {
			s.rollbackPlugin(w, r, name)
		} else {
			http.Error(w, "Not found", http.StatusNotFound)
		}
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) getPlugin(w http.ResponseWriter, r *http.Request, name string) {
	plugin, err := s.manager.GetPluginConfig(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(plugin)
}

func (s *Server) loadPlugin(w http.ResponseWriter, r *http.Request, name string) {
	err := s.manager.LoadPlugin(fmt.Sprintf("./plugins/%s.so", name))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) unloadPlugin(w http.ResponseWriter, r *http.Request, name string) {
	err := s.manager.UnloadPlugin(name)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) executePlugin(w http.ResponseWriter, r *http.Request, name string) {
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

func (s *Server) updatePlugin(w http.ResponseWriter, r *http.Request, name string) {
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

func (s *Server) rollbackPlugin(w http.ResponseWriter, r *http.Request, name string) {
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

func (s *Server) handleMarket(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	plugins := s.manager.ListAvailablePlugins()
	json.NewEncoder(w).Encode(plugins)
}

func (s *Server) handleMarketOperations(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := r.URL.Path[len("/market/"):]
	if r.URL.Path[len("/market/"):] == name+"/install" {
		s.installPlugin(w, r, name)
	} else {
		http.Error(w, "Not found", http.StatusNotFound)
	}
}

func (s *Server) installPlugin(w http.ResponseWriter, r *http.Request, name string) {
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

func main() {
	manager, err := pluginmanager.NewManager("./plugins", "config.msgpack")
	if err != nil {
		panic(err)
	}

	server := NewServer(manager)
	err = server.Start()
	if err != nil {
		panic(err)
	}
}
