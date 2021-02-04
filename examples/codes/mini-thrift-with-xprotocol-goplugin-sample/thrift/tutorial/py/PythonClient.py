#!/usr/bin/env python

#
# Licensed to the Apache Software Foundation (ASF) under one
# or more contributor license agreements. See the NOTICE file
# distributed with this work for additional information
# regarding copyright ownership. The ASF licenses this file
# to you under the Apache License, Version 2.0 (the
# "License"); you may not use this file except in compliance
# with the License. You may obtain a copy of the License at
#
#   http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied. See the License for the
# specific language governing permissions and limitations
# under the License.
#

import sys
import glob
import time

sys.path.append('../gen-py')

from tutorial import Calculator
from tutorial.ttypes import InvalidOperation, Operation, Work

from thrift import Thrift
from thrift.transport import TSocket
from thrift.transport import TTransport
from thrift.protocol import TBinaryProtocol

import argparse


def do_request(host, port, cont, interval):
    # Make socket

    print('================================================')
    print('sending request to {}:{}'.format(host, port))

    transport = TSocket.TSocket(host=host, port=port)

    # Buffering is critical. Raw sockets are very slow
    transport = TTransport.TBufferedTransport(transport)

    # Wrap in a protocol
    protocol = TBinaryProtocol.TBinaryProtocol(transport)

    # Create a client to use the protocol encoder
    client = Calculator.Client(protocol)

    # Connect!
    transport.open()

    # client.ping()
    # print('ping()')

    while True:
        sum_ = client.add(1, 1)
        print('1+1=%d' % sum_)

        work = Work()

        work.op = Operation.DIVIDE
        work.num1 = 1
        work.num2 = 0

        try:
            quotient = client.calculate(1, work)
            print('Whoa? You know how to divide by zero?')
            print('FYI the answer is %d' % quotient)
        except InvalidOperation as e:
            print('InvalidOperation: %r' % e)

        work.op = Operation.SUBTRACT
        work.num1 = 15
        work.num2 = 10

        diff = client.calculate(1, work)
        print('15-10=%d' % diff)

        log = client.getStruct(1)
        print('Check log: %s' % log.value)

        if cont:
            time.sleep(interval)
        else:
            break

    # Close!
    transport.close()


def do_request_cont(host, port, cont, interval):
    do_request(host, port, cont, interval)


def main(host, port, cont, interval, threads):
    do_request_cont(host, port, cont, interval)


if __name__ == '__main__':
    try:
        parser = argparse.ArgumentParser(description='Process some integers.')
        parser.add_argument('--host', type=str, default='127.0.0.1',
                            help='thrift tutorial server host')
        parser.add_argument('--port', type=int, default=3399,
                            help='thrift tutorial server port')
        parser.add_argument('--cont', action='store_true',
                            help='keep sending requests, none stop')
        parser.add_argument('--interval', type=float, default=1.0,
                            help='interval when keep sending requests')
        parser.add_argument('--threads', type=int, default=1,
                            help='parallel threads to send requests')

        args = parser.parse_args()

        main(args.host, args.port, args.cont, args.interval, args.threads)
    except Thrift.TException as tx:
        print('%s' % tx.message)
