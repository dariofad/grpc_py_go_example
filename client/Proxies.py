import grpc
import protos.satellite_pb2
import protos.satellite_pb2_grpc

import asyncio

class SimpleProxy:
    

    def __init__(self, server):

        self.server = server
        self.ca_cert, self.client_cert, self.client_key = self._readCertificates()
        self.grpc_credentials = grpc.ssl_channel_credentials(
            root_certificates=self.ca_cert,
            private_key=self.client_key,
            certificate_chain=self.client_cert)

        
    def _readCertificates(self):

        with open("./certs/ca-cert.pem", 'rb') as f1:
            caCert = f1.read()

        with open("./certs/client-cert.pem", 'rb') as f2:
            clientCert = f2.read()

        with open("./certs/client-key.pem", 'rb') as f3:
            clientKey = f3.read()
            
        return caCert, clientCert, clientKey
        

    def GetRequests(self, locations):

        # Get single locations
        print("[*] gRPC single requests:")

        return asyncio.get_event_loop().run_until_complete(self._getRequests(locations))
    

    async def _getRequests(self, locations):

        result = []

        async with grpc.aio.secure_channel(self.server, self.grpc_credentials) as channel:

            # Init the client stub
            stub = protos.satellite_pb2_grpc.SatelliteStub(channel)

            for loc in locations:
                try:
                    response = await self._get_img(stub, loc)
                    result.append((loc[0], loc[1], response.img))
                except grpc.RpcError as e:
                    status_code = e.code()
                    if grpc.StatusCode.OUT_OF_RANGE == status_code:
                        print("Bad request, out of bound location", loc)
                    elif grpc.StatusCode.PERMISSION_DENIED == status_code:
                        print("\n[Error] Permission denied", e.details())
                    elif grpc.StatusCode.DEADLINE_EXCEEDED == status_code:
                        print("\n[Error] Deadline exceeded, please reduce the server sleep_time")
                    else:
                        print(e)
                        print("Undefined error")
        return result
        

    def _get_img(self, stub: protos.satellite_pb2_grpc.SatelliteStub, location):

        loc = protos.satellite_pb2.Location()
        loc.x = location[0]
        loc.y = location[1]
    
        return stub.GetImage(loc, timeout=0.04, metadata=(("token", ('03357-1')),)) # 40 ms timeout
    

    def GetStream(self, queue, area):

        self.queue = queue

        # Get location (server) stream
        print("[*] gRPC server-stream request:")

        asyncio.get_event_loop().run_until_complete(self._getStream(area))
        

    async def _getStream(self, area):

        async with grpc.aio.secure_channel(self.server, self.grpc_credentials) as channel:

            # Init the client stub
            stub = protos.satellite_pb2_grpc.SatelliteStub(channel)

            try:
                await self._get_imgs(self.queue, stub, area[0], area[1])
            except grpc.RpcError as e:
                status_code = e.code()
                if grpc.StatusCode.OUT_OF_RANGE == status_code:
                    print("Bad request, out of bound location", area)
                elif grpc.StatusCode.DEADLINE_EXCEEDED == status_code:
                    print("\n[Error] Deadline exceeded, please reduce the server sleep_time")
                else:
                    print(e)
                    print("undefined error")
                    

    async def _get_imgs(self, queue, stub: protos.satellite_pb2_grpc.SatelliteStub, xy1, xy2):
    
        ll = protos.satellite_pb2.Location()
        ll.x = xy1[0]
        ll.y = xy1[1]
    
        ur = protos.satellite_pb2.Location()
        ur.x = xy2[0]
        ur.y = xy2[1]

        area = protos.satellite_pb2.Area()
        area.ll.CopyFrom(ll)
        area.ur.CopyFrom(ur)

        responses = stub.GetImages(area, timeout=10) # 10 s timeout
        async for response in responses:
            queue.put([response.x, response.y, response.img])

        queue.put(None)
