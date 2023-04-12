// Copyright 2022 The kpt Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package git

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/protocol/packp/capability"
	"github.com/go-git/go-git/v5/storage"
	"k8s.io/klog/v2"
)

// GitServer is a mock git server implementing "just enough" of the git protocol
type GitServer struct {
	repos Repos
}

// NewGitServer constructs a GitServer backed by the specified repo.
func NewGitServer(repos Repos, opts ...GitServerOption) (*GitServer, error) {
	gs := &GitServer{
		repos: repos,
	}

	for _, opt := range opts {
		if err := opt.apply(gs); err != nil {
			return nil, err
		}
	}

	return gs, nil
}

// ListenAndServe starts the git server on "listen".
// The address we actually start listening on will be posted to addressChannel
func (s *GitServer) ListenAndServe(ctx context.Context, listen string, addressChannel chan<- net.Addr) error {
	httpServer := &http.Server{
		Addr:           listen,
		Handler:        s,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   60 * time.Second, // We need more time to build the pack file
		MaxHeaderBytes: 1 << 20,
	}

	ln, err := net.Listen("tcp", httpServer.Addr)
	if err != nil {
		close(addressChannel)
		return err
	}

	ctxWithCancel, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		<-ctxWithCancel.Done()
		if err := httpServer.Shutdown(context.Background()); err != nil {
			klog.Warningf("error from git httpServer.Shutdown: %v", err)
		}
		if err := httpServer.Close(); err != nil {
			klog.Warningf("error from git httpServer.Close: %v", err)
		}
	}()

	addressChannel <- ln.Addr()

	return httpServer.Serve(ln)
}

