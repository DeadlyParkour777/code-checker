gen-auth:
	@protoc -I=../proto \
		--go_out=auth --go_opt=paths=source_relative \
		--go-grpc_out=auth --go-grpc_opt=paths=source_relative \
		../proto/auth.proto

gen-result:
	@protoc -I=../proto \
		--go_out=result --go_opt=paths=source_relative \
		--go-grpc_out=result --go-grpc_opt=paths=source_relative \
		../proto/result.proto

gen-submission:
	@protoc -I=../proto \
	--go_out=submission --go_opt=paths=source_relative \
	--go-grpc_out=submission --go-grpc_opt=paths=source_relative \
	../proto/submission.proto

gen-problem:
	@protoc -I=../proto \
	--go_out=problem --go_opt=paths=source_relative \
	--go-grpc_out=problem --go-grpc_opt=paths=source_relative \
	../proto/problem.proto

