// Copyright 2019 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The sandbox program is an HTTP server that receives untrusted
// linux/amd64 binaries in a POST request and then executes them in
// a gvisor sandbox using Docker, returning the output as a response
// to the POST.
//
// It's part of the Go playground (https://play.golang.org/).
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	api "sandbox/api/gen"
	"sandbox/internal"
	"sync"
	"time"
)

var (
	readyContainer chan *Container
)

type Container struct {
	name string

	stdin  io.WriteCloser
	stdout *limitedWriter
	stderr *limitedWriter

	cmd       *exec.Cmd
	cancelCmd context.CancelFunc

	waitErr chan error // 1-buffered; receives error from WaitOrStop(..., cmd, ...)
}

// containedStartMessage is the first thing written to stdout by the
// gvisor-contained process when it starts up. This lets the parent HTTP
// server know that a particular container is ready to run a binary.
const containedStartMessage = "golang-gvisor-process-started\n"

// containedStderrHeader is written to stderr after the gvisor-contained process
// successfully reads the processMeta JSON line + executable binary from stdin,
// but before it's run.
var containedStderrHeader = []byte("golang-gvisor-process-got-input\n")

// processMeta is the JSON sent to the gvisor container before the untrusted binary.
// It currently contains only the arguments to pass to the binary.
// It might contain environment or other things later.
type processMeta struct {
	Args []string `json:"args"`
}

func (c *Container) Close() {
	setContainerWanted(c.name, false)

	c.cancelCmd()
	if err := c.Wait(); err != nil {
		log.Printf("error in c.Wait() for %q: %v", c.name, err)
	}
}

func (c *Container) Wait() error {
	err := <-c.waitErr
	c.waitErr <- err
	return err
}

