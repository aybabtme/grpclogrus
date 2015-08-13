/*
Package grpclogrus tries to parse grpc-go logs and emit logrus structure logs.

This is sort of a hack to alleviate https://github.com/grpc/grpc-go/issues/289.

The parsing rules should be valid for grpc-go checked out
at commit 91c8b79535eb6045d70ec671d302213f88a3ab95.
*/
package grpclogrus

import (
	"fmt"

	"github.com/Sirupsen/logrus"

	"google.golang.org/grpc/grpclog"
)

type log struct {
	l *logrus.Entry
}

// New makes a grpclog.Logger from a logrus.Entry.
func New(l *logrus.Entry) grpclog.Logger {
	if l == nil {
		l = logrus.WithFields(logrus.Fields{"source": "grpc"})
	}
	return &log{l: l}
}

// Inject a logrus logger in grpclog.
func Inject(l *logrus.Entry) {
	grpclog.SetLogger(New(l))
}

func (l *log) Fatal(args ...interface{})                 { l.fatal(l.tryParseln(args...)) }
func (l *log) Fatalf(format string, args ...interface{}) { l.fatal(l.tryParseF(format, args...)) }
func (l *log) Fatalln(args ...interface{})               { l.fatal(l.tryParseln(args...)) }
func (l *log) Print(args ...interface{})                 { l.print(l.tryParseln(args...)) }
func (l *log) Printf(format string, args ...interface{}) { l.print(l.tryParseF(format, args...)) }
func (l *log) Println(args ...interface{})               { l.print(l.tryParseln(args...)) }

func (l *log) fatal(fields logrus.Fields, message string) {
	l.l.WithFields(fields).Fatal(message)
}

func (l *log) print(fields logrus.Fields, message string) {
	l.l.WithFields(fields).Info(message)
}

func (l *log) tryParseF(format string, args ...interface{}) (fields logrus.Fields, message string) {
	rule, ok := parsefRules[format]
	if !ok {
		return l.defaultParsef(format, args...)
	}
	defer func() {
		if recover() != nil {
			fields, message = l.defaultParsef(format, args...)
		}
	}()
	fields, message = rule(args)
	return fields, message
}

func (l *log) tryParseln(args ...interface{}) (fields logrus.Fields, message string) {
	if len(args) < 1 {
		return logrus.Fields{}, ""
	}
	format := fmt.Sprint(args[0])
	args = args[1:]
	defer func() {
		if recover() != nil {
			fields, message = l.defaultParsef(format, args...)
		}
	}()
	rule, ok := parselnRules[format]
	if !ok {
		return l.defaultParsef(format, args...)
	}
	fields, message = rule(args)
	return fields, message
}

func (l *log) defaultParsef(format string, args ...interface{}) (logrus.Fields, string) {
	fields := logrus.Fields{}
	for i, arg := range args {
		fields[fmt.Sprintf("arg%d", i)] = fmt.Sprintf("%v", arg)
	}
	return fields, format
}

