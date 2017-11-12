build: assets protoc sun

sun: pre-build
	go build -o 'bin/sun' -ldflags '-X github.com/damnever/sunflower/version.Build=$(shell date +%Y_%m_%d_`date +%s`)' ./cmd/sun

flower: pre-build # ignore it..
	go build -o 'bin/flower' ./cmd/flower

pre-build:
	mkdir -p bin

protoc:
	# go get github.com/gogo/protobuf/protoc-gen-gogoslick
	protoc --proto_path=$(shell go list -e -f '{{.Dir}}' github.com/damnever/sunflower)/vendor --proto_path=. --gogoslick_out=. msg/msgpb/msg.proto

assets: npm-build
	importPath=$$(go list -e -f "{{.ImportPath}}"); \
	deps=$$(cd cmd/flower && go list -f "{{.Deps}}" | tr -d "[]"); \
	fileredDeps="cmd/flower"; \
	for dep in $$deps; do \
			if [[ $$dep != $${importPath}/* ]]; then continue; fi; \
			dep=$${dep#$$importPath/}; \
			fileredDeps="$${fileredDeps} $${dep}"; \
	done; \
	zip -r sun/fe/flower.zip cmd/flower $${fileredDeps} # the prefix, TODO: use cat >> bin && zip -A..
	go-bindata -o=./sun/web/assets.go -pkg=web -prefix=sun/fe/ ./sun/fe/index.html ./sun/fe/dist/... ./sun/fe/flower.zip
	rm -f sun/fe/flower.zip

npm-install:
	cd ./sun/fe; npm install --registry=https://registry.npm.taobao.org && npm shrinkwrap

npm-build:
	cd ./sun/fe; npm run build

npm-serve:
	cd ./sun/fe; npm run dev

clean:
	find . -type f -name '.DS_Store' -exec rm -f {} +