func runOldHandler(w http.ResponseWriter, r *http.Request) {
	t0 := time.Now()
	tlast := t0
	var logmu sync.Mutex
	logf := func(format string, args ...interface{}) {
		if !*dev {
			return
		}
		logmu.Lock()
		defer logmu.Unlock()
		t := time.Now()
		d := t.Sub(tlast)
		d0 := t.Sub(t0)
		tlast = t
		log.Print(fmt.Sprintf("+%10v +%10v ", d0, d) + fmt.Sprintf(format, args...))
	}
	logf("/run")

	if r.Method != "POST" {
		http.Error(w, "expected a POST", http.StatusBadRequest)
		return
	}

	// Bound the number of requests being processed at once.
	// (Before we slurp the binary into memory)
	select {
	case runSem <- struct{}{}:
	case <-r.Context().Done():
		return
	}
	defer func() { <-runSem }()

	bin, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBinarySize))
	if err != nil {
		log.Printf("failed to read request body: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	logf("read %d bytes", len(bin))

	c, err := getContainer(r.Context())
	//c, err := startContainer(r.Context())
	if err != nil {
		if cerr := r.Context().Err(); cerr != nil {
			log.Printf("GetContainer, client side cancellation: %v", cerr)
			return
		}
		http.Error(w, "failed to get container", http.StatusInternalServerError)
		log.Printf("failed to get container: %v", err)
		return
	}
	logf("got container %s", c.name)

	// тут надо скомпилить, если не скомпилено и копировавть в контейнер

	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	closed := make(chan struct{})
	defer func() {
		logf("leaving handler; about to close container")
		cancel()
		<-closed
	}()
	go func() {
		<-ctx.Done()
		if ctx.Err() == context.DeadlineExceeded {
			logf("timeout")
		}
		c.Close()
		close(closed)
	}()
	var meta processMeta
	meta.Args = r.Header["X-Argument"]
	metaJSON, _ := json.Marshal(&meta)
	metaJSON = append(metaJSON, '\n')
	if _, err := c.stdin.Write(metaJSON); err != nil {
		log.Printf("failed to write meta to child: %v", err)
		http.Error(w, "unknown error during docker run", http.StatusInternalServerError)
		return
	}
	if _, err := c.stdin.Write([]byte("echo \"Hello, World!\";")); err != nil {
		log.Printf("failed to write binary to child: %v", err)
		http.Error(w, "unknown error during docker run", http.StatusInternalServerError)
		return
	}
	c.stdin.Close()
	logf("wrote+closed")
	err = c.Wait()
	select {
	case <-ctx.Done():
		// Timed out or canceled before or exactly as Wait returned.
		// Either way, treat it as a timeout.
		sendError(w, "timeout running program")
		return
	default:
		logf("finished running; about to close container")
		cancel()
	}
	res := &api.SandboxResponse{}
	if err != nil {
		if c.stderr.n < 0 || c.stdout.n < 0 {
			// Do not send truncated output, just send the error.
			sendError(w, errTooMuchOutput.Error())
			return
		}
		var ee *exec.ExitError
		if !errors.As(err, &ee) {
			http.Error(w, "unknown error during docker run", http.StatusInternalServerError)
			return
		}
		res.ExitCode = ee.ExitCode()
	}
	res.Stdout = c.stdout.dst.Bytes()
	res.Stderr = cleanStderr(c.stderr.dst.Bytes())
	sendRunResponse(w, res)
}

// runInGvisor is run when we're now inside gvisor. We have no network
// at this point. We can read our binary in from stdin and then run
// it.
func runInGvisor() {
	const binPath = "/tmpfs/play"
	if _, err := io.WriteString(os.Stdout, containedStartMessage); err != nil {
		log.Fatalf("writing to stdout: %v", err)
	}
	slurp, err := io.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("reading stdin in contained mode: %v", err)
	}
	nl := bytes.IndexByte(slurp, '\n')
	if nl == -1 {
		log.Fatalf("no newline found in input")
	}
	metaJSON, bin := slurp[:nl], slurp[nl+1:]

	if err := os.WriteFile(binPath, bin, 0755); err != nil {
		log.Fatalf("writing contained binary: %v", err)
	}
	defer os.Remove(binPath) // not that it matters much, this container will be nuked

	var meta processMeta
	if err := json.NewDecoder(bytes.NewReader(metaJSON)).Decode(&meta); err != nil {
		log.Fatalf("error decoding JSON meta: %v", err)
	}

	if _, err := os.Stderr.Write(containedStderrHeader); err != nil {
		log.Fatalf("writing header to stderr: %v", err)
	}

	cmd := exec.Command(binPath)
	cmd.Args = append(cmd.Args, meta.Args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		log.Fatalf("cmd.Start(): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout-500*time.Millisecond)
	defer cancel()
	if err = internal.WaitOrStop(ctx, cmd, os.Interrupt, 250*time.Millisecond); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Fprintln(os.Stderr, "timeout running program")
		}
	}
	os.Exit(errExitCode(err))
	return
}

var (
	wantedMu        sync.Mutex
	containerWanted = map[string]bool{}
)

// setContainerWanted records whether a named container is wanted or
// not. Any unwanted containers are cleaned up asynchronously as a
// sanity check against leaks.
//
// TODO(bradfitz): add leak checker (background docker ps loop)
func setContainerWanted(name string, wanted bool) {
	wantedMu.Lock()
	defer wantedMu.Unlock()
	if wanted {
		containerWanted[name] = true
	} else {
		delete(containerWanted, name)
	}
}

func isContainerWanted(name string) bool {
	wantedMu.Lock()
	defer wantedMu.Unlock()
	return containerWanted[name]
}

