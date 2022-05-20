package main

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"log"
	"net"

	"satserver/example.com/satellitepb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	OutOfBoundFmt   string = "Out of range location received (x: %d, y: %d)"
	InvalidAreaFmt  string = "Invalid area received (%d;%d)->(%d;%d)\n"
	InvalidTokenFmt string = "Invalid token format\n"
	AuthFalureFmt   string = "Permission denied, invalid token\n"
	PermDeniedFmt   string = "No permission associated with token (%s)\n"
	height          int    = 32
	width           int    = 80
	sleepTime       int    = 0
)

type Server struct {
	satellitepb.UnimplementedSatelliteServer
	InfoLog              *log.Logger
	ErrorLog             *log.Logger
	sMap                 []string
	AuthenticationTokens map[string]bool
}

func NewServer() *Server {

	s := &Server{}
	s.InfoLog = log.New(os.Stdout, "INFO:\t", log.Lmicroseconds)
	s.ErrorLog = log.New(os.Stdout, "ERROR:\t", log.Lmicroseconds)
	s.sMap = make([]string, height)
	s.AuthenticationTokens = make(map[string]bool)
	// just an example of metadata
	s.AuthenticationTokens["03357-1"] = true

	return s
}

func (s *Server) LoadMap(fname string) {

	f, err := os.Open(fname)
	if err != nil {
		panic("Cant't open the map")
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	i := height - 1
	for scanner.Scan() && i >= 0 {
		s.sMap[i] = scanner.Text()
		i--
	}

}

func loadTLSCreds() (credentials.TransportCredentials, error) {

	caCert, err := ioutil.ReadFile("../certs/ca-cert.pem")
	if err != nil {
		return nil, err
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to add CA's certificate")
	}

	serverCert, err := tls.LoadX509KeyPair("../certs/server-cert.pem", "../certs/server-key.pem")
	if err != nil {
		return nil, err
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientAuth:   tls.RequireAndVerifyClientCert,
		ClientCAs:    certPool,
	}

	return credentials.NewTLS(cfg), nil
}

func (s *Server) CheckLocation(loc *satellitepb.Location) bool {

	if loc.X < 0 || loc.X > int32(width)-1 ||
		loc.Y < 0 || loc.Y > int32(height)-1 {

		s.ErrorLog.Print(fmt.Errorf(
			OutOfBoundFmt, loc.X, loc.Y))

		return false
	}

	return true
}

func (s *Server) CheckArea(locLl, locUr *satellitepb.Location) bool {

	if locLl.X > locUr.X ||
		locLl.Y > locUr.Y {

		s.ErrorLog.Printf(
			InvalidAreaFmt, locLl.X, locLl.Y, locUr.X, locUr.Y)

		return false
	}

	return true
}

func (s *Server) GetImages(area *satellitepb.Area, stream satellitepb.Satellite_GetImagesServer) error {

	s.InfoLog.Printf(
		"GetImages stream request (x: %d, y: %d) -> (x: %d, y: %d)\n",
		area.Ll.X, area.Ll.Y, area.Ur.X, area.Ur.Y)

	isValidCoordLl := s.CheckLocation(area.Ll)
	if !isValidCoordLl {
		return status.Errorf(codes.OutOfRange, OutOfBoundFmt, area.Ll.X, area.Ll.Y)
	}

	isValidCoordUr := s.CheckLocation(area.Ur)
	if !isValidCoordUr {
		return status.Errorf(codes.OutOfRange, OutOfBoundFmt, area.Ur.X, area.Ur.Y)
	}

	isValideArea := s.CheckArea(area.Ll, area.Ur)
	if !isValideArea {
		return status.Errorf(codes.OutOfRange, InvalidAreaFmt, area.Ll.X, area.Ll.Y, area.Ur.X, area.Ur.Y)
	}

	for j := int(area.Ur.Y - area.Ll.Y); j > 0; j-- {
		for i := 0; i < int(area.Ur.X-area.Ll.X); i++ {

			w := int(area.Ll.X) + i
			h := int(area.Ll.Y) + j

			img := &satellitepb.Image{
				Y:   int32(h),
				X:   int32(w),
				Img: []byte(s.sMap[h][w : w+1]),
			}

			if err := stream.Send(img); err != nil {
				return err
			}

			time.Sleep(time.Duration(sleepTime) * time.Millisecond)

		}
	}

	return nil
}

func (s *Server) GetImage(ctx context.Context, loc *satellitepb.Location) (*satellitepb.Image, error) {

	s.InfoLog.Printf("GetImage request (x: %d, y: %d)\n", loc.X, loc.Y)

	metadata, ok := metadata.FromIncomingContext(ctx)
	if ok {

		if token, found := metadata["token"]; found {
			if _, valid := s.AuthenticationTokens[token[0]]; !valid {
				s.ErrorLog.Printf("Unauthorized token: %v", token[0])
				return nil, status.Errorf(codes.PermissionDenied, PermDeniedFmt, token[0])
			}
			s.InfoLog.Printf("Client-provided token: %s", token[0])
		} else {
			s.ErrorLog.Printf("Request blocked, no valid token provided")
			return nil, status.Errorf(codes.PermissionDenied, AuthFalureFmt)
		}

	}

	isValidCoord := s.CheckLocation(loc)

	if !isValidCoord {
		return nil, status.Errorf(codes.OutOfRange,
			OutOfBoundFmt,
			loc.X,
			loc.Y)
	}

	h := loc.Y
	w := loc.X

	img := &satellitepb.Image{
		Y:   int32(h),
		X:   int32(w),
		Img: []byte{s.sMap[h][w+1]},
	}

	return img, nil
}

func main() {

	lis, err := net.Listen("tcp", "0.0.0.0:8000")
	if err != nil {
		log.Fatalf("Cannot listen to 0.0.0.0:8000 %v", err)
	}

	// get the sleep interval
	sleepTime, err = strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatalf("Cannot read sleep interval %v", err)
	}

	s := NewServer()
	s.LoadMap("map.txt")

	// load TLS credentials
	tlsCreds, err := loadTLSCreds()
	if err != nil {
		log.Fatalf("Error loading TLS credentials %v", err)
	}

	// init the gRPC server
	grpcServer := grpc.NewServer(grpc.Creds(tlsCreds))
	satellitepb.RegisterSatelliteServer(grpcServer, s)

	log.Println("Server starting...")
	log.Fatal(grpcServer.Serve(lis))
}
