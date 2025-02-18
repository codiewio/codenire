package internal

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func DownloadTemplates(tmpDir, s3Endpoint, s3Bucket, s3Prefix, region string) (*string, error) {
	s3Svc := s3.New(s3.Options{
		Region:       region,
		BaseEndpoint: aws.String(s3Endpoint),
	})

	list, err := s3Svc.ListObjectsV2(context.Background(), &s3.ListObjectsV2Input{
		Bucket: aws.String(s3Bucket),
		Prefix: aws.String(s3Prefix),
	})

	if err != nil {
		log.Println("Err", err)
		return nil, err
	}

	files := make(map[string]io.ReadCloser)
	for _, content := range list.Contents {
		if content.Key == nil {
			continue
		}

		path := *content.Key
		if !isFile(path) {
			continue
		}

		getObjectOutput, err2 := s3Svc.GetObject(context.Background(), &s3.GetObjectInput{
			Bucket: aws.String(s3Bucket),
			Key:    aws.String(path),
		})

		if err2 != nil {
			continue
		}
		defer func() {
			_ = getObjectOutput.Body.Close()
		}()

		prefix := s3Prefix
		if prefix != "" && !strings.HasSuffix(prefix, "/") {
			prefix += "/"
		}

		file := strings.TrimPrefix(path, prefix)
		files[file] = getObjectOutput.Body
	}

	err = copyFilesToTmpDir(tmpDir, files)
	if err != nil {
		log.Println("Err", err)
		return nil, err
	}

	return &tmpDir, nil
}

func copyFilesToTmpDir(tmpDir string, files map[string]io.ReadCloser) error {
	for path, src := range files {
		in := filepath.Join(tmpDir, path)
		if strings.Contains(path, "/") {
			if err := os.MkdirAll(filepath.Dir(in), 0755); err != nil {
				return err
			}
		}

		dst, err := os.Create(in)
		if err != nil {
			return fmt.Errorf("error creating temp file %q: %w", in, err)
		}
		defer func() {
			_ = dst.Close()
		}()

		_, err = io.Copy(dst, src)
		if err != nil {
			return fmt.Errorf("error copying data to file %q: %w", in, err)
		}
	}

	return nil
}

func isFile(path string) bool {
	return !strings.HasSuffix(path, "/")
}
