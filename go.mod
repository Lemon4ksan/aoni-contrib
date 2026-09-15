module github.com/lemon4ksan/aoni-x

go 1.27.0

require (
	github.com/lemon4ksan/aoni v0.0.0-00010101000000-000000000000
	github.com/lemon4ksan/foundation v0.0.0-20260912143841-ae415cdfd72d
	github.com/lemon4ksan/mach v0.0.0-00010101000000-000000000000
	github.com/oschwald/maxminddb-golang/v2 v2.6.0
	github.com/pkg/sftp v1.13.11
	golang.org/x/crypto v0.56.0
	golang.org/x/net v0.58.0
	golang.org/x/sys v0.47.0
	google.golang.org/protobuf v1.36.12
)

require (
	github.com/kr/fs v0.1.0 // indirect
	github.com/kr/text v0.2.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/lemon4ksan/aoni => ../aoni
	github.com/lemon4ksan/foundation => ../foundation
	github.com/lemon4ksan/mach => ../mach
)
