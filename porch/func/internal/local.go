package internal

import (
	"bytes"
	"context"

	"github.com/GoogleContainerTools/kpt-functions-catalog/functions/go/apply-setters/applysetters"
	pb "github.com/GoogleContainerTools/kpt/porch/func/evaluator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/kio"
)

var localfunctions map[string]framework.ResourceListProcessorFunc = map[string]framework.ResourceListProcessorFunc{
	"gcr.io/kpt-fn/apply-setters:v0.2.0": applySetters,
}

func fastEval(ctx context.Context, req *pb.EvaluateFunctionRequest) (bool, *pb.EvaluateFunctionResponse, error) {
	fn, fast := localfunctions[req.Image]
	if !fast {
		return false, nil, nil
	}

	resp, err := fastfn(ctx, fn, req.ResourceList)
	return true, resp, err
}

func fastfn(ctx context.Context, fn framework.ResourceListProcessorFunc, input []byte) (*pb.EvaluateFunctionResponse, error) {
	var out bytes.Buffer
	rw := &kio.ByteReadWriter{
		Reader:                bytes.NewReader(input),
		Writer:                &out,
		KeepReaderAnnotations: true,
	}
	if err := framework.Execute(fn, rw); err != nil {
		return nil, status.Errorf(codes.Internal, "Failed to execute function %")
	}

	return &pb.EvaluateFunctionResponse{
		ResourceList: out.Bytes(),
	}, nil
}

func applySetters(rl *framework.ResourceList) error {
	a := applysetters.ApplySettersProcessor{}
	return a.Process(rl)
}
