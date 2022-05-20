.PHONY: protos venv setup upgrade certs run all clean client server stop

SHELL 			:= /bin/bash
MAKE			:= make --no-print-directory

space=$() $()

VENVNAME		:= $(subst P,p,$(subst $(space),-,$(shell python3 --version)))
VENV			:= $(PWD)/.direnv/$(VENVNAME)
VIRTUALENV		:= python3 -m venv
ACTIVATE		:= $(VENV)/bin/activate
PYTHON			:= $(VENV)/bin/python
PIP			:= $(PYTHON) -m pip
REQUIREMENTS		:= requirements.txt
BIN			:= binaries
GO			:= $(shell which go)
CERTS			:= certs
PORT			:= 8000
SLEEP_TIME		:= 0

all: | setup server client stop

venv: $(REQUIREMENTS)
	@ echo "[*] Ensure bin was created successfully"
	@ test -d $(BIN) || mkdir $(BIN)
	@ echo "[*] Creating virtual environment"
	@ test -d $(VENV) || $(VIRTUALENV) $(VENV)
	@ $(PIP) install --upgrade pip
	@ $(PIP) install -r $(REQUIREMENTS)
	@ touch $(ACTIVATE)

dependencies:
	@ echo "[*] Installing protoc"	
	@ cd $(BIN); \
	  wget https://github.com/protocolbuffers/protobuf/releases/download/v3.19.4/protoc-3.19.4-linux-x86_64.zip; \
	  unzip protoc-3.19.4-linux-x86_64.zip; \
	  mv bin/protoc .; \
	  rm -rf bin; \
	  rm protoc-3.19.4-linux-x86_64.zip
	@ echo "[*] Installing gRPC Go plugin"
	@ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1

protos:
	@ echo "[*] Generating Go gRPC definitions"
	@ protoc --go-grpc_out=server/ --go_out=server/ protos/satellite.proto
	@ echo "[*] Generating Python gRPC definitions"
	@ $(PYTHON) -m grpc_tools.protoc -I. --python_out=client --grpc_python_out=client protos/satellite.proto

certs:
# CA's private key & certificate
	@ openssl req -x509 -newkey rsa:4096 -days 30 -nodes -keyout $(CERTS)/ca-key.pem \
		-out $(CERTS)/ca-cert.pem -subj "/C=/ST=/L=/O=My CA/OU=/CN=/emailAddress="
# Server's private key + CSR
	@ openssl req -newkey rsa:4096 -nodes -keyout $(CERTS)/server-key.pem -out $(CERTS)/server-req.pem \
		-subj "/C=/ST=/L=/O=Server/OU=/CN=/emailAddress="
# Use the CA to approve the CSR and generate the (server) certificate
	@ openssl x509 -req -in $(CERTS)/server-req.pem -days 30 -CA $(CERTS)/ca-cert.pem \
		-CAkey $(CERTS)/ca-key.pem -CAcreateserial -out $(CERTS)/server-cert.pem \
		-extfile $(CERTS)/server-ext.cnf
# Client's private key + CSR
	@ openssl req -newkey rsa:4096 -nodes -keyout $(CERTS)/client-key.pem -out $(CERTS)/client-req.pem \
		-subj "/C=/ST=/L=/O=Client/OU=/CN=/emailAddress="
# Use the CA to approve the CSR and generate the (client) certificate
	@ openssl x509 -req -in $(CERTS)/client-req.pem -days 30 -CA $(CERTS)/ca-cert.pem \
		-CAkey $(CERTS)/ca-key.pem -CAcreateserial -out $(CERTS)/client-cert.pem \
		-extfile $(CERTS)/client-ext.cnf

setup: | venv dependencies protos certs

upgrade:
	@ echo "[*] Upgrade virtual environment"
	@ test -d $(VENV) || $(VIRTUALENV) $(VENV)
	@ $(PIP) install --upgrade pip
	@ $(PIP) install -r $(REQUIREMENTS)

activate:
	@ touch $(ACTIVATE)

stop:
	@ echo "[*] Stopping the server"
	@ fuser -kf $(PORT)/tcp

server: activate
	@ echo "[*] Starting the server"
	@ cd server; $(GO) run server.go $(SLEEP_TIME) &
	@ echo "[*] Wait untile the server is ready..."
	@ sleep 2
	@ echo "[*] ...server ready"

client:
	@ echo "[*] Starting the client"
	@ $(PYTHON) client/app.py

clean:
	@ echo "[*] Removing binaries"
	@ rm -rf $(BIN)
	@ rm -rf .PID
	@ echo "[*] Removing python virtual environment"
	@ rm -rf $(VENV)
	@ rm -rf *.egg-info/
	@ rm -rf __pycache__
	@ rm -rf client/__pycache__
	@ echo "[*] Removing gRPC auto-generated files"
	@ rm -rf server/example.com
	@ rm -rf client/protos
	@ find . -iname '*~' -exec rm {} \;
	@ echo "[*] Removing certificates and keys"
	@ rm -rf certs/*.pem
	@ rm -rf certs/*.srl