// ServeHTTP is the entrypoint for http requests.
func (s *GitServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.serveRequest(w, r); err != nil {
		klog.Warningf("internal error from %s %s: %v", r.Method, r.URL, err)

		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// serveRequest is the main dispatcher for http requests.
func (s *GitServer) serveRequest(w http.ResponseWriter, r *http.Request) error {
	pathTokens := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(pathTokens) > 1 {
		repoID := pathTokens[0]
		repo, err := s.repos.FindRepo(r.Context(), repoID)
		if err != nil {
			klog.Warningf("500 for %s %s: %v", r.Method, r.URL, err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return nil

		}
		if repo == nil {
			// TODO: Should we send something consistent with auth failure?
			klog.Warningf("404 for %s %s (repo not found)", r.Method, r.URL)
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return nil
		}

		if repo.username != "" || repo.password != "" {
			username, password, ok := r.BasicAuth()
			if !ok || username != repo.username || password != repo.password {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return nil
			}
		}
		gitPath := strings.Join(pathTokens[1:], "/")
		if gitPath == "info/refs" {
			return s.serveGitInfoRefs(w, r, repo)
		}
		if gitPath == "git-upload-pack" {
			return s.serveGitUploadPack(w, r, repo)
		}
		if gitPath == "git-receive-pack" {
			return s.serveGitReceivePack(w, r, repo)
		}
	}

	klog.Warningf("404 for %s %s", r.Method, r.URL)
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	return nil
}

// serveGitInfoRefs serves the info/refs (discovery) endpoint
func (s *GitServer) serveGitInfoRefs(w http.ResponseWriter, r *http.Request, repo *Repo) error {
	query := r.URL.Query()
	serviceName := query.Get("service")

	capabilities := []string{string(capability.DeleteRefs)}

	switch serviceName {
	case "git-upload-pack":
		// OK
		capabilities = append(capabilities, "symref=HEAD:refs/heads/main")

	case "git-receive-pack":
		// OK

	default:
		return fmt.Errorf("unknown service-name %q", serviceName)
	}

	// We send an advertisement for each of our references
	it, err := repo.gogit.References()
	if err != nil {
		return fmt.Errorf("failed to get git references: %w", err)
	}

	// Find HEAD so we can return it first (https://git-scm.com/docs/http-protocol)
	var head *plumbing.Reference
	refs := map[string]*plumbing.Reference{} // Resolving symbolic refs will lead to dupes. De-dupe them.
	if err := it.ForEach(func(ref *plumbing.Reference) error {
		var resolved *plumbing.Reference
		// Resolve symbolic references.
		switch ref.Type() {
		case plumbing.SymbolicReference:
			if r, err := repo.gogit.Reference(ref.Name(), true); err != nil {
				klog.Warningf("Skipping unresolvable symbolic reference %q: %v", ref.Name(), err)
				return nil
			} else {
				resolved = r
			}
		case plumbing.HashReference:
			resolved = ref
		default:
			return fmt.Errorf("unexpected reference encountered: %s", ref)
		}

		resolvedName := resolved.Name()
		if resolvedName.IsRemote() {
			klog.Infof("skipping remote ref %q", resolvedName)
			return nil
		}

		refs[resolvedName.String()] = resolved
		if ref.Name() == plumbing.HEAD {
			head = resolved
		}
		return nil
	}); err != nil {
		return fmt.Errorf("error iterating through references: %w", err)
	}

	w.Header().Set("Content-Type", "application/x-"+serviceName+"-advertisement")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	gw := NewPacketLineWriter(w)

	gw.WriteLine("# service=" + serviceName)
	gw.WriteZeroPacketLine()

	sorted := sortedRefs(refs, head)
	writeRefs(gw, sorted, capabilities)

	gw.WriteZeroPacketLine()

	if err := gw.Flush(); err != nil {
		klog.Warningf("error from flush: %v", err)
		// Too late to send a real error code
		return nil
	}

	return nil
}

func sortedRefs(refs map[string]*plumbing.Reference, head *plumbing.Reference) []*plumbing.Reference {
	sorted := make([]*plumbing.Reference, 0, len(refs))
	for _, v := range refs {
		sorted = append(sorted, v)
	}
	sort.Slice(sorted, func(i, j int) bool {
		switch {
		case sorted[i] == head:
			return true
		case sorted[j] == head:
			return false
		default:
			return sorted[i].Name().String() < sorted[j].Name().String()
		}
	})
	return sorted
}

func writeRefs(gw *PacketLineWriter, sorted []*plumbing.Reference, capabilities []string) {
	// empty_list = PKT-LINE(zero-id SP "capabilities^{}" NUL cap-list LF)
	if len(sorted) == 0 {
		var zero plumbing.Hash
		s := fmt.Sprintf("%s capabilities^{}\000%s", zero, strings.Join(capabilities, " "))
		gw.WriteLine(s)
		return
	}

	// non_empty_list  =  PKT-LINE(obj-id SP name NUL cap_list LF)
	//   *ref_record
	// ref_record      =  any_ref / peeled_ref
	// any_ref         =  PKT-LINE(obj-id SP name LF)
	for i, ref := range sorted {
		s := fmt.Sprintf("%s %s", ref.Hash(), ref.Name())
		if i == 0 {
			// We attach capabilities to the first line
			s += "\000" + strings.Join(capabilities, " ")
		}
		gw.WriteLine(s)
	}
}

// serveGitUploadPack serves the git-upload-pack endpoint
func (s *GitServer) serveGitUploadPack(w http.ResponseWriter, r *http.Request, repo *Repo) error {
	// See https://git-scm.com/docs/pack-protocol/2.2.3#_packfile_negotiation

	// The client sends a line for each sha it wants and each sha it has
	scanner := pktline.NewScanner(r.Body)
	for {
		if !scanner.Scan() {
			err := scanner.Err()
			if err != nil {
				return fmt.Errorf("error parsing request: %w", err)
			}
			break
		}
		line := scanner.Bytes()
		klog.V(4).Infof("request line: %s", string(line))
	}

	// We implement a very dumb version of the protocol; we always send everything
	// This works, and is correct on the "clean pull" scenario, but is not efficient in the real world.

	// Gather all the objects
	walker := newObjectWalker(repo.gogit.Storer)
	if err := walker.walkAllRefs(); err != nil {
		return fmt.Errorf("error walking refs: %w", err)
	}

	objects := make([]plumbing.Hash, 0, len(walker.seen))
	for h := range walker.seen {
		objects = append(objects, h)
	}

	// Send a NAK indicating we're sending everything
	encoder := NewPacketLineWriter(w)
	encoder.WriteLine("NAK")
	if err := encoder.Flush(); err != nil {
		klog.Warningf("error encoding response: %v", err)
		return nil // Too late
	}

	// Send the packfile data
	klog.Infof("sending %d objects in packfile", len(objects))

	useRefDeltas := false
	storer := repo.gogit.Storer

	// TODO: Buffer on disk first?
	packFileEncoder := packfile.NewEncoder(w, storer, useRefDeltas)

	// packWindow specifies the size of the sliding window used
	// to compare objects for delta compression;
	// 0 turns off delta compression entirely.
	packWindow := uint(0)

	packfileHash, err := packFileEncoder.Encode(objects, packWindow)
	if err != nil {
		klog.Warningf("error encoding packfile: %v", err)
		return nil // Too late
	}

	klog.Infof("packed as %v", packfileHash)

	return nil
}

type GitHash = plumbing.Hash

// RefUpdate stores requested tag/branch updates
type RefUpdate struct {
	From GitHash
	To   GitHash
	Ref  string
}

func (s *GitServer) serveGitReceivePack(w http.ResponseWriter, r *http.Request, repo *Repo) error {
	var refUpdates []RefUpdate

	body := r.Body

	contentEncoding := r.Header.Get("Content-Encoding")
	switch contentEncoding {
	case "":
		// OK

	case "gzip":
		gzr, err := gzip.NewReader(body)
		if err != nil {
			return fmt.Errorf("gzip.NewReader failed: %w", err)
		}
		defer gzr.Close()
		body = gzr

	default:
		return fmt.Errorf("unknown content-encoding %q", contentEncoding)
	}

	// The client sends a line for each ref it wants to update, then it sends the packfile data
	gr := pktline.NewScanner(body)

	var clientCapabilites []string

	firstLine := true
	for {
		if !gr.Scan() {
			err := gr.Err()
			if err != nil {
				return fmt.Errorf("error reading request line: %w", err)
			}
			return fmt.Errorf("error reading request line: EOF")
		}

		line := string(gr.Bytes())

		klog.V(4).Infof("client sent %q", line)
		if line == "" {
			break
		}

		tokens := strings.SplitN(line, " ", 3)
		if len(tokens) != 3 {
			return fmt.Errorf("unexpected line (spaces) %q", line)
		}
		refTokens := strings.Split(tokens[2], "\000")
		ref := refTokens[0]
		if !firstLine {
			if len(refTokens) != 1 {
				return fmt.Errorf("unexpected line (nulls) %q", line)
			}
		} else {
			if len(refTokens) > 1 {
				clientCapabilites = refTokens[1:]
			}
			firstLine = false
		}

		from, err := parseHash(tokens[0])
		if err != nil {
			return fmt.Errorf("unexpected line (hash1) %q", line)
		}

		to, err := parseHash(tokens[1])
		if err != nil {
			return fmt.Errorf("unexpected line (hash2) %q", line)
		}

		refUpdates = append(refUpdates, RefUpdate{From: from, To: to, Ref: ref})
	}

	klog.V(2).Infof("clientCapabilites %v", clientCapabilites)
	klog.V(2).Infof("updates %+v", refUpdates)

	// TODO: In a real implementation, we would check the shas here

	w.Header().Set("Content-Type", "application/x-git-upload-pack-result")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)

	gitWriter := NewPacketLineWriter(w)

	switch err := packfile.UpdateObjectStorage(repo.gogit.Storer, body); err {
	case nil, packfile.ErrEmptyPackfile:
		// ok
	default:
		klog.Warningf("error parsing packfile: %v", err)
		gitWriter.WriteLine("unpack error parsing packfile")
		gitWriter.Flush()
		return nil
	}

	// TODO: In a real implementation, we would validate the packfile data

	gitWriter.WriteLine("unpack ok")
	gitWriter.WriteZeroPacketLine()
	if err := gitWriter.Flush(); err != nil {
		klog.Warningf("error flushing response: %w", err)
		return nil // too late for real errors
	}

	// Having accepted the packfile into our store, we should update the SHAs

	// TODO: Concurrency, if we ever pull this out of test code
	for _, refUpdate := range refUpdates {
		switch {
		case refUpdate.To.IsZero():
			klog.Infof("Deleting reference %s", refUpdate.Ref)
			repo.gogit.Storer.RemoveReference(plumbing.ReferenceName(refUpdate.Ref))

		default:
			ref := plumbing.NewHashReference(plumbing.ReferenceName(refUpdate.Ref), refUpdate.To)
			if err := repo.gogit.Storer.SetReference(ref); err != nil {
				klog.Warningf("failed to update reference %v: %v", refUpdate, err)
			} else {
				klog.Warningf("updated reference %v -> %v", refUpdate.Ref, refUpdate.To)
			}
		}
	}

	return nil
}

// objectWalker is based on objectWalker in go-git/v5

type objectWalker struct {
	Storer storage.Storer
	// seen is the set of objects seen in the repo.
	// seen map can become huge if walking over large
	// repos. Thus using struct{} as the value type.
	seen map[plumbing.Hash]struct{}
}

func newObjectWalker(s storage.Storer) *objectWalker {
	return &objectWalker{s, map[plumbing.Hash]struct{}{}}
}

// walkAllRefs walks all (hash) references from the repo.
func (p *objectWalker) walkAllRefs() error {
	// Walk over all the references in the repo.
	it, err := p.Storer.IterReferences()
	if err != nil {
		return err
	}
	defer it.Close()
	err = it.ForEach(func(ref *plumbing.Reference) error {
		// Exit this iteration early for non-hash references.
		if ref.Type() != plumbing.HashReference {
			return nil
		}
		return p.walkObjectTree(ref.Hash())
	})
	return err
}

func (p *objectWalker) isSeen(hash plumbing.Hash) bool {
	_, seen := p.seen[hash]
	return seen
}

func (p *objectWalker) add(hash plumbing.Hash) {
	p.seen[hash] = struct{}{}
}

// walkObjectTree walks over all objects and remembers references
// to them in the objectWalker. This is used instead of the revlist
// walks because memory usage is tight with huge repos.
func (p *objectWalker) walkObjectTree(hash plumbing.Hash) error {
	// Check if we have already seen, and mark this object
	if p.isSeen(hash) {
		return nil
	}
	p.add(hash)
	// Fetch the object.
	obj, err := object.GetObject(p.Storer, hash)
	if err != nil {
		return fmt.Errorf("getting object %s failed: %v", hash, err)
	}
	// Walk all children depending on object type.
	switch obj := obj.(type) {
	case *object.Commit:
		err = p.walkObjectTree(obj.TreeHash)
		if err != nil {
			return err
		}
		for _, h := range obj.ParentHashes {
			err = p.walkObjectTree(h)
			if err != nil {
				return err
			}
		}
	case *object.Tree:
	nextEntry:
		for i := range obj.Entries {
			switch obj.Entries[i].Mode {
			case filemode.Executable, filemode.Regular, filemode.Symlink:
				p.add(obj.Entries[i].Hash)
				continue nextEntry
			case filemode.Submodule:
				// hash is the submodule ref, I believe
				continue nextEntry
			case filemode.Dir:
				// process recursively
			default:
				klog.Warningf("unknown entry mode %s", obj.Entries[i].Mode)
			}
			// Normal walk for sub-trees (and symlinks etc).
			err = p.walkObjectTree(obj.Entries[i].Hash)
			if err != nil {
				return err
			}
		}
	case *object.Tag:
		return p.walkObjectTree(obj.Target)
	default:
		// Error out on unhandled object types.
		return fmt.Errorf("unknown object %s %s %T", obj.ID(), obj.Type(), obj)
	}
	return nil
}

// parseHash is a helper that parses a GitHash provided by the client.
func parseHash(s string) (GitHash, error) {
	var h GitHash
	b, err := hex.DecodeString(s)
	if err != nil {
		return h, fmt.Errorf("hash %q was not hex", s)
	}
	if len(b) != 20 {
		return h, fmt.Errorf("hash %q was wrong length", s)
	}
	copy(h[:], b)
	return h, nil
}

// NewPackageLineWriter constructs a PacketLineWriter
func NewPacketLineWriter(w io.Writer) *PacketLineWriter {
	bw := bufio.NewWriter(w)
	return &PacketLineWriter{
		w: bw,
	}
}

// PacketLineWriter implements the git protocol line framing, with deferred error handling.
type PacketLineWriter struct {
	err error
	w   *bufio.Writer
}

// Flush writes any buffered data, and returns an error if one has accumulated.
func (w *PacketLineWriter) Flush() error {
	if w.err != nil {
		return w.err
	}
	return w.w.Flush()
}

// WriteLine frames and writes a line, accumulating errors until Flush is called.
func (w *PacketLineWriter) WriteLine(s string) {
	if w.err != nil {
		return
	}

	n := 4 + len(s) + 1
	prefix := fmt.Sprintf("%04x", n)

	if _, err := w.w.Write([]byte(prefix)); err != nil {
		w.err = err
		return
	}
	if _, err := w.w.Write([]byte(s)); err != nil {
		w.err = err
		return
	}
	if _, err := w.w.Write([]byte("\n")); err != nil {
		w.err = err
		return
	}

	klog.V(4).Infof("writing pktline %q", s)
}

// WriteZeroPacketLine writes a special "0000" line - often used to indicate the end of a block in the git protocol
func (w *PacketLineWriter) WriteZeroPacketLine() {
	if w.err != nil {
		return
	}

	if _, err := w.w.Write([]byte("0000")); err != nil {
		w.err = err
		return
	}

	klog.V(4).Infof("writing pktline 0000")
}

// Options

type GitServerOption interface {
	apply(*GitServer) error
}
