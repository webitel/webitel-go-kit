package internal

import (
	"fmt"

	"google.golang.org/genproto/googleapis/api/annotations"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"

	pb "github.com/webitel/webitel-go-kit/cmd/protoc-gen-go-webitel/gen/go/proto/webitel"
)

type HttpBinding struct {
	Path   string
	Method string
}

func extractServiceObjClassOption(p *descriptorpb.ServiceDescriptorProto) (string, error) {
	ext, err := extractOption(p.Options, pb.E_Objclass)
	if err != nil {
		return "", err
	}

	obj, ok := ext.(string)
	if !ok {
		return "", fmt.Errorf("extension is %T; want string", ext)
	}

	return obj, nil
}

func extractServiceAdditionalLicenseOption(p *descriptorpb.ServiceDescriptorProto) ([]string, error) {
	ext, err := extractOption(p.Options, pb.E_Objclass)
	if err != nil {
		return nil, err
	}

	obj, ok := ext.([]string)
	if !ok {
		return nil, fmt.Errorf("extension is %T; want []string", ext)
	}

	return obj, nil
}

func extractMethodHttpOption(p *descriptorpb.MethodDescriptorProto) ([]*HttpBinding, error) {
	ext, err := extractOption(p.Options, annotations.E_Http)
	if err != nil {
		return nil, err
	}

	obj, ok := ext.(*annotations.HttpRule)
	if !ok {
		return nil, fmt.Errorf("extension is %T; want *annotations.HttpRule", ext)
	}

	bs := make([]*HttpBinding, 0)
	b, err := newHttpBinding(obj, p.GetName())
	if err != nil {
		return nil, err
	}

	bs = append(bs, b)
	if len(obj.GetAdditionalBindings()) > 0 {
		for _, v := range obj.GetAdditionalBindings() {
			if len(v.AdditionalBindings) > 0 {
				return nil, fmt.Errorf("additional_binding in additional_binding not allowed: %s", p.GetName())
			}

			b, err := newHttpBinding(v, p.GetName())
			if err != nil {
				return nil, err
			}

			bs = append(bs, b)
		}
	}

	return bs, nil
}

func extractMethodAccessOption(p *descriptorpb.MethodDescriptorProto) (int, error) {
	ext, err := extractOption(p.Options, pb.E_Access)
	if err != nil {
		return 0, err
	}

	obj, ok := ext.(pb.Action)
	if !ok {
		return 0, fmt.Errorf("extension is %T; want int", ext)
	}

	return int(obj.Number()), nil
}

func extractOption(m proto.Message, xt protoreflect.ExtensionType) (any, error) {
	if m == nil {
		return nil, fmt.Errorf("message %T is nil", m)
	}

	if !proto.HasExtension(m, xt) { // TODO
		// return nil, fmt.Errorf("message %s doesnt contain extension %s", m, xt.TypeDescriptor().Name())
	}

	return proto.GetExtension(m, xt), nil
}

func newHttpBinding(r *annotations.HttpRule, m string) (*HttpBinding, error) {
	var b HttpBinding
	switch {
	case r.GetGet() != "":
		b.Method = "GET"
		b.Path = r.GetGet()
		if r.Body != "" {
			return nil, fmt.Errorf("must not set request body when http method is GET: %s", m)
		}

	case r.GetPut() != "":
		b.Method = "PUT"
		b.Path = r.GetPut()

	case r.GetPost() != "":
		b.Method = "POST"
		b.Path = r.GetPost()

	case r.GetDelete() != "":
		b.Method = "DELETE"
		b.Path = r.GetDelete()
		if r.Body != "" {
			return nil, fmt.Errorf("must not set request body when http method is DELETE except allow_delete_body option is true: %s", m)
		}

	case r.GetPatch() != "":
		b.Method = "PATCH"
		b.Path = r.GetPatch()

	case r.GetCustom() != nil:
		custom := r.GetCustom()
		b.Method = custom.Kind
		b.Path = custom.Path

	default: // TODO
		// return nil, fmt.Errorf("no pattern specified in google.api.HttpRule: %s", m)
	}

	return &b, nil
}
