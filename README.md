# Calling Go from Python via gRPC

This repo is an example to show how to make remote procedure calls via
gRPC between Python and Go. TLS (multi) is used to secure the network
communication.

Read [this article](TODO) for a complete walk-through.

## Quickstart

+ install `direnv` as a shell extension
+ run `direnv allow`
+ install `Python` and `Go` (1.18)
+ run `make setup`
+ open two terminal windows and run respectively:
  - `make server`
  - `make client`
+ run `make stop` to shutdown the server
+ run `make clean` to remove the installed binaries

### The example

Basic explanation. Imagine you have a web mapping platform like Google
Maps, you may have some satellites scanning the Earth and storing the
data on a server, and clients performing some queries to the server to
get the view of a location. For simplicity, assume you already have a
80x32 (width, height) 2D map of the Earth stored on the server. The
goal is to support queries to get the image associated with the `xy`
coordinate of a location, and also queries to get the view associated
with a broader rectangular area delimited by the `xy` coordinates
associated with its bottom-left and top-right corners. Also, you may
want to encrypt the communication between client and server to counter
eavesdropping and ensure no third-parties are able to perform MITM
attacks (i.e., like replacing the image associated with a coordinate).