var parsefRules = map[string]func(args ...interface{}) (logrus.Fields, string){
	"%v compleled with error code %d, want %d": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"stream": args[0], "grpc.Code(err)": args[1], "codes.Canceled": args[2]}, "completed with wrong error code"
	},
	"%v.CloseAndRecv() got error code %d, want %d": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"stream": args[0], "want.code": args[1], "got.code": args[2]}, "stream CloseAndRecv() got wrong error code"
	},
	"Getting feature for point (%d, %d)": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"point.latitude": args[0], "point.longitude": args[1]}, "Getting feature for point"
	},
	"Got %d reply, want %d": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"want.count": args[0], "got.count": args[1]}, "got wrong count of replies"
	},
	"Got message %s at point(%d, %d)": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"message": args[0], "point.latitude": args[1], "point.longitude": args[2]}, "got message at point"
	},
	"Got reply body of length %d, want %d": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"want.length": args[0], "got.length": args[1]}, "Got reply body of wrong length"
	},
	"Got the reply of type %d, want %d": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"got.type": args[0], "want.type": args[1]}, "Got the reply of wrong type"
	},
	"Got the reply with type %d len %d; want %d, %d": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"got.type": args[0], "got.len": args[1], "want.type": args[2], "want.len": args[3]}, "Got the reply with wrong type and length"
	},
	"Requested a response with invalid length %d": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"length": args[0]}, "Requested a response with invalid length"
	},
	"Sent a request of size %d, aggregated size %d": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"request.size": args[0], "aggregated.size": args[1]}, "Sent a request of wrong size"
	},
	"Traversing %d points.": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"count": args[0]}, "traversing points"
	},
	"Unsupported payload type: %d": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"type": args[0]}, "unsupported payload type"
	},
	"%v failed to complele the ping pong test: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"stream": args[0], "err": args[1]}, "failed to complele the ping pong test"
	},
	"%v.CloseAndRecv() got error %v, want %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"stream": args[0], "err": args[1]}, "stream CloseAndRecv() got error, expected none"
	},
	"%v.CloseSend() got %v, want %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"stream": args[0], "err": args[1]}, "stream CloseSend() got error, expected none"
	},
	"%v.CloseAndRecv().GetAggregatePayloadSize() = %v; want %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"stream": args[0], "reply.GetAggregatedPayloadSize()": args[1], "sum": args[2]}, "stream CloseAndRecv().GetAggregatePayloadSize() got wrong size"
	},
	"%v.GetFeatures(_) = _, %v: ": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"client": args[0], "err": args[1]}, "GetFeatures"
	},
	"%v.ListFeatures(_) = _, %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"client": args[0], "err": args[1]}, "ListFeatures"
	},
	"%v.RecordRoute(_) = _, %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"client": args[0], "err": args[1]}, "RecordRoute"
	},
	"%v.RouteChat(_) = _, %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"client": args[0], "err": args[1]}, "RouteChat"
	},
	"%v.FullDuplexCall(_) = _, %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"tc": args[0], "err": args[1]}, "FullDuplexCall"
	},
	"%v.Recv() = %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"stream": args[0], "err": args[1]}, "stream .Recv() got error"
	},
	"%v.Send(%v) = %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"stream": args[0], "point": args[1], "err": args[2]}, "stream .Send() got error"
	},
	"Got OAuth scope %q which is NOT a substring of %q.": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"got.scope": args[0], "want.scope": args[1]}, "Got OAuth scope which is NOT a substring of expected scope"
	},
	"Got user name %q which is NOT a substring of %q.": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"user": args[0], "json.key": args[1]}, "Got user name which is NOT a substring json key"
	},
	"Got user name %q, want %q.": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"got.user": args[0], "want.user": args[1]}, "wrong user name"
	},
	"grpc: ClientConn.resetTransport failed to create client transport: %v; Reconnecting to %q": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "grpc", "err": args[0], "cc.target": args[1]}, "ClientConn.resetTransport failed to create client transport, reconnecting"
	},
	"grpc: Server.RegisterService found duplicate service registration for %q": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "grpc", "service.name": args[0]}, "Server.RegisterService found duplicate service registration"
	},
	"NewClientConn(%q) failed to create a ClientConn %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"addr": args[0], "err": args[1]}, "NewClientConn(_) failed to create a ClientConn"
	},
	"transport: http2Server.HandleStreams received bogus greeting from client: %q": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "transport", "preface": args[0]}, "http2Server.HandleStreams received bogus greeting from client"
	},
	"%v.SendHeader(%v) = %v, want %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"stream": args[0], "md": args[1], "err": args[2], "nil": args[3]}, "SendHeader"
	},
	"%v.StreamingCall(_) = _, %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"tc": args[0], "err": args[1]}, "StreamingCall"
	},
	"%v.StreamingInputCall(_) = _, %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"tc": args[0], "err": args[1]}, "StreamingInputCall"
	},
	"%v.StreamingOutputCall(_) = _, %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"tc": args[0], "err": args[1]}, "StreamingOutputCall"
	},
	"/TestService/EmptyCall receives %v, want %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"reply": args[0], "testpb.Empty{}": args[1]}, "/TestService/EmptyCall receives"
	},
	"Dial(%q) = %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"addr, err": args[0]}, "Dial"
	},
	"Fail to dial: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "fail to dial"
	},
	"fail to dial: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "fail to dial"
	},
	"Failed to convert %v to *http2Server": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"s.ServerTransport()": args[0]}, "Failed to convert to *http2Server"
	},
	"Failed to create credentials %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "Failed to create credentials"
	},
	"Failed to create JWT credentials: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "Failed to create JWT credentials"
	},
	"Failed to create TLS credentials %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "Failed to create TLS credentials"
	},
	"Failed to decode (%q, %q): %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"f.Name": args[0], "f.Value": args[1], "err": args[2]}, "Failed to decode"
	},
	"Failed to dial %s: %v; please retry.": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"target, err": args[0]}, "Failed to dial, please retry"
	},
	"Failed to finish the server streaming rpc: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "Failed to finish the server streaming rpc"
	},
	"Failed to generate credentials %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "Failed to generate credentials"
	},
	"Failed to listen: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "failed to listen"
	},
	"failed to listen: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "failed to listen"
	},
	"Failed to load default features: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "Failed to load default features"
	},
	"Failed to parse listener address: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "Failed to parse listener address"
	},
	"failed to parse listener address: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "Failed to parse listener address"
	},
	"Failed to read the service account key file: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "Failed to read the service account key file"
	},
	"Failed to receive a note : %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "Failed to receive a note"
	},
	"Failed to send a note: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "Failed to send a not"
	},
	"Failed to serve: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "Failed to serve"
	},
	"grpc.SendHeader(%v, %v) = %v, want %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"ctx": args[0], "md": args[1], "err": args[2]}, "grpc.SendHeader"
	},
	"transport: http2Server.HandleStreams saw invalid preface type %T from client": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "transport", "frame": fmt.Sprintf("%T", args[0])}, "http2Server.HandleStreams saw invalid preface type from client"
	},
	"grpc: ClientConn.transportMonitor exits due to: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "grpc", "err": args[0]}, "ClientConn.transportMonitor exits"
	},
	"grpc: SendHeader: %v has no ServerTransport to send header metadata.": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "grpc", "stream": args[0]}, "SendHeader: stream has no ServerTransport to send header metadata"
	},
	"grpc: Server failed to encode response %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "grpc", "err": args[0]}, "Server failed to encode response"
	},
	"grpc: Server.handleStream failed to write status: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "grpc", "err": args[0]}, "Server.handleStream failed to write status"
	},
	"grpc: Server.processUnaryRPC failed to write status: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "grpc", "err": args[0]}, "Server.processUnaryRPC failed to write status"
	},
	"grpc: Server.RegisterService found the handler of type %v that does not satisfy %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "grpc", "found.type": args[0], "expected.type": args[1]}, "Server.RegisterService found handler of type that does not satisfy expectations"
	},
	"handleStream got error: %v, want <nil>; result: %v, want %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0], "p": args[1], "req": args[2]}, "handleStream got error"
	},
	"Looking for features within %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"rect": args[0]}, "Looking for features withi rectangle"
	},
	"PayloadType UNCOMPRESSABLE is not supported": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{}, "PayloadType UNCOMPRESSABLE is not supported"
	},
	"Route summary: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"reply": args[0]}, "Route summary"
	},
	"StreamingCall(_).Recv: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "StreamingCall(_).Recv"
	},
	"StreamingCall(_).Send: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "StreamingCall(_).Send"
	},
	"TLS is not enabled. TLS is required to execute compute_engine_creds test case.": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{}, "TLS is not enabled. TLS is required to execute compute_engine_creds test case"
	},
	"TLS is not enabled. TLS is required to execute service_account_creds test case.": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{}, "TLS is not enabled. TLS is required to execute service_account_creds test case"
	},
	"transport: http2Client.controller got unexpected item type %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "transport", "item.type": args[0]}, "http2Client.controller got unexpected item type"
	},
	"transport: http2Client.notifyError got notified that the client transport was broken %v.": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "transport", "err": args[0]}, "http2Client.notifyError got notified that the client transport was broken"
	},
	"transport: http2Client.reader got unhandled frame type %v.": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "transport", "frame": args[0]}, "http2Client.reader got unhandled frame type"
	},
	"transport: http2Server %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "transport", "err": args[0]}, "http2Server error"
	},
	"transport: http2Server.controller got unexpected item type %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "transport", "item.type": args[0]}, "http2Server.controller got unexpected item type"
	},
	"transport: http2Server.HandleStreams failed to read frame: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "transport", "err": args[0]}, "http2Server.HandleStreams failed to read frame"
	},
	"transport: http2Server.HandleStreams failed to receive the preface from client: %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "transport", "err": args[0]}, "http2Server.HandleStreams failed to receive the preface from client"
	},
	"transport: http2Server.HandleStreams found unhandled frame type %v.": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "transport", "frame": args[0]}, "http2Server.HandleStreams found unhandled frame type"
	},
	"transport: http2Server.operateHeader found %v": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "transport", "err": args[0]}, "http2Server.operateHeader found"
	},
}

