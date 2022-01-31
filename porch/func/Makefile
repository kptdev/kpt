COMPILED_PROTO=evaluator/evaluator_grpc.pb.go evaluator/evaluator.pb.go

all: $(COMPILED_PROTO)

$(COMPILED_PROTO): evaluator/evaluator.proto
	protoc \
	  -I /usr/local/include/google/protobuf \
	  -I ./evaluator \
	  --go_out=./evaluator --go_opt=paths=source_relative \
	  --go-grpc_out=./evaluator --go-grpc_opt=paths=source_relative \
	  ./evaluator/evaluator.proto 
