//go:build e2e

package e2e_test

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/padok-team/tfautomv/internal/terraform"
)

// ANSI escape sequence used for color output
const colorEscapeSequence = "\x1b"

func TestE2E(t *testing.T) {
	tt := []struct {
		name    string
		workdir string
		args    []string

		wantChanges       int
		wantOutputInclude []string
		wantOutputExclude []string

		skip       bool
		skipReason string
	}{
		{
			name:        "same attributes",
			workdir:     filepath.Join("testdata", "same-attributes"),
			wantChanges: 0,
			wantOutputInclude: []string{
				colorEscapeSequence,
			},
		},
		{
			name:        "requires dependency analysis",
			workdir:     filepath.Join("testdata", "requires-dependency-analysis"),
			wantChanges: 0,
			wantOutputInclude: []string{
				colorEscapeSequence,
			},
			skip:       true,
			skipReason: "tfautomv cannot yet solve this case",
		},
		{
			name:        "same type",
			workdir:     filepath.Join("testdata", "same-type"),
			wantChanges: 0,
			wantOutputInclude: []string{
				colorEscapeSequence,
			},
		},
		{
			name:        "different attributes",
			workdir:     filepath.Join("testdata", "different-attributes"),
			wantChanges: 2,
			wantOutputInclude: []string{
				colorEscapeSequence,
			},
		},
		{
			name:    "ignore different attributes",
			workdir: filepath.Join("testdata", "different-attributes"),
			args: []string{
				"-ignore=everything:random_pet:length",
			},
			wantChanges: 1,
			wantOutputInclude: []string{
				colorEscapeSequence,
			},
		},
		{
			name:    "no color",
			workdir: filepath.Join("testdata", "same-attributes"),
			args: []string{
				"-no-color",
			},
			wantChanges: 0,
			wantOutputExclude: []string{
				colorEscapeSequence,
			},
		},
		{
			name:        "terraform cloud",
			workdir:     filepath.Join("testdata", "terraform-cloud"),
			wantChanges: 0,
			skip:        true,
			skipReason:  "tfautomv is currently incompatible with Terraform Cloud workspaces with the \"Remote\" execution mode.\nFor more details, see https://github.com/padok-team/tfautomv/issues/17",
		},
	}

	binPath := buildBinary(t)

	for _, tc := range tt {

		for _, outputFormat := range []string{"blocks", "commands"} {

			tcName := fmt.Sprintf("%s_output=%s", tc.name, outputFormat)
			t.Run(tcName, func(t *testing.T) {
				if tc.skip {
					t.Skip(tc.skipReason)
				}

				setupWorkdir(t, tc.workdir)

				workdir := filepath.Join(tc.workdir, "refactored-code")

				args := append(tc.args, fmt.Sprintf("-output=%s", outputFormat))

				tfautomvCmd := exec.Command(binPath, args...)
				tfautomvCmd.Dir = workdir

				var outputStdout bytes.Buffer
				tfautomvCmd.Stdout = io.MultiWriter(&outputStdout, os.Stderr)

				var outputStderr bytes.Buffer
				tfautomvCmd.Stderr = io.MultiWriter(&outputStderr, os.Stderr)

				if err := tfautomvCmd.Run(); err != nil {
					t.Fatalf("running tfautomv: %v", err)
				}

				outputStr := outputStdout.String() + outputStderr.String()
				t.Logf("full output:\n%q", outputStr)

				for _, s := range tc.wantOutputInclude {
					if !strings.Contains(outputStr, s) {
						t.Errorf("output should contain %q but does not", s)
					}
				}
				for _, s := range tc.wantOutputExclude {
					if strings.Contains(outputStr, s) {
						t.Errorf("output should not contain %q but does", s)
					}
				}

				switch outputFormat {
				case "blocks":
					// nothing to do, integrated in moved blocks
				case "commands":
					// run terraform state mv commands

					tfMvCmds := strings.Split(outputStdout.String(), "\n")

					for _, tfMvCmd := range tfMvCmds {

						// ignore empty lines, such as the last one
						if tfMvCmd == "" {
							continue
						}

						t.Logf("running %q", tfMvCmd)

						cmdName := strings.Fields(tfMvCmd)[0]
						args := strings.Fields(tfMvCmd)[1:]
						// trim " around args
						trimArgs := make([]string, 0, len(args))
						for _, arg := range args {
							trimArgs = append(trimArgs, strings.Trim(arg, "\""))
						}

						cmd := exec.Command(cmdName, trimArgs...)
						cmd.Dir = workdir

						var output bytes.Buffer
						cmd.Stdout = io.MultiWriter(&output, os.Stderr)
						cmd.Stderr = io.MultiWriter(&output, os.Stderr)

						if err := cmd.Run(); err != nil {
							t.Fatalf("running %q: %v", tfMvCmd, err)
						}
					}
				}

				plan, err := terraform.NewRunner(workdir).Plan()
				if err != nil {
					t.Fatalf("terraform plan (after addings moves): %v", err)
				}

				changes := plan.NumChanges()
				if changes != tc.wantChanges {
					t.Errorf("%d changes remaining, want %d", changes, tc.wantChanges)
				}
			})
		}
	}
}

func buildBinary(t *testing.T) string {
	t.Helper()

	rootDir, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("could not get root directory: %v", err)
	}

	buildCmd := exec.Command("make", "build")
	buildCmd.Dir = rootDir
	buildCmd.Stdout = os.Stderr
	buildCmd.Stderr = os.Stderr

	t.Log("Building tfautomv binary...")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("make build: %v", err)
	}

	binPath := filepath.Join(rootDir, "bin", "tfautomv")
	return binPath
}

func setupWorkdir(t *testing.T, workdir string) {
	t.Helper()

	originalWorkdir := filepath.Join(workdir, "original-code")
	refactoredWorkdir := filepath.Join(workdir, "refactored-code")

	filesToRemove := []string{
		filepath.Join(originalWorkdir, "terraform.tfstate"),
		filepath.Join(originalWorkdir, ".terraform.lock.hcl"),
		filepath.Join(refactoredWorkdir, "terraform.tfstate"),
		filepath.Join(refactoredWorkdir, ".terraform.lock.hcl"),
		filepath.Join(refactoredWorkdir, "moves.tf"),
	}
	for _, f := range filesToRemove {
		ensureFileRemoved(t, f)
	}

	directoriesToRemove := []string{
		filepath.Join(originalWorkdir, ".terraform"),
		filepath.Join(refactoredWorkdir, ".terraform"),
	}
	for _, d := range directoriesToRemove {
		ensureDirectoryRemoved(t, d)
	}

	original := terraform.NewRunner(originalWorkdir)

	if err := original.Init(); err != nil {
		t.Fatal(err)
	}
	if err := original.Apply(); err != nil {
		t.Fatal(err)
	}

	os.Rename(
		filepath.Join(originalWorkdir, "terraform.tfstate"),
		filepath.Join(refactoredWorkdir, "terraform.tfstate"),
	)
}

func ensureFileRemoved(t *testing.T, path string) {
	t.Helper()

	err := os.Remove(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("could not remove file %q: %v", path, err)
	}
}

func ensureDirectoryRemoved(t *testing.T, path string) {
	t.Helper()

	err := os.RemoveAll(path)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("could not remove directory %q: %v", path, err)
	}
}