var parselnRules = map[string]func(args ...interface{}) (logrus.Fields, string){
	"/TestService/EmptyCall RPC failed: ": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "/TestService/EmptyCall RPC failed"
	},
	"/TestService/UnaryCall RPC failed: ": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"err": args[0]}, "/TestService/UnaryCall RPC failed"
	},
	"CancelAfterBegin done": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{}, "CancelAfterBegin done"
	},
	"CancelAfterFirstResponse done": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{}, "CancelAfterFirstResponse done"
	},
	"Client profiling address: ": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"addr": args[0]}, "Client profiling address"
	},
	"ClientStreaming done": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{}, "ClientStreaming done"
	},
	"ComputeEngineCreds done": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{}, "ComputeEngineCreds done"
	},
	"EmptyUnaryCall done": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{}, "EmptyUnaryCall done"
	},
	"grpc: Server.Serve failed to complete security handshake.": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "grpc"}, "Server.Serve failed to complete security handshake"
	},
	"grpc: Server.Serve failed to create ServerTransport: ": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "grpc", "err": args[0]}, "Server.Serve failed to create ServerTransport"
	},
	"LargeUnaryCall done": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{}, "LargeUnaryCall done"
	},
	"Pingpong done": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{}, "Pingpong done"
	},
	"Server Address: ": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"addr": args[0]}, "Server Address"
	},
	"Server profiling address: ": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"addr": args[0]}, "Server profiling address"
	},
	"ServerStreaming done": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{}, "ServerStreaming done"
	},
	"ServiceAccountCreds done": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{}, "ServiceAccountCreds done"
	},
	"transport: http2Client.handleRSTStream found no mapped gRPC status for the received http2 error ": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "transport", "err": args[0]}, "http2Client.handleRSTStream found no mapped gRPC status for the received http2 error"
	},
	"transport: http2Server.HandleStreams received an illegal stream id: ": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"package": "transport", "id": args[0]}, "http2Server.HandleStreams received an illegal stream id"
	},
	"Unsupported test case: ": func(args ...interface{}) (logrus.Fields, string) {
		return logrus.Fields{"test.case": args[0]}, "Unsupported test case"
	},
}
