# Generate GRPC GO proto pb files 

protoc ./proto/*.proto --go_out=plugins=grpc:./src