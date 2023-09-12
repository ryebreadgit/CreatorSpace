package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func parseRangeHeader(rangeHeader string, fileSize int64, maxChunkSize int64) (int64, int64, error) {
	var start int64
	var end int64
	var err error

	rangeHeaderValue := strings.Split(rangeHeader, "=")
	if len(rangeHeaderValue) != 2 {
		return 0, fileSize - 1, nil
	}
	rangeValue := strings.Split(rangeHeaderValue[1], "-")

	if len(rangeValue) != 2 {
		return 0, 0, errors.New("invalid range header")
	}

	if rangeValue[0] != "" {
		start, err = strconv.ParseInt(rangeValue[0], 10, 64)
		if err != nil {
			return 0, 0, err
		}
	} else {
		start = 0
	}

	if rangeValue[1] != "" {
		end, err = strconv.ParseInt(rangeValue[1], 10, 64)
		if err != nil {
			end = start + maxChunkSize
		}
	} else {
		end = start + maxChunkSize
	}

	if end >= fileSize {
		end = start + maxChunkSize
	}

	if start > end || start < 0 {
		return 0, 0, errors.New("invalid range header")
	}

	return start, end, nil
}

func wrapper(f func(c *gin.Context) (string, error)) gin.HandlerFunc {

	return func(c *gin.Context) {
		data, err := f(c)
		if err != nil {
			c.JSON(503, gin.H{"ret": 503, "err": err.Error()})
			return
		}

		var result map[string]interface{}
		err = json.Unmarshal([]byte(data), &result)
		if err != nil {
			c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(data))
			return
		}

		c.JSON(http.StatusOK, gin.H{"status": http.StatusOK, "data": result})
	}
}

func StreamDirect(c *gin.Context, filePath string, mimeType string) error {
	// Get file info
	file, err := os.Open(filePath)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	fileSize := fileInfo.Size()

	// Set the maximum chunk size to 10MB
	const maxChunkSize int64 = 5 * 1024 * 1024

	// Parse the range header
	rangeHeader := c.Request.Header.Get("Range")
	start, end, err := parseRangeHeader(rangeHeader, fileSize, maxChunkSize)
	if err != nil {
		c.Status(http.StatusBadRequest)
		return err
	}

	// if asking for whole file, return 204 no content status
	if start == 0 && end == fileSize-1 {
		c.Status(http.StatusNoContent)
		return nil
	}

	c.Status(http.StatusPartialContent)
	c.Header("Content-Type", mimeType)
	c.Header("Content-Disposition", "inline")

	// Calculate the size of the chunk to be served
	chunkSize := end - start + 1
	if chunkSize > maxChunkSize {
		chunkSize = maxChunkSize
		end = start + maxChunkSize - 1
	}

	// Set the content-range header
	c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, fileSize))

	// Seek to the start position in the file
	_, err = file.Seek(start, io.SeekStart)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return err
	}

	// Serve the chunk
	bytesCopied, err := io.CopyN(c.Writer, file, chunkSize)
	if err != nil && err != io.EOF {
		c.Status(http.StatusInternalServerError)
		return err
	}

	// If the last chunk was smaller than the maximum chunk size,
	// adjust the content-range header to reflect the actual size
	if bytesCopied < chunkSize {
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, start+bytesCopied-1, fileSize))
	}

	// Serve subsequent chunks until the end of the file is reached
	for end-bytesCopied >= start {
		// Calculate the size of the next chunk
		chunkSize = end - bytesCopied + 1
		if chunkSize > maxChunkSize {
			chunkSize = maxChunkSize
			end = bytesCopied + maxChunkSize - 1
		}

		// Set the content-range header for the next chunk
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", bytesCopied, end, fileSize))

		// Serve the next chunk
		_, err = io.CopyN(c.Writer, file, chunkSize)
		if err != nil && err != io.EOF {
			c.Status(http.StatusInternalServerError)
			return err
		}

		bytesCopied += chunkSize
	}
	return nil
}
