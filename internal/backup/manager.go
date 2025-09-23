package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/techishthoughts/GitPersona/internal/observability"
)

// BackupManager handles backup and disaster recovery
type BackupManager struct {
	logger      observability.Logger
	backupDir   string
	retention   time.Duration
	compression bool
	encryption  bool
	mutex       sync.RWMutex
}

// BackupMetadata contains information about a backup
type BackupMetadata struct {
	ID          string            `json:"id"`
	Type        string            `json:"type"`
	CreatedAt   time.Time         `json:"created_at"`
	Size        int64             `json:"size"`
	Checksum    string            `json:"checksum"`
	Files       []BackupFile      `json:"files"`
	Tags        []string          `json:"tags"`
	Description string            `json:"description"`
	Encrypted   bool              `json:"encrypted"`
	Compressed  bool              `json:"compressed"`
	Metadata    map[string]string `json:"metadata"`
}

// BackupFile represents a file in a backup
type BackupFile struct {
	Path         string    `json:"path"`
	RelativePath string    `json:"relative_path"`
	Size         int64     `json:"size"`
	Modified     time.Time `json:"modified"`
	Checksum     string    `json:"checksum"`
	Permissions  uint32    `json:"permissions"`
}

// RestoreOptions configures backup restoration
type RestoreOptions struct {
	TargetDir     string
	Overwrite     bool
	Selective     []string // Specific files to restore
	DryRun        bool
	PreservePerms bool
}

// BackupOptions configures backup creation
type BackupOptions struct {
	Type        string
	Description string
	Tags        []string
	Include     []string
	Exclude     []string
	Compress    bool
	Encrypt     bool
	Incremental bool
}

// NewBackupManager creates a new backup manager
func NewBackupManager(logger observability.Logger, backupDir string) *BackupManager {
	if backupDir == "" {
		homeDir, _ := os.UserHomeDir()
		backupDir = filepath.Join(homeDir, ".config", "gitpersona", "backups")
	}

	// Ensure backup directory exists
	_ = os.MkdirAll(backupDir, 0755)

	return &BackupManager{
		logger:      logger,
		backupDir:   backupDir,
		retention:   30 * 24 * time.Hour, // 30 days default
		compression: true,
		encryption:  false, // Disabled by default for simplicity
	}
}

// CreateBackup creates a complete system backup
func (bm *BackupManager) CreateBackup(ctx context.Context, sourcePaths []string, options *BackupOptions) (*BackupMetadata, error) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	if options == nil {
		options = &BackupOptions{
			Type:        "manual",
			Description: "Manual backup",
			Compress:    bm.compression,
			Encrypt:     bm.encryption,
		}
	}

	bm.logger.Info(ctx, "creating_backup",
		observability.F("type", options.Type),
		observability.F("source_paths", len(sourcePaths)),
		observability.F("compress", options.Compress),
		observability.F("encrypt", options.Encrypt),
	)

	// Generate backup ID
	backupID := fmt.Sprintf("%s-%d", options.Type, time.Now().Unix())

	// Create backup directory
	backupPath := filepath.Join(bm.backupDir, backupID)
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Initialize metadata
	metadata := &BackupMetadata{
		ID:          backupID,
		Type:        options.Type,
		CreatedAt:   time.Now(),
		Files:       make([]BackupFile, 0),
		Tags:        options.Tags,
		Description: options.Description,
		Compressed:  options.Compress,
		Encrypted:   options.Encrypt,
		Metadata:    make(map[string]string),
	}

	// Process each source path
	totalSize := int64(0)
	for _, sourcePath := range sourcePaths {
		if err := bm.backupPath(ctx, sourcePath, backupPath, metadata, options); err != nil {
			bm.logger.Error(ctx, "failed_to_backup_path",
				observability.F("path", sourcePath),
				observability.F("error", err.Error()),
			)
			// Continue with other paths
		}
	}

	// Calculate total size and checksum
	for _, file := range metadata.Files {
		totalSize += file.Size
	}
	metadata.Size = totalSize

	// Calculate backup checksum
	checksum, err := bm.calculateBackupChecksum(backupPath)
	if err != nil {
		bm.logger.Warn(ctx, "failed_to_calculate_backup_checksum",
			observability.F("error", err.Error()),
		)
	} else {
		metadata.Checksum = checksum
	}

	// Save metadata
	metadataPath := filepath.Join(backupPath, "metadata.json")
	if err := bm.saveMetadata(metadata, metadataPath); err != nil {
		return nil, fmt.Errorf("failed to save backup metadata: %w", err)
	}

	bm.logger.Info(ctx, "backup_created_successfully",
		observability.F("backup_id", backupID),
		observability.F("size", totalSize),
		observability.F("file_count", len(metadata.Files)),
	)

	return metadata, nil
}

