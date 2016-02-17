package server

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
)

var (
	fileHashMap   map[string]string
	fileHashMapMu sync.Mutex
)

func init() {
	fileHashMap = map[string]string{}
}

var (
	generatePatchMu sync.Mutex
)

// Patch struct is a representation of a patch generated by bsdiff.
type Patch struct {
	oldfile string
	newfile string
	File    string
}

const (
	patchesDirectory = "patches/"
)

func init() {
	err := os.MkdirAll(patchesDirectory, os.ModeDir|0700)
	if err != nil {
		log.Fatalf("Could not create directory for storing patches: %q", err)
	}
}

func fileExists(s string) bool {
	if _, err := os.Stat(s); err == nil {
		return true
	}
	return false
}

func fileHash(s string) string {
	fileHashMapMu.Lock()
	defer fileHashMapMu.Unlock()

	if hash, ok := fileHashMap[s]; ok {
		return hash
	}

	var err error
	var fp *os.File

	h := sha256.New()

	if fp, err = os.Open(s); err != nil {
		log.Fatalf("Failed to open file %s: %q", s, err)
	}
	defer fp.Close()

	if _, err = io.Copy(h, fp); err != nil {
		log.Fatalf("Failed to read file %s: %q", s, err)
	}

	fileHashMap[s] = fmt.Sprintf("%x", h.Sum(nil))
	return fileHashMap[s]
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

	patchfile = patchesDirectory + fmt.Sprintf("%x", sha256.Sum256([]byte(oldfileHash+"|"+newfileHash)))

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

// generatePatch compares the contents of two URLs and generates a patch.
func generatePatch(oldfileURL string, newfileURL string) (p *Patch, err error) {
	generatePatchMu.Lock()
	defer generatePatchMu.Unlock()

	p = new(Patch)

	if p.oldfile, err = downloadAsset(oldfileURL); err != nil {
		return nil, err
	}

	if p.newfile, err = downloadAsset(newfileURL); err != nil {
		return nil, err
	}

	if p.File, err = bsdiff(p.oldfile, p.newfile); err != nil {
		return nil, err
	}

	return p, nil
}
