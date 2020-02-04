module umenosuke.net/ping-grpc

go 1.13

replace (
	umenosuke.net/labelinglog => ../labelinglog
	umenosuke.net/pinger4 => ../pinger4
)

require (
	github.com/golang/protobuf v1.3.2
	google.golang.org/grpc v1.26.0
	umenosuke.net/labelinglog v0.0.0-00010101000000-000000000000
	umenosuke.net/pinger4 v0.0.0-00010101000000-000000000000
)
