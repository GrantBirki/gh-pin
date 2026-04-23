package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
)

const (
	indexDigest  = "sha256:944ee135f516fd1bc20a4e840486e034f988f3e60ec03ba047a00da9c8881d1c"
	amd64Digest  = "sha256:2222222222222222222222222222222222222222222222222222222222222222"
	arm64Digest  = "sha256:3333333333333333333333333333333333333333333333333333333333333333"
	indexType    = "application/vnd.oci.image.index.v1+json"
	manifestType = "application/vnd.oci.image.manifest.v1+json"
)

type indexManifest struct {
	SchemaVersion int          `json:"schemaVersion"`
	MediaType     string       `json:"mediaType"`
	Manifests     []descriptor `json:"manifests"`
}

type descriptor struct {
	MediaType string   `json:"mediaType"`
	Digest    string   `json:"digest"`
	Size      int      `json:"size"`
	Platform  platform `json:"platform"`
}

type platform struct {
	OS           string `json:"os"`
	Architecture string `json:"architecture"`
}

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("listen: %v", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/", handleV2)

	fmt.Printf("REGISTRY_ADDR=%s\n", listener.Addr().String())
	log.Fatal(http.Serve(listener, mux))
}

func handleV2(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Docker-Distribution-API-Version", "registry/2.0")

	if r.URL.Path == "/v2/" {
		w.WriteHeader(http.StatusOK)
		return
	}

	repository, _, ok := parseManifestPath(r.URL.Path)
	if !ok {
		http.NotFound(w, r)
		return
	}
	log.Printf("%s %s", r.Method, repository)

	switch r.Method {
	case http.MethodGet:
		writeIndex(w)
	case http.MethodHead:
		writeIndexHeaders(w, len(indexBody()))
	case http.MethodOptions:
		w.Header().Set("Allow", "GET, HEAD, OPTIONS")
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, HEAD, OPTIONS")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func parseManifestPath(path string) (string, string, bool) {
	rest, ok := strings.CutPrefix(path, "/v2/")
	if !ok {
		return "", "", false
	}

	repository, reference, ok := strings.Cut(rest, "/manifests/")
	if !ok || repository == "" || reference == "" {
		return "", "", false
	}

	return repository, reference, true
}

func writeIndex(w http.ResponseWriter) {
	body := indexBody()
	writeIndexHeaders(w, len(body))
	if _, err := w.Write(body); err != nil {
		log.Printf("write response: %v", err)
	}
}

func writeIndexHeaders(w http.ResponseWriter, size int) {
	w.Header().Set("Content-Type", indexType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.Header().Set("Docker-Content-Digest", indexDigest)
	w.WriteHeader(http.StatusOK)
}

func indexBody() []byte {
	body, err := json.Marshal(indexManifest{
		SchemaVersion: 2,
		MediaType:     indexType,
		Manifests: []descriptor{
			{
				MediaType: manifestType,
				Digest:    amd64Digest,
				Size:      123,
				Platform: platform{
					OS:           "linux",
					Architecture: "amd64",
				},
			},
			{
				MediaType: manifestType,
				Digest:    arm64Digest,
				Size:      123,
				Platform: platform{
					OS:           "linux",
					Architecture: "arm64",
				},
			},
		},
	})
	if err != nil {
		panic(err)
	}

	return body
}
