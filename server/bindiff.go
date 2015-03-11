package server

import (
	"crypto/sha1"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

const (
	patchesDirectory = "patches/"
)

func init() {
	err := os.MkdirAll(patchesDirectory, os.ModeDir|0700)
	if err != nil {
		log.Fatal("Could not create directory for storing patches: %q", err)
	}
}

func fileExists(s string) bool {
	if _, err := os.Stat(s); err == nil {
		return true
	}
	return false
}

func fileHash(s string) string {
	var err error
	var fp *os.File

	h := sha1.New()

	if fp, err = os.Open(s); err != nil {
		log.Fatal("Failed to open file %s: %q", s, err)
	}
	defer fp.Close()

	if _, err = io.Copy(h, fp); err != nil {
		log.Fatal("Failed to read file %s: %q", s, err)
	}

	return fmt.Sprintf("%x", h.Sum(nil))
}

func bspatch(oldfile string, newfile string, patchfile string) (err error) {
	if !fileExists(oldfile) {
		return fmt.Errorf("File %s does not exist.", oldfile)
	}

	if !fileExists(patchfile) {
		return fmt.Errorf("File %s does not exist.", oldfile)
	}

	cmd := exec.Command(
		"bspatch",
		oldfile,
		newfile,
		patchfile,
	)

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Failed to apply patch with bspatch: %q", err)
	}

	return nil
}

func bsdiff(oldfile string, newfile string) (patchfile string, err error) {
	if !fileExists(oldfile) {
		return "", fmt.Errorf("File %s does not exist.", oldfile)
	}

	if !fileExists(newfile) {
		return "", fmt.Errorf("File %s does not exist.", oldfile)
	}

	oldfileHash := fileHash(oldfile)
	newfileHash := fileHash(newfile)

	patchfile = patchesDirectory + fmt.Sprintf("%x", sha1.Sum([]byte(oldfileHash+newfileHash))) + ".patch"

	if fileExists(patchfile) {
		// Patch already exists, no need to compute it again.
		return patchfile, nil
	}

	cmd := exec.Command(
		"bsdiff",
		oldfile,
		newfile,
		patchfile,
	)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("Failed to generate patch with bsdiff: %q", err)
	}

	return patchfile, nil
}