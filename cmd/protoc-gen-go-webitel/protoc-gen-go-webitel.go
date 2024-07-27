package main

import (
	"flag"
	"fmt"

	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/webitel/webitel-go-kit/cmd/protoc-gen-go-webitel/internal"
)

const version = "1.0.0"

func main() {
	showVersion := flag.Bool("version", false, "print the version and exit")
	flag.Parse()
	if *showVersion {
		fmt.Printf("protoc-gen-go-webitel %v\n", version)
		return
	}

	protogen.Options{}.Run(run(version))
}

func run(version string) func(gen *protogen.Plugin) error {
	return func(gen *protogen.Plugin) error {
		gen.SupportedFeatures = uint64(pluginpb.CodeGeneratorResponse_FEATURE_PROTO3_OPTIONAL)
		var files []*protogen.File
		for _, f := range gen.Files {
			if f.Generate {
				files = append(files, f)
			}
		}
		if err := internal.Generate(gen, files, version); err != nil {
			return err
		}

		return nil
	}
}
