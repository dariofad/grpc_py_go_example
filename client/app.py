import os

from Proxies import SimpleProxy
from Maps import SimpleMap

from multiprocessing import Process, Queue

if __name__ == "__main__":

    """
    Proxy to query the gRPC server
    """

    server = '0.0.0.0:8000'
    proxy = SimpleProxy(server)
    

    """
    Unary RPCs
    """
    
    locations = {(1,2), (30,25), (50,4), (0,0), (50,16), (10, 1)}
    result = proxy.GetRequests(locations)

    for loc in result:
        print("\t(x: {}, y: {}): {}".format(loc[0], loc[1], loc[2]))    

    input("\n[I] Press <Enter> to continue...")

    os.system("clear")    


    """
    Server Streaming RPCs + Basic Map Rendering (terminal with monospace font)
    """

    # create an 80x32 rectangular map
    width = 80
    height = 32
    sMap = SimpleMap(width, height)


    # use the proxy to query the server and simultaneously print the map on the console
    queue = Queue()
    area = [(0,0), (79,31)] # area to download
    def Query(proxy, queue, area):
        proxy.GetStream(queue, area)
        
    query = Process(target=Query, args=(proxy, queue, area,))
    query.start()

    # rendering to console
    while True:
        
        data =  queue.get()
        # break if the stream was closed by the server
        if data is None:
            break
        else:
            sMap.SetLocation(*data)
            
    # keep showing the map
    input("\n[I] Press <Enter> to exit...")
