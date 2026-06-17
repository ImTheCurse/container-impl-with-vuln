package archive

import "errors"

var (
	UUIDCreationError                 = errors.New("failed to create uuid for rootfs path")
	RootfsPathResolutionError         = errors.New("failed to resolve temporary rootfs path")
	RootfsDirectoryCreationError      = errors.New("failed to create temporary rootfs directory")
	ArchiveOpenError                  = errors.New("failed to open rootfs archive")
	ArchiveIdentifyError              = errors.New("failed to identify archive format")
	ArchiveExtractionUnsupportedError = errors.New("archive format does not support extraction")
	ArchiveExtractError               = errors.New("failed to extract archive")
	ArchivePathResolutionError        = errors.New("failed to resolve archive path")
	ArchivePathTraversalError         = errors.New("archive path escapes rootfs")
	TargetDirectoryCreationError      = errors.New("failed to create target directory")
	HardLinkCreateError               = errors.New("failed to create hard link")
	SymlinkCreateError                = errors.New("failed to create symlink")
	ArchiveFileOpenError              = errors.New("failed to open archive file entry")
	TargetFileOpenError               = errors.New("failed to open target file")
	ArchiveFileCopyError              = errors.New("failed to copy archive file entry")
	ArchiveMetadataApplyError         = errors.New("failed to apply archive metadata")
	FilePermissionApplyError          = errors.New("failed to apply file permissions")
	FileOwnershipApplyError           = errors.New("failed to apply file ownership")
	HardLinkRestoreError              = errors.New("failed to restore deferred hard links")
)