// RestoreBackup restores from a backup
func (bm *BackupManager) RestoreBackup(ctx context.Context, backupID string, options *RestoreOptions) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.logger.Info(ctx, "restoring_backup",
		observability.F("backup_id", backupID),
		observability.F("target_dir", options.TargetDir),
		observability.F("dry_run", options.DryRun),
	)

	// Load backup metadata
	backupPath := filepath.Join(bm.backupDir, backupID)
	metadata, err := bm.loadMetadata(filepath.Join(backupPath, "metadata.json"))
	if err != nil {
		return fmt.Errorf("failed to load backup metadata: %w", err)
	}

	// Verify backup integrity
	if err := bm.verifyBackupIntegrity(ctx, backupPath, metadata); err != nil {
		return fmt.Errorf("backup integrity check failed: %w", err)
	}

	// Restore files
	restoredCount := 0
	for _, file := range metadata.Files {
		// Check if file should be restored
		if len(options.Selective) > 0 {
			shouldRestore := false
			for _, pattern := range options.Selective {
				if strings.Contains(file.RelativePath, pattern) {
					shouldRestore = true
					break
				}
			}
			if !shouldRestore {
				continue
			}
		}

		// Determine target path
		targetPath := filepath.Join(options.TargetDir, file.RelativePath)

		if options.DryRun {
			bm.logger.Info(ctx, "would_restore_file",
				observability.F("source", file.Path),
				observability.F("target", targetPath),
			)
			continue
		}

		// Check if target exists and handle overwrite
		if _, err := os.Stat(targetPath); err == nil && !options.Overwrite {
			bm.logger.Warn(ctx, "skipping_existing_file",
				observability.F("path", targetPath),
			)
			continue
		}

		// Restore the file
		if err := bm.restoreFile(ctx, backupPath, file, targetPath, options); err != nil {
			bm.logger.Error(ctx, "failed_to_restore_file",
				observability.F("file", file.RelativePath),
				observability.F("error", err.Error()),
			)
			continue
		}

		restoredCount++
	}

	bm.logger.Info(ctx, "backup_restoration_completed",
		observability.F("backup_id", backupID),
		observability.F("files_restored", restoredCount),
		observability.F("total_files", len(metadata.Files)),
	)

	return nil
}

// ListBackups returns a list of available backups
func (bm *BackupManager) ListBackups(ctx context.Context) ([]*BackupMetadata, error) {
	bm.mutex.RLock()
	defer bm.mutex.RUnlock()

	bm.logger.Debug(ctx, "listing_backups")

	entries, err := os.ReadDir(bm.backupDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read backup directory: %w", err)
	}

	backups := make([]*BackupMetadata, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		metadataPath := filepath.Join(bm.backupDir, entry.Name(), "metadata.json")
		if metadata, err := bm.loadMetadata(metadataPath); err == nil {
			backups = append(backups, metadata)
		}
	}

	// Sort by creation time (newest first)
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

// DeleteBackup deletes a backup
func (bm *BackupManager) DeleteBackup(ctx context.Context, backupID string) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.logger.Info(ctx, "deleting_backup",
		observability.F("backup_id", backupID),
	)

	backupPath := filepath.Join(bm.backupDir, backupID)
	if err := os.RemoveAll(backupPath); err != nil {
		return fmt.Errorf("failed to delete backup: %w", err)
	}

	bm.logger.Info(ctx, "backup_deleted",
		observability.F("backup_id", backupID),
	)

	return nil
}

// CleanupOldBackups removes backups older than the retention period
func (bm *BackupManager) CleanupOldBackups(ctx context.Context) error {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.logger.Info(ctx, "cleaning_up_old_backups",
		observability.F("retention_period", bm.retention.String()),
	)

	backups, err := bm.ListBackups(ctx)
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	cutoffTime := time.Now().Add(-bm.retention)
	deletedCount := 0

	for _, backup := range backups {
		if backup.CreatedAt.Before(cutoffTime) {
			if err := bm.DeleteBackup(ctx, backup.ID); err != nil {
				bm.logger.Error(ctx, "failed_to_delete_old_backup",
					observability.F("backup_id", backup.ID),
					observability.F("error", err.Error()),
				)
			} else {
				deletedCount++
			}
		}
	}

	bm.logger.Info(ctx, "old_backups_cleanup_completed",
		observability.F("deleted_count", deletedCount),
	)

	return nil
}

// CreateSystemSnapshot creates a snapshot of critical GitPersona files
func (bm *BackupManager) CreateSystemSnapshot(ctx context.Context) (*BackupMetadata, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Critical paths to backup
	criticalPaths := []string{
		filepath.Join(homeDir, ".config", "gitpersona"),
		filepath.Join(homeDir, ".ssh"),
		filepath.Join(homeDir, ".gitconfig"),
		filepath.Join(homeDir, ".zsh_secrets"),
		filepath.Join(homeDir, ".zshrc"),
		filepath.Join(homeDir, ".bashrc"),
	}

	// Filter existing paths
	existingPaths := make([]string, 0)
	for _, path := range criticalPaths {
		if _, err := os.Stat(path); err == nil {
			existingPaths = append(existingPaths, path)
		}
	}

	options := &BackupOptions{
		Type:        "system_snapshot",
		Description: "Automated system snapshot",
		Tags:        []string{"system", "automatic"},
		Compress:    true,
		Encrypt:     false,
	}

	return bm.CreateBackup(ctx, existingPaths, options)
}

