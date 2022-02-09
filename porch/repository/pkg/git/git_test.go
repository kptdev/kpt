// Copyright 2022 Google LLC
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
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/GoogleContainerTools/kpt/porch/api/porch/v1alpha1"
	configapi "github.com/GoogleContainerTools/kpt/porch/controllers/pkg/apis/porch/v1alpha1"
	"github.com/GoogleContainerTools/kpt/porch/repository/pkg/repository"
	"github.com/go-git/go-git/v5"
	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/filemode"
	"github.com/go-git/go-git/v5/plumbing/format/packfile"
	"github.com/go-git/go-git/v5/plumbing/format/pktline"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage"
	"k8s.io/klog/v2"
)

func TestMain(m *testing.M) {
	klog.InitFlags(nil)
	flag.Parse()
	os.Exit(m.Run())
}

// TestGitPackageRoundTrip creates a package in git and verifies we can read the contents back.
func TestGitPackageRoundTrip(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	tempdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("TempDir failed: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tempdir); err != nil {
			t.Errorf("RemoveAll(%q) failed: %v", tempdir, err)
		}
	}()

	// Start a mock git server
	gitServerAddressChannel := make(chan net.Addr)

	p := filepath.Join(tempdir, "repo")
	serverRepo, err := gogit.PlainInit(p, true)
	if err != nil {
		t.Fatalf("failed to open source repo %q: %v", p, err)
	}

	if err := initRepo(serverRepo); err != nil {
		t.Fatalf("failed to init repo: %v", err)
	}

	gitServer, err := NewGitServer(serverRepo)
	if err != nil {
		t.Fatalf("NewGitServer() failed: %v", err)
	}

	go func() {
		if err := gitServer.ListenAndServe(ctx, "127.0.0.1:0", gitServerAddressChannel); err != nil {
			if ctx.Err() == nil {
				t.Errorf("ListenAndServe failed: %v", err)
			}
		}
	}()

	gitServerAddress, ok := <-gitServerAddressChannel
	if !ok {
		t.Fatalf("could not get address from server")
	}

	// Now that we are running a git server, we can create a GitRepository backed by it

	gitServerURL := "http://" + gitServerAddress.String()
	name := ""
	namespace := ""
	spec := &configapi.GitRepository{
		Repo: gitServerURL,
	}

	var credentialResolver repository.CredentialResolver
	root := filepath.Join(tempdir, "work")

	repo, err := OpenRepository(ctx, name, namespace, spec, credentialResolver, root)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}
	// TODO: is there any state? should we  defer repo.Close()

	t.Logf("repo is %#v", repo)

	// Push a package to the repo
	packageName := "test-package"
	revision := "v123"

	wantResources := map[string]string{
		"hello": "world",
	}

	{
		packageRevision := &v1alpha1.PackageRevision{}
		packageRevision.Spec.PackageName = packageName
		packageRevision.Spec.Revision = revision

		draft, err := repo.CreatePackageRevision(ctx, packageRevision)
		if err != nil {
			t.Fatalf("CreatePackageRevision(%#v) failed: %v", packageRevision, err)
		}

		newResources := &v1alpha1.PackageRevisionResources{}
		newResources.Spec.Resources = wantResources
		task := &v1alpha1.Task{}
		if err := draft.UpdateResources(ctx, newResources, task); err != nil {
			t.Fatalf("draft.UpdateResources(%#v, %#v) failed: %v", newResources, task, err)
		}

		revision, err := draft.Close(ctx)
		if err != nil {
			t.Fatalf("draft.Close() failed: %v", err)
		}
		klog.Infof("created revision %v", revision.Name())
	}

	// We approve the draft so that we can fetch it
	{
		approved, err := repo.(*gitRepository).ApprovePackageRevision(ctx, packageName, revision)
		if err != nil {
			t.Fatalf("ApprovePackageRevision(%q, %q) failed: %v", packageName, revision, err)
		}

		klog.Infof("approved revision %v", approved.Name())
	}

	// We reopen to refetch
	// TODO: This is pretty hacky...
	repo, err = OpenRepository(ctx, name, namespace, spec, credentialResolver, root)
	if err != nil {
		t.Fatalf("failed to open repository: %v", err)
	}
	// TODO: is there any state? should we  defer repo.Close()

	// Get the package again, the resources should match what we push
	{
		version := "v123"

		path := "test-package"
		packageRevision, gitLock, err := repo.GetPackage(version, path)
		if err != nil {
			t.Fatalf("GetPackage(%q, %q) failed: %v", version, path, err)
		}

		t.Logf("packageRevision is %s", packageRevision.Name())
		t.Logf("gitLock is %#v", gitLock)

		resources, err := packageRevision.GetResources(ctx)
		if err != nil {
			t.Fatalf("GetResources() failed: %v", err)
		}

		t.Logf("resources is %v", resources.Spec.Resources)

		if !reflect.DeepEqual(resources.Spec.Resources, wantResources) {
			t.Fatalf("resources did not match expected; got %v, want %v", resources.Spec.Resources, wantResources)
		}
	}
}

// GitServer is a mock git server implementing "just enough" of the git protocol
type GitServer struct {
	repo *gogit.Repository
}

