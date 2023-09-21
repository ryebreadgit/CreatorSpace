package general

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func FileExists(filepath string) error {
	if _, err := os.Stat(filepath); err == nil {
		return nil
	} else {
		return err
	}
}

// SanitizeFileName to take a full file path and make it valid for Windows and Linux

func SanitizeFilePath(fp string) (string, error) {
	invalidChars := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
	invalidFileName := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	maxLength := 255

	// Get the file name without the extension
	fileExt := filepath.Ext(fp)
	fileName := filepath.Base(fp)
	fileName = strings.TrimSuffix(fileName, fileExt)

	var safeFileName string

	// Check if the file name is invalid
	for _, invalidName := range invalidFileName {
		if strings.EqualFold(fileName, invalidName) {
			safeFileName = "File"
			break
		}
	}

	// If the file name is valid, check if it contains any invalid characters
	if safeFileName == "" {
		for _, invalidChar := range invalidChars {
			fileName = strings.ReplaceAll(fileName, invalidChar, "")
		}
		safeFileName = fileName
	}

	// If the file name is still empty, set it to "File"
	if safeFileName == "" {
		safeFileName = "File"
	}

	var safeFilePath string

	safeFilePath = fmt.Sprintf("%v/%v%v", filepath.Dir(fp), safeFileName, fileExt)

	// If the path is too long, set the file name to the id
	if len(safeFilePath) > maxLength {
		// Get the id from the file path between ( and ) at the end of the file name
		id := fileName[strings.LastIndex(fileName, "(")+1:]
		id = id[:strings.LastIndex(id, ")")]

		// Get all extensions after ) in filepath
		tempExt := fp[strings.LastIndex(fp, ")")+1:]
		if len(tempExt) != 0 && tempExt[0] == '.' {
			fileExt = tempExt
		}

		safeFilePath = fmt.Sprintf("%v/%v%v", filepath.Dir(fp), id, fileExt)
	}

	safeFilePath = filepath.Clean(safeFilePath)
	// remove double slashes and backslashes
	safeFilePath = strings.ReplaceAll(safeFilePath, "//", "/")
	safeFilePath = strings.ReplaceAll(safeFilePath, "\\", "/")

	return safeFilePath, nil
}

func SanitizeFileName(fileName string) (string, error) {
	invalidChars := []string{"\\", "/", ":", "*", "?", "\"", "<", ">", "|"}
	invalidFileName := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}
	maxLength := 255

	var safeFileName string

	// Check if the file name is invalid
	for _, invalidName := range invalidFileName {
		if strings.EqualFold(fileName, invalidName) {
			safeFileName = "File"
			break
		}
	}

	// If the file name is valid, check if it contains any invalid characters
	if safeFileName == "" {
		for _, invalidChar := range invalidChars {
			fileName = strings.ReplaceAll(fileName, invalidChar, "")
		}
		safeFileName = fileName
	}

	// If the file name is still empty, set it to "File"
	if safeFileName == "" {
		safeFileName = "File"
	}

	// If the path is too long, trim the file name to fit. If this still exceeds the max length, return an error
	if len(safeFileName) > maxLength {
		safeFileName = safeFileName[:maxLength]
	}

	// Strip leading and trailing whitespace
	safeFileName = strings.TrimSpace(safeFileName)

	return safeFileName, nil
}
