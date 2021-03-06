// Copyright 2020 The Kubernetes Authors.
// SPDX-License-Identifier: Apache-2.0

package git

import (
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/api/filesys"
	"sigs.k8s.io/kustomize/api/types"
)

// Arbitrary, but non-infinite, timeout for running commands.
const defaultDuration = 27 * time.Second

// gitRunner runs the external git binary.
type gitRunner struct {
	gitProgram string
	duration   time.Duration
	dir        filesys.ConfirmedDir
}

// newCmdRunner returns a gitRunner if it can find the binary.
// It also creats a temp directory for cloning repos.
func newCmdRunner() (*gitRunner, error) {
	gitProgram, err := exec.LookPath("git")
	if err != nil {
		return nil, errors.Wrap(err, "no 'git' program on path")
	}
	dir, err := filesys.NewTmpConfirmedDir()
	if err != nil {
		return nil, err
	}
	return &gitRunner{
		gitProgram: gitProgram,
		duration:   defaultDuration,
		dir:        dir,
	}, nil
}

// run a command with a timeout.
func (r gitRunner) run(args ...string) (err error) {
	ch := make(chan bool, 1)
	defer close(ch)
	//nolint: gosec
	cmd := exec.Command(r.gitProgram, args...)
	cmd.Dir = r.dir.String()
	timer := time.NewTimer(r.duration)
	defer timer.Stop()
	go func() {
		_, err = cmd.CombinedOutput()
		ch <- true
	}()
	select {
	case <-ch:
		if err != nil {
			return errors.Wrapf(err, "git cmd = '%s'", cmd.String())
		}
		return nil
	case <-timer.C:
		return types.NewErrTimeOut(r.duration, cmd.String())
	}
}
