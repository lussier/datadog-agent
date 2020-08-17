// +build linux_bpf

package bytecode

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
)

// TODO: replace hard-coded arch
var arch = "x86"

var baseDirGlobs = []string{
	"/usr/src/kernels/*",
	"/usr/src/linux-*",
}

var subDirs = []string{
	"include",
	"include/uapi",
	"include/generated/uapi",
	fmt.Sprintf("arch/%s/include", arch),
	fmt.Sprintf("arch/%s/include/uapi", arch),
	fmt.Sprintf("arch/%s/include/generated", arch),
}

func getKernelIncludePaths() []string {
	var matches []string
	for _, glob := range baseDirGlobs {
		matches, _ = filepath.Glob(glob)
		if len(matches) > 0 {
			break
		}
	}

	var baseDirs []string
	for _, m := range matches {
		if isDirectory(m) {
			baseDirs = append(baseDirs, m)
		}
	}

	var includePaths []string
	for _, dir := range baseDirs {
		for _, sub := range subDirs {
			includePaths = append(includePaths, path.Join(dir, sub))
		}
	}

	return includePaths
}

func isDirectory(path string) bool {
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	return fileInfo.IsDir()
}
