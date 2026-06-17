package archive

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/mholt/archives"
)

type deferredHardLink struct {
	targetPath string
	linkToPath string
}

func ExtractFileSystem(archivePath string) (string, error) {
	tmpRootfsPath, err := createTempRootfsPath()
	if err != nil {
		return "", err
	}

	ctx := context.Background()
	extractor, stream, archiveFile, err := openArchiveExtractor(ctx, archivePath)
	if err != nil {
		return "", err
	}
	defer archiveFile.Close()

	pendingHardLinks, err := extractArchiveEntries(ctx, extractor, stream, tmpRootfsPath)
	if err != nil {
		return "", err
	}

	if err := restoreDeferredHardLinks(pendingHardLinks); err != nil {
		return "", err
	}

	return tmpRootfsPath, nil
}

func createTempRootfsPath() (string, error) {
	uid := uuid.New()
	uniqueID := uid.String()
	if uniqueID == "" {
		return "", UUIDCreationError
	}

	tmpRootfsPath := filepath.Join("/tmp", uniqueID, "rootfs")
	resolvedPath, err := filepath.Abs(tmpRootfsPath)
	if err != nil {
		return "", fmt.Errorf("%w: %v", RootfsPathResolutionError, err)
	}

	if err := os.MkdirAll(resolvedPath, 0o755); err != nil {
		return "", fmt.Errorf("%w: %v", RootfsDirectoryCreationError, err)
	}

	return resolvedPath, nil
}

func openArchiveExtractor(ctx context.Context, archivePath string) (extractor archives.Extractor, stream io.Reader, file *os.File, retErr error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %v", ArchiveOpenError, err)
	}

	defer func() {
		if retErr != nil {
			_ = file.Close()
		}
	}()

	format, stream, err := archives.Identify(ctx, archivePath, file)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%w: %v", ArchiveIdentifyError, err)
	}

	parsedExtractor, ok := format.(archives.Extractor)
	if !ok {
		return nil, nil, nil, fmt.Errorf("%w: format %s", ArchiveExtractionUnsupportedError, format.Extension())
	}

	return parsedExtractor, stream, file, nil
}

func extractArchiveEntries(ctx context.Context, extractor archives.Extractor, stream io.Reader, rootfsPath string) ([]deferredHardLink, error) {
	var pendingHardLinks []deferredHardLink

	err := extractor.Extract(ctx, stream, func(_ context.Context, info archives.FileInfo) error {
		return extractArchiveEntry(rootfsPath, info, &pendingHardLinks)
	})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ArchiveExtractError, err)
	}

	return pendingHardLinks, nil
}

func extractArchiveEntry(rootfsPath string, info archives.FileInfo, pendingHardLinks *[]deferredHardLink) error {
	targetPath, err := resolveArchivePath(rootfsPath, info.NameInArchive)
	if err != nil {
		return err
	}
	if targetPath == rootfsPath {
		return nil
	}

	if info.IsDir() {
		return extractDirectoryEntry(targetPath, info)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("%w: %v", TargetDirectoryCreationError, err)
	}

	if hdr, ok := info.Header.(*tar.Header); ok && hdr.Typeflag == tar.TypeLink {
		return extractHardLinkEntry(rootfsPath, targetPath, hdr, info, pendingHardLinks)
	}

	if info.Mode()&os.ModeSymlink != 0 {
		return extractSymlinkEntry(targetPath, info)
	}

	return extractRegularFileEntry(targetPath, info)
}

func extractDirectoryEntry(targetPath string, info archives.FileInfo) error {
	if err := os.MkdirAll(targetPath, info.Mode()); err != nil {
		return fmt.Errorf("%w: %v", TargetDirectoryCreationError, err)
	}
	return applyArchiveMetadata(targetPath, info, true)
}

