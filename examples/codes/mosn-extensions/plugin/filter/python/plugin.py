from concurrent import futures
import sys
import time

import argparse

import grpc

import logging

import json

import plugin_pb2
import plugin_pb2_grpc

class PluginServicer(plugin_pb2_grpc.PluginServicer):

    def Call(self, request, context):
            logging.info("begin do plugin something..")
            
            for item in checker.config:
                if request.header[item]!=checker.config[item]:
                    return plugin_pb2.Response(status=-1)
                
            return plugin_pb2.Response(status=1)

class Checker:
   def __init__(self,config):
       self.config=config
    
def serve(checker):
    # Start the server
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    #checker
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
    
    logging.basicConfig(filename='./plugin-py.log',format='[%(asctime)s-%(filename)s-%(levelname)s:%(message)s]', level = logging.DEBUG,filemode='a',datefmt='%Y-%m-%d%I:%M:%S %p')
    
    parser = argparse.ArgumentParser(description='Process some integers.')
    parser.add_argument('-c',dest="config",default="checkconf.json",
                    help='-c checkconf.json')
    args = parser.parse_args()
    
    f = open(args.config, encoding='utf-8')  
    setting = json.load(f)
    
    checker = Checker(setting)
    
    serve(checker)
