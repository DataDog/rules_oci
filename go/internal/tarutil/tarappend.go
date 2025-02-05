package tarutil

import (
	"archive/tar"
	"fmt"
	"io"
	"io/fs"
	"os"
	"strings"
	"time"
)

const (
	ownerRead    = 0o0400
	ownerWrite   = 0o0200
	ownerExecute = 0o0100

	groupRead    = 0o0040
	groupWrite   = 0o0020
	groupExecute = 0o0010

	otherRead    = 0o0004
	otherWrite   = 0o0002
	otherExecute = 0o0001

	setuid = 0o4000
	setgid = 0o2000
	sticky = 0o1000

	allSupportedBits = ownerRead | ownerWrite | ownerExecute | groupRead | groupWrite | groupExecute | otherRead | otherWrite | otherExecute | setuid | setgid | sticky
)

func AppendFileToTarWriter(
	hostPath string,
	tarPath string,
	tarMode int64,
	tarUid,
	tarGid *int,
	tw *tar.Writer,
) error {
	f, err := os.Open(hostPath)
	if err != nil {
		return fmt.Errorf("error opening file %s: %w", hostPath, err)
	}
	defer f.Close()

	s, err := f.Stat()
	if err != nil {
		return fmt.Errorf("error stating file %s: %w", hostPath, err)
	}

	m := s.Mode()

	h := baseHeader(
		/* defaultName */ tarPath,
		/* defaultMode */ int64(m)&allSupportedBits,
	)

	switch {
	case m.IsRegular():
		h.Typeflag = tar.TypeReg
		h.Size = s.Size()
	case s.IsDir():
		h.Typeflag = tar.TypeDir
		h.Name = ensureTrailingSlash(tarPath)
		h.Size = 0
	case m.Type() == fs.ModeSymlink:
		h.Typeflag = tar.TypeSymlink
		linkName, err := os.Readlink(hostPath)
		if err != nil {
			return fmt.Errorf("error reading symlink %s: %w", hostPath, err)
		}
		h.Linkname = linkName
		h.Size = 0
	case m.Type() == fs.ModeDevice:
		h.Typeflag = tar.TypeBlock
		h.Size = 0
	case m.Type() == fs.ModeCharDevice:
		h.Typeflag = tar.TypeChar
		h.Size = 0
	case m.Type() == fs.ModeNamedPipe:
		h.Typeflag = tar.TypeFifo
		h.Size = 0
	case m.Type() == fs.ModeSocket:
		return fmt.Errorf("archive/tar: sockets not supported")
	default:
		return fmt.Errorf("archive/tar: unknown file mode %v", m)
	}

	updateMode(h, tarMode)
	updateUid(h, tarUid)
	updateGid(h, tarGid)

	if err := tw.WriteHeader(h); err != nil {
		return fmt.Errorf("error writing tar header for %s: %w", hostPath, err)
	}

	if m.IsRegular() {
		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("error writing %s into tarball: %w", hostPath, err)
		}
	}

	return nil
}

func AppendSymlinkToTarWriter(
	tarPath string,
	tarTarget string,
	tarMode int64,
	tarUid,
	tarGid *int,
	tw *tar.Writer,
) error {
	h := baseHeader(
		/* defaultName */ tarPath,
		/* defaultMode */ 0o777,
	)

	h.Typeflag = tar.TypeSymlink
	h.Linkname = tarTarget
	h.Size = 0

	updateMode(h, tarMode)
	updateUid(h, tarUid)
	updateGid(h, tarGid)

	if err := tw.WriteHeader(h); err != nil {
		return fmt.Errorf("error writing tar header for symlink %s: %w", tarPath, err)
	}

	return nil
}

func baseHeader(defaultName string, defaultMode int64) *tar.Header {
	return &tar.Header{
		Name: defaultName,

		AccessTime: time.Unix(0, 0),
		ChangeTime: time.Unix(0, 0),
		ModTime:    time.Unix(0, 0),

		Mode: defaultMode,

		Gid: 0,
		Uid: 0,
	}
}

func ensureTrailingSlash(s string) string {
	if !strings.HasSuffix(s, "/") {
		return s + "/"
	}
	return s
}

func updateMode(h *tar.Header, mode int64) {
	if mode == 0 {
		return
	}
	h.Mode = mode
}

func updateUid(h *tar.Header, uid *int) {
	if uid == nil {
		return
	}
	h.Uid = *uid
}

func updateGid(h *tar.Header, gid *int) {
	if gid == nil {
		return
	}
	h.Gid = *gid
}
