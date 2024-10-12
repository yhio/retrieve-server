package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/ipfs/go-cid"
	"github.com/yhio/retrieve-server/middleware"
)

type RootBlock struct {
	Root  string `json:"root"`
	Block []byte `json:"block"`
}

type RootSize struct {
	Root string `json:"root"`
	Size int    `json:"size"`
}

func (s *Server) Handle() {
	http.HandleFunc("POST /block", middleware.Timer(s.upsertHandle, "upsert"))
	http.HandleFunc("GET /block/{root}", middleware.Timer(s.blockHandle, "block"))
	http.HandleFunc("GET /size/{root}", middleware.Timer(s.sizeHandle, "size"))
	http.HandleFunc("DELETE /block/{root}", middleware.Timer(s.deleteHandle, "delete"))
}

func (s *Server) upsertHandle(w http.ResponseWriter, r *http.Request) {
	var rb RootBlock
	err := json.NewDecoder(r.Body).Decode(&rb)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = verify(&rb)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = s.upsert(&rb)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) blockHandle(w http.ResponseWriter, r *http.Request) {
	root := r.PathValue("root")
	block, err := s.block(root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rb := RootBlock{
		Root:  root,
		Block: block,
	}

	err = json.NewEncoder(w).Encode(rb)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) sizeHandle(w http.ResponseWriter, r *http.Request) {
	root := r.PathValue("root")
	size, err := s.size(root)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	rz := RootSize{
		Root: root,
		Size: size,
	}

	err = json.NewEncoder(w).Encode(rz)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

}

func (s *Server) deleteHandle(w http.ResponseWriter, r *http.Request) {
	err := s.delete(r.PathValue("root"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func verify(rb *RootBlock) error {
	root, err := cid.Parse(rb.Root)
	if err != nil {
		return err
	}

	new, err := root.Prefix().Sum(rb.Block)
	if err != nil {
		return err
	}

	if !new.Equals(root) {
		return fmt.Errorf("cid not match, %s!=%s", root, new)
	}

	return nil
}