// backupPath backs up a single path
func (bm *BackupManager) backupPath(ctx context.Context, sourcePath, backupPath string, metadata *BackupMetadata, options *BackupOptions) error {
	return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check exclude patterns
		if bm.shouldExclude(path, options.Exclude) {
			return nil
		}

		// Calculate relative path
		relPath, err := filepath.Rel(filepath.Dir(sourcePath), path)
		if err != nil {
			relPath = filepath.Base(path)
		}

		// Copy file to backup
		targetPath := filepath.Join(backupPath, relPath)
		if err := bm.copyFile(path, targetPath); err != nil {
			return fmt.Errorf("failed to copy file %s: %w", path, err)
		}

		// Calculate file checksum
		checksum, err := bm.calculateFileChecksum(path)
		if err != nil {
			bm.logger.Warn(ctx, "failed_to_calculate_file_checksum",
				observability.F("file", path),
				observability.F("error", err.Error()),
			)
			checksum = ""
		}

		// Add to metadata
		backupFile := BackupFile{
			Path:         path,
			RelativePath: relPath,
			Size:         info.Size(),
			Modified:     info.ModTime(),
			Checksum:     checksum,
			Permissions:  uint32(info.Mode().Perm()),
		}

		metadata.Files = append(metadata.Files, backupFile)
		return nil
	})
}

// restoreFile restores a single file from backup
func (bm *BackupManager) restoreFile(ctx context.Context, backupPath string, file BackupFile, targetPath string, options *RestoreOptions) error {
	sourcePath := filepath.Join(backupPath, file.RelativePath)

	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Copy file
	if err := bm.copyFile(sourcePath, targetPath); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Restore permissions if requested
	if options.PreservePerms {
		if err := os.Chmod(targetPath, os.FileMode(file.Permissions)); err != nil {
			bm.logger.Warn(ctx, "failed_to_restore_file_permissions",
				observability.F("file", targetPath),
				observability.F("error", err.Error()),
			)
		}
	}

	return nil
}

// copyFile copies a file from source to destination
func (bm *BackupManager) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			// Log: Failed to close source file
			_ = err
		}
	}()

	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			// Log: Failed to close dest file
			_ = err
		}
	}()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

// shouldExclude checks if a path should be excluded
func (bm *BackupManager) shouldExclude(path string, excludePatterns []string) bool {
	for _, pattern := range excludePatterns {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	// Default exclusions
	defaultExclusions := []string{
		".git/",
		".cache/",
		".tmp/",
		"node_modules/",
		".DS_Store",
	}

	for _, pattern := range defaultExclusions {
		if strings.Contains(path, pattern) {
			return true
		}
	}

	return false
}

// calculateFileChecksum calculates SHA256 checksum of a file
func (bm *BackupManager) calculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// calculateBackupChecksum calculates checksum for entire backup
func (bm *BackupManager) calculateBackupChecksum(backupPath string) (string, error) {
	hash := sha256.New()

	err := filepath.Walk(backupPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, _ := filepath.Rel(backupPath, path)
			hash.Write([]byte(relPath))

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer func() {
				_ = file.Close()
			}()

			if _, err := io.Copy(hash, file); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// verifyBackupIntegrity verifies the integrity of a backup
func (bm *BackupManager) verifyBackupIntegrity(ctx context.Context, backupPath string, metadata *BackupMetadata) error {
	bm.logger.Debug(ctx, "verifying_backup_integrity",
		observability.F("backup_path", backupPath),
	)

	// Verify files exist and checksums match
	for _, file := range metadata.Files {
		filePath := filepath.Join(backupPath, file.RelativePath)

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("backup file missing: %s", file.RelativePath)
		}

		// Verify checksum if available
		if file.Checksum != "" {
			actualChecksum, err := bm.calculateFileChecksum(filePath)
			if err != nil {
				return fmt.Errorf("failed to calculate checksum for %s: %w", file.RelativePath, err)
			}

			if actualChecksum != file.Checksum {
				return fmt.Errorf("checksum mismatch for %s", file.RelativePath)
			}
		}
	}

	// Verify overall backup checksum if available
	if metadata.Checksum != "" {
		actualChecksum, err := bm.calculateBackupChecksum(backupPath)
		if err != nil {
			return fmt.Errorf("failed to calculate backup checksum: %w", err)
		}

		if actualChecksum != metadata.Checksum {
			return fmt.Errorf("backup checksum mismatch")
		}
	}

	bm.logger.Debug(ctx, "backup_integrity_verified")
	return nil
}

// saveMetadata saves backup metadata to file
func (bm *BackupManager) saveMetadata(metadata *BackupMetadata, filePath string) error {
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}

// loadMetadata loads backup metadata from file
func (bm *BackupManager) loadMetadata(filePath string) (*BackupMetadata, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var metadata BackupMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, err
	}

	return &metadata, nil
}

// SetRetentionPeriod sets the backup retention period
func (bm *BackupManager) SetRetentionPeriod(period time.Duration) {
	bm.mutex.Lock()
	defer bm.mutex.Unlock()

	bm.retention = period
}