// NewGitServer constructs a GitServer backed by the specified repo.
func NewGitServer(repo *gogit.Repository) (*GitServer, error) {
	return &GitServer{
		repo: repo,
	}, nil
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
	path := r.URL.Path
	if path == "/info/refs" {
		return s.serveGitInfoRefs(w, r)
	}
	if path == "/git-upload-pack" {
		return s.serveGitUploadPack(w, r)
	}
	if path == "/git-receive-pack" {
		return s.serveGitReceivePack(w, r)
	}

	klog.Warningf("404 for %s %s", r.Method, r.URL)
	http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
	return nil
}

// serveGitInfoRefs serves the info/refs (discovery) endpoint
func (s *GitServer) serveGitInfoRefs(w http.ResponseWriter, r *http.Request) error {
	query := r.URL.Query()
	serviceName := query.Get("service")

	capabilities := []string{}

	switch serviceName {
	case "git-upload-pack":
		// OK
		capabilities = append(capabilities, "symref=HEAD:refs/heads/main")

	case "git-receive-pack":
		// OK
		// TODO: capabilities?

	default:
		return fmt.Errorf("unknown service-name %q", serviceName)
	}

	// We send an advertisement for each of our references
	it, err := s.repo.References()
	if err != nil {
		return fmt.Errorf("failed to get git references: %w", err)
	}
	var refs []string
	if err := it.ForEach(func(ref *plumbing.Reference) error {
		name := ref.Name()
		if name.IsRemote() {
			klog.Infof("skipping remote ref %q", name)
			return nil
		}
		s := fmt.Sprintf("%s %s", ref.Hash().String(), name)
		refs = append(refs, s)
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

	for i, ref := range refs {
		s := ref
		if i == 0 {
			// We attach capabilities to the first line
			s += "\000" + strings.Join(capabilities, " ")
		}
		gw.WriteLine(s)
	}

	gw.WriteZeroPacketLine()

	if err := gw.Flush(); err != nil {
		klog.Warningf("error from flush: %v", err)
		// Too late to send a real error code
		return nil
	}

	return nil
}

// serveGitUploadPack serves the git-upload-pack endpoint
func (s *GitServer) serveGitUploadPack(w http.ResponseWriter, r *http.Request) error {
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
	walker := newObjectWalker(s.repo.Storer)
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
	storer := s.repo.Storer

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

func (s *GitServer) serveGitReceivePack(w http.ResponseWriter, r *http.Request) error {
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

	if err := packfile.UpdateObjectStorage(s.repo.Storer, body); err != nil {
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
		ref := plumbing.NewHashReference(plumbing.ReferenceName(refUpdate.Ref), refUpdate.To)
		if err := s.repo.Storer.SetReference(ref); err != nil {
			klog.Warningf("failed to update reference %v: %v", refUpdate, err)
		} else {
			klog.Warningf("updated reference %v -> %v", refUpdate.Ref, refUpdate.To)
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
		return fmt.Errorf("Getting object %s failed: %v", hash, err)
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
		return fmt.Errorf("Unknown object %s %s %T\n", obj.ID(), obj.Type(), obj)
	}
	return nil
}

// initRepo is a helper that creates a first commit, ensuring the repo is not empty.
func initRepo(repo *git.Repository) error {
	store := repo.Storer

	var objectHash plumbing.Hash
	{
		data := []byte("This is a test repo")
		eo := store.NewEncodedObject()
		eo.SetType(plumbing.BlobObject)
		eo.SetSize(int64(len(data)))

		w, err := eo.Writer()
		if err != nil {
			return fmt.Errorf("error creating object writer: %w", err)
		}

		if _, err = w.Write(data); err != nil {
			w.Close()
			return fmt.Errorf("error writing object data: %w", err)
		}
		if err := w.Close(); err != nil {
			return fmt.Errorf("error closing object data: %w", err)
		}

		if h, err := store.SetEncodedObject(eo); err != nil {
			return fmt.Errorf("error storing object: %w", err)
		} else {
			objectHash = h
		}
	}

	var treeHash plumbing.Hash
	{
		tree := object.Tree{}

		te := object.TreeEntry{
			Name: "README.md",
			Mode: filemode.Regular,
			Hash: objectHash,
		}
		tree.Entries = append(tree.Entries, te)

		eo := store.NewEncodedObject()
		if err := tree.Encode(eo); err != nil {
			return fmt.Errorf("error encoding tree: %w", err)
		}
		if h, err := store.SetEncodedObject(eo); err != nil {
			return fmt.Errorf("error storing tree: %w", err)
		} else {
			treeHash = h
		}
	}

	var commitHash plumbing.Hash
	{
		now := time.Now()
		commit := &object.Commit{
			Author: object.Signature{
				Name:  "Porch Author",
				Email: "author@kpt.dev",
				When:  now,
			},
			Committer: object.Signature{
				Name:  "Porch Committer",
				Email: "committer@kpt.dev",
				When:  now,
			},
			Message:  "First commit",
			TreeHash: treeHash,
		}

		eo := store.NewEncodedObject()
		if err := commit.Encode(eo); err != nil {
			return fmt.Errorf("error encoding commit: %w", err)
		}
		if h, err := store.SetEncodedObject(eo); err != nil {
			return fmt.Errorf("error storing commit: %w", err)
		} else {
			commitHash = h
		}
	}

	{
		ref := plumbing.NewHashReference("refs/heads/main", commitHash)
		if err := repo.Storer.SetReference(ref); err != nil {
			return fmt.Errorf("error setting reference: %w", err)
		}
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