func extractHardLinkEntry(rootfsPath, targetPath string, hdr *tar.Header, info archives.FileInfo, pendingHardLinks *[]deferredHardLink) error {
	linkToPath, err := resolveArchivePath(rootfsPath, hdr.Linkname)
	if err != nil {
		return err
	}

	_ = os.RemoveAll(targetPath)
	if err := os.Link(linkToPath, targetPath); err != nil {
		if os.IsNotExist(err) {
			*pendingHardLinks = append(*pendingHardLinks, deferredHardLink{targetPath: targetPath, linkToPath: linkToPath})
			return nil
		}
		return fmt.Errorf("%w: %v", HardLinkCreateError, err)
	}

	return applyArchiveMetadata(targetPath, info, false)
}

func extractSymlinkEntry(targetPath string, info archives.FileInfo) error {
	_ = os.RemoveAll(targetPath)
	if err := os.Symlink(info.LinkTarget, targetPath); err != nil {
		return fmt.Errorf("%w: %v", SymlinkCreateError, err)
	}
	return applyArchiveMetadata(targetPath, info, false)
}

func extractRegularFileEntry(targetPath string, info archives.FileInfo) error {
	src, err := info.Open()
	if err != nil {
		return fmt.Errorf("%w: %v", ArchiveFileOpenError, err)
	}
	defer src.Close()

	dst, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return fmt.Errorf("%w: %v", TargetFileOpenError, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("%w: %v", ArchiveFileCopyError, err)
	}

	return applyArchiveMetadata(targetPath, info, true)
}

func restoreDeferredHardLinks(pendingHardLinks []deferredHardLink) error {
	for len(pendingHardLinks) > 0 {
		var remaining []deferredHardLink
		progress := false

		for _, hardLink := range pendingHardLinks {
			if err := os.MkdirAll(filepath.Dir(hardLink.targetPath), 0o755); err != nil {
				return fmt.Errorf("%w: %v", TargetDirectoryCreationError, err)
			}

			_ = os.RemoveAll(hardLink.targetPath)
			if err := os.Link(hardLink.linkToPath, hardLink.targetPath); err != nil {
				if os.IsNotExist(err) {
					remaining = append(remaining, hardLink)
					continue
				}
				return fmt.Errorf("%w: %v", HardLinkCreateError, err)
			}
			progress = true
		}

		if !progress {
			first := remaining[0]
			return fmt.Errorf("%w: missing link target for %q -> %q", HardLinkRestoreError, first.targetPath, first.linkToPath)
		}

		pendingHardLinks = remaining
	}

	return nil
}

func resolveArchivePath(rootfsPath, archivePath string) (string, error) {
	entryPath := path.Clean(archivePath)
	entryPath = strings.TrimPrefix(entryPath, "/")

	targetPath := filepath.Join(rootfsPath, filepath.FromSlash(entryPath))
	rel, err := filepath.Rel(rootfsPath, targetPath)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ArchivePathResolutionError, err)
	}
	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("%w: %q", ArchivePathTraversalError, archivePath)
	}
	return targetPath, nil
}

func applyArchiveMetadata(targetPath string, info archives.FileInfo, applyMode bool) error {
	if applyMode && info.Mode()&os.ModeSymlink == 0 {
		specialBits := info.Mode() & (os.ModeSetuid | os.ModeSetgid | os.ModeSticky)
		if err := os.Chmod(targetPath, info.Mode().Perm()|specialBits); err != nil {
			return fmt.Errorf("%w: %w", ArchiveMetadataApplyError, fmt.Errorf("%w: %v", FilePermissionApplyError, err))
		}
	}

	hdr, ok := info.Header.(*tar.Header)
	if !ok {
		return nil
	}

	if err := os.Lchown(targetPath, hdr.Uid, hdr.Gid); err != nil {
		return fmt.Errorf("%w: %w", ArchiveMetadataApplyError, fmt.Errorf("%w: %v", FileOwnershipApplyError, err))
	}
	return nil
}