func getContainer(ctx context.Context) (*Container, error) {
	select {
	case c := <-readyContainer:
		return c, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func startContainer(ctx context.Context, image, imageTag string) (c *Container, err error) {
	name := fmt.Sprintf("play_run_%s_%s", imageTag, internal.RandHex(8))
	setContainerWanted(name, true)

	runtime := "--runtime=runsc"
	if *dev {
		runtime = ""
	}

	cmd := exec.Command("docker", "run",
		"-d",
		"--Name="+name,
		"--rm",
		"--tmpfs=/tmpfs:exec",
		"-it",
		runtime,
		"--network=none",
		"--memory="+fmt.Sprint(memoryLimitBytes),
		image,
		//"--mode=contained",

		"tail",
		"-f",
		"/dev/null",
	)

	log.Printf("Prepare command %s", cmd.String())

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()
	stdout := &limitedWriter{dst: &bytes.Buffer{}, n: maxOutputSize + int64(len(containedStartMessage))}
	stderr := &limitedWriter{dst: &bytes.Buffer{}, n: maxOutputSize}
	cmd.Stdout = &switchWriter{switchAfter: []byte(containedStartMessage), dst1: pw, dst2: stdout}
	cmd.Stderr = stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	c = &Container{
		name:      name,
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
		cmd:       cmd,
		cancelCmd: cancel,
		waitErr:   make(chan error, 1),
	}

	go func() {
		terr := internal.WaitOrStop(ctx, cmd, os.Interrupt, 250*time.Millisecond)
		c.waitErr <- terr
	}()

	defer func() {
		if err != nil {
			c.Close()
		}
	}()

	startErr := make(chan error, 1)
	go func() {
		buf := make([]byte, len(containedStartMessage))
		_, err := io.ReadFull(pr, buf)
		if err != nil {
			startErr <- fmt.Errorf("error reading header from sandbox container: %v", err)
		} else if string(buf) != containedStartMessage {
			startErr <- fmt.Errorf("sandbox container sent wrong header %q; want %q", buf, containedStartMessage)
		} else {
			startErr <- nil
		}
	}()

	timer := time.NewTimer(startTimeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		err := fmt.Errorf("timeout starting container %q", name)
		cancel()
		<-startErr
		return nil, err

	case err := <-startErr:
		if err != nil {
			return nil, err
		}
	}

	log.Printf("started container %q", name)
	return c, nil
}

// limitedWriter is an io.Writer that returns an errTooMuchOutput when the cap (n) is hit.
type limitedWriter struct {
	dst *bytes.Buffer
	n   int64 // max bytes remaining
}

// Write is an io.Writer function that returns errTooMuchOutput when the cap (n) is hit.
//
// Partial data will be written to dst if p is larger than n, but errTooMuchOutput will be returned.
func (l *limitedWriter) Write(p []byte) (int, error) {
	defer func() { l.n -= int64(len(p)) }()

	if l.n <= 0 {
		return 0, errTooMuchOutput
	}

	if int64(len(p)) > l.n {
		n, err := l.dst.Write(p[:l.n])
		if err != nil {
			return n, err
		}
		return n, errTooMuchOutput
	}

	return l.dst.Write(p)
}

// switchWriter writes to dst1 until switchAfter is written, the it writes to dst2.
type switchWriter struct {
	dst1        io.Writer
	dst2        io.Writer
	switchAfter []byte
	buf         []byte
	found       bool
}

func (s *switchWriter) Write(p []byte) (int, error) {
	if s.found {
		return s.dst2.Write(p)
	}

	s.buf = append(s.buf, p...)
	i := bytes.Index(s.buf, s.switchAfter)
	if i == -1 {
		if len(s.buf) >= len(s.switchAfter) {
			s.buf = s.buf[len(s.buf)-len(s.switchAfter)+1:]
		}
		return s.dst1.Write(p)
	}

	s.found = true
	nAfter := len(s.buf) - (i + len(s.switchAfter))
	s.buf = nil

	n, err := s.dst1.Write(p[:len(p)-nAfter])
	if err != nil {
		return n, err
	}
	n2, err := s.dst2.Write(p[len(p)-nAfter:])
	return n + n2, err
}

func cleanStderr(x []byte) []byte {
	i := bytes.Index(x, containedStderrHeader)
	if i == -1 {
		return x
	}
	return x[i+len(containedStderrHeader):]
}
