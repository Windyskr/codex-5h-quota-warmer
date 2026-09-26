package main

import (
	"archive/zip"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func main() {
	if err := packageRelease(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func packageRelease() error {
	input := flag.String("input", "", "dynamic library to package")
	output := flag.String("output", "", "platform ZIP file to create")
	flag.Parse()
	if *input == "" || *output == "" {
		return fmt.Errorf("input and output are required")
	}
	if errMkdir := os.MkdirAll(filepath.Dir(*output), 0o755); errMkdir != nil {
		return fmt.Errorf("create package directory: %w", errMkdir)
	}
	library, errOpen := os.Open(*input)
	if errOpen != nil {
		return fmt.Errorf("open dynamic library: %w", errOpen)
	}
	info, errStat := library.Stat()
	if errStat != nil {
		_ = library.Close()
		return fmt.Errorf("stat dynamic library: %w", errStat)
	}
	archive, errCreate := os.Create(*output)
	if errCreate != nil {
		_ = library.Close()
		return fmt.Errorf("create package: %w", errCreate)
	}
	zipWriter := zip.NewWriter(archive)
	header, errHeader := zip.FileInfoHeader(info)
	if errHeader == nil {
		header.Name = filepath.Base(*input)
		header.Method = zip.Deflate
		var writer io.Writer
		writer, errHeader = zipWriter.CreateHeader(header)
		if errHeader == nil {
			_, errHeader = io.Copy(writer, library)
		}
	}
	errZip := zipWriter.Close()
	errArchive := archive.Close()
	errLibrary := library.Close()
	for _, result := range []struct {
		name string
		err  error
	}{
		{"write package", errHeader},
		{"close ZIP", errZip},
		{"close package", errArchive},
		{"close dynamic library", errLibrary},
	} {
		if result.err != nil {
			return fmt.Errorf("%s: %w", result.name, result.err)
		}
	}
	data, errRead := os.ReadFile(*output)
	if errRead != nil {
		return fmt.Errorf("read package for checksum: %w", errRead)
	}
	checksum := sha256.Sum256(data)
	line := fmt.Sprintf("%x  %s\n", checksum, filepath.Base(*output))
	if errWrite := os.WriteFile(*output+".sha256", []byte(line), 0o644); errWrite != nil {
		return fmt.Errorf("write package checksum: %w", errWrite)
	}
	return nil
}
