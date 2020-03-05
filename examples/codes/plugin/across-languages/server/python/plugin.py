from concurrent import futures
import sys
import time

import grpc

import plugin_pb2
import plugin_pb2_grpc

class PluginServicer(plugin_pb2_grpc.PluginServicer):

    def Call(self, request, context):
            return plugin_pb2.Response(body=bytes('hello python', 'ascii'))

def serve():
    # Start the server
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    plugin_pb2_grpc.add_PluginServicer_to_server(PluginServicer(), server)
    server.add_insecure_port('127.0.0.1:1234')
    server.start()

    # Output information
    print("1|1|tcp|127.0.0.1:1234|grpc")
    sys.stdout.flush()

    try:
        while True:
            time.sleep(60 * 60 * 24)
    except KeyboardInterrupt:
        server.stop(0)

if __name__ == '__main__':
    serve()
